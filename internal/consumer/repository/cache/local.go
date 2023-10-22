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
	RPush(ctx context.Context, msg domain.Message)
	LPop(ctx context.Context, biz string) (domain.Message, error)
	Delete(key string)
}

var (
	ErrNoKey     = errs.NewErrKeyNotFound("没有key")
	ErrNoMessage = errors.New("没有消息")
)

type localCache struct {
	data  map[string]*queue.ConcurrentQueue[domain.Message]
	mutex *sync.Mutex
}

func NewLocalCache() LocalCache {
	return &localCache{
		data:  make(map[string]*queue.ConcurrentQueue[domain.Message]),
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
