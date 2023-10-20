package lock

import (
	"context"
	"errors"
	"log"
	"time"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	redisLock "github.com/xuhaidong1/go-generic-tools/redis_lock"
	"github.com/xuhaidong1/go-generic-tools/redis_lock/errs"
	"github.com/xuhaidong1/offlinepush/config"
)

var (
	ErrRefreshFailed = errors.New("续约失败")
	ErrLockNotHold   = errs.ErrLockNotHold
)

type LockController struct {
	lockClient *redisLock.Client
	lockConfig config.LockConfig
	// 服务实例名
	podName string
	// 申请到的锁
	l *redisLock.Lock
	// 根据有没有锁/推送开关开没开 控制业务（produceworker，loadbalanceworker）的启动停止
	startCond *cond.CondAtomic
	stopCond  *cond.CondAtomic
}

func NewLockController(lockClient *redisLock.Client, podName string, startCond, stopCond *cond.CondAtomic, lc config.LockConfig) *LockController {
	return &LockController{lockClient: lockClient, podName: podName, startCond: startCond, stopCond: stopCond, lockConfig: lc}
}

// AutoRefresh 需要在外面开启goroutine
func (m *LockController) AutoRefresh(ctx context.Context, l *redisLock.Lock, timeout time.Duration) error {
	ticker := time.NewTicker(m.lockConfig.RefreshInterval)
	// 不断续约 直到收到退出信号
	defer ticker.Stop()
	// 重试次数计数
	retryCnt := 0
	// 超时重试控制信号
	retryCh := make(chan struct{}, 1)
	for {
		select {
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(ctx, timeout)
			err := l.Refresh(refreshCtx)
			cancel()
			if err == context.DeadlineExceeded {
				retryCh <- struct{}{}
				continue
			}
			if err != nil {
				// 不可挽回的错误 要考虑中断业务执行，应该先中断生产，后解锁，
				// 如果先解锁，其它实例成为生产者，我们还没中断生产，生产进度还没写回到redis，新生产者就开始生产了，会导致生产两份相同的消息
				// 应该先中断生产，生产黑匣子写回redis，解锁
				// 生产写回redis&派任务时收到停止生产信号，应该无视，下一轮发现ctx没锁了，就退出了
				// 新生产者上任：检查上一个生产者的黑匣子（一个指定key：producer_black_box）（记录了意外中断时在生产但还没有写回redis的business名字）（应该只有1个），生产日期time），
				// 把这些黑匣子里的内容都重新生产一遍。
				// 生产者正常写回缓存时应该一股脑写到redis，但负载均衡应该按照本地缓存里面的分片的消息一一分配，完成一个business的生产-写redis-对本地缓存做负载均衡-一片数据负载均衡一次-完成分配就清除本地缓存
				return ErrRefreshFailed
			}
			retryCnt = 0
		case <-retryCh:
			retryCnt++
			refreshCtx, cancel := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(refreshCtx)
			cancel()
			if err == context.DeadlineExceeded {
				// 可以超时重试
				if retryCnt > 3 {
					return ErrRefreshFailed
				}
				// 如果retryCh容量不是1，在这会阻塞
				retryCh <- struct{}{}
				continue
			}
			if err != nil {
				// 不可挽回的错误 要考虑中断业务执行
				return ErrRefreshFailed
			}
			retryCnt = 0
		case <-ctx.Done():
			// 生产者主动退出
			if ctx.Err() == context.Canceled {
				return nil
			}
			return nil
		}
	}
}

func (m *LockController) Apply(ctx context.Context) (*redisLock.Lock, error) {
	// ctx 申请producer用的ctx，可以级联取消
	// 每隔一秒申请一次 直到收到退出信号
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l, err := m.lockClient.TryLock(ctx, m.lockConfig.LockKey, m.podName, m.lockConfig.Expiration)
			// 抢锁失败
			if err != nil && !errors.Is(err, ErrLockNotHold) {
				return nil, err
			}
			if errors.Is(err, ErrLockNotHold) {
				continue
			}
			return l, nil
		case <-ctx.Done():
			// 生产者主动退出
			return nil, context.Canceled
		}
	}
}

// Start 需要开启goroutine调用这个方法
func (m *LockController) Start(ctx context.Context) {
	lock, err := m.Apply(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	m.l = lock
	m.startCond.L.Lock()
	m.startCond.Broadcast()
	m.startCond.L.Unlock()
	err = m.AutoRefresh(ctx, lock, time.Millisecond*300)
	// todo 错误处理细化: 丢锁，手动停止，服务关闭
	if errors.Is(err, ErrRefreshFailed) {
		// log.Println("生产者续约失败，需要中断生产")
		m.stopCond.L.Lock()
		m.stopCond.Broadcast()
		m.stopCond.L.Unlock()
	}
	err = m.l.Unlock(context.WithoutCancel(ctx))
	m.l = nil
	if err != nil {
		panic(err)
	}
	log.Println("lock controller closed")
	return
}

// Stop 手动停止抢锁，一般是在服务关闭时调用
func (m *LockController) Stop() {
	if m.l != nil {
		_ = m.l.Unlock(context.Background())
	}
}
