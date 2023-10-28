package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/xuhaidong1/go-generic-tools/cache/errs"
	queue "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type LocalCache interface {
	// RPush 并发队列 生产者用
	RPush(ctx context.Context, msg domain.Message)
	LPop(ctx context.Context, biz string) (domain.Message, error)
	// BRPush 并发阻塞队列 多个消费者一起用
	BRPush(ctx context.Context, msg domain.Message) error
	BLPop(ctx context.Context, biz string) (domain.Message, error)
	Delete(key string)
	// LPopAll 出队消费者队列的所有元素，在cancel写回存储介质的时候用
	LPopAll(ctx context.Context, biz string) ([]domain.Message, error)
}

var (
	ErrNoKey     = errs.NewErrKeyNotFound("没有key")
	ErrNoMessage = errors.New("没有消息")
)

type localCache struct {
	data  map[string]*queue.ConcurrentQueue[domain.Message]
	bData map[string]*queue.ConcurrentBlockingQueue[domain.Message]
	mutex *sync.Mutex
}

func NewLocalCache() LocalCache {
	return &localCache{
		data:  make(map[string]*queue.ConcurrentQueue[domain.Message]),
		bData: make(map[string]*queue.ConcurrentBlockingQueue[domain.Message]),
		mutex: &sync.Mutex{},
	}
}

func (l *localCache) RPush(ctx context.Context, msg domain.Message) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	q, ok := l.data[msg.Business.Name]
	if !ok {
		newQ := queue.NewConcurrentQueue[domain.Message](1024)
		_ = newQ.Enqueue(ctx, msg)
		l.data[msg.Business.Name] = newQ
		return
	}
	_ = q.Enqueue(ctx, msg)
	l.data[msg.Business.Name] = q
	return
}

func (l *localCache) LPop(ctx context.Context, biz string) (domain.Message, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	q, ok := l.data[biz]
	if !ok {
		return domain.Message{}, ErrNoKey
	}
	msg, err := q.Dequeue(ctx)
	if err != nil && !errors.Is(err, queue.ErrQueueEmpty) {
		return domain.Message{}, err
	}
	if errors.Is(err, queue.ErrQueueEmpty) {
		return domain.Message{}, ErrNoMessage
	}
	l.data[biz] = q
	return msg, nil
}

func (l *localCache) Delete(key string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.data, key)
}

func (l *localCache) BRPush(ctx context.Context, msg domain.Message) error {
	l.mutex.Lock()
	q, ok := l.bData[msg.Business.Name]
	if !ok {
		newQ := queue.NewConcurrentBlockingQueue[domain.Message](10240)
		_ = newQ.Enqueue(ctx, msg)
		l.bData[msg.Business.Name] = newQ
		l.mutex.Unlock()
		return nil
	}
	l.mutex.Unlock()
	return q.Enqueue(ctx, msg)
}

func (l *localCache) BLPop(ctx context.Context, biz string) (domain.Message, error) {
	l.mutex.Lock()
	q, ok := l.bData[biz]
	l.mutex.Unlock()
	if !ok {
		return domain.Message{}, ErrNoKey
	}
	return q.Dequeue(ctx)
}

func (l *localCache) LPopAll(ctx context.Context, biz string) (msgs []domain.Message, err error) {
	l.mutex.Lock()
	q, ok := l.bData[biz]
	l.mutex.Unlock()
	if !ok {
		return nil, ErrNoKey
	}
	for !q.IsEmpty() {
		// EOF不一定被写到了本地缓存中，所以不能用EOF判断是否完全出队完毕
		// EOF被写到了本地缓存中，拿到了EOF则队列一定为空
		// 队列为空不一定最后一个是EOF，队列中很可能存的是中间某片数据
		res, er := q.Dequeue(ctx)
		if er != nil {
			return nil, er
		}
		if !domain.IsEOF(res) {
			msgs = append(msgs, res)
		}
	}
	return msgs, nil
}
