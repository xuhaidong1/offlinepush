package lock

import (
	"context"
	"errors"
	"time"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"go.uber.org/zap"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	redisLock "github.com/xuhaidong1/go-generic-tools/redis_lock"
	"github.com/xuhaidong1/go-generic-tools/redis_lock/errs"
	"github.com/xuhaidong1/offlinepush/config"
)

var (
	ErrRefreshFailed       = errors.New("续约失败")
	ErrFailedToPreemptLock = errs.ErrFailedToPreemptLock
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
	logger    *zap.Logger
}

func NewLockController(lockClient *redisLock.Client, podName string,
	startCond, stopCond *cond.CondAtomic, lc config.LockConfig,
) *LockController {
	return &LockController{
		lockClient: lockClient,
		podName:    podName,
		startCond:  startCond,
		stopCond:   stopCond,
		lockConfig: lc,
		logger:     ioc.Logger,
	}
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
			return ctx.Err()
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
			if err != nil && !errors.Is(err, ErrFailedToPreemptLock) {
				return nil, err
			}
			if errors.Is(err, ErrFailedToPreemptLock) {
				continue
			}
			return l, nil
		case <-ctx.Done():
			// 生产者主动退出
			return nil, context.Canceled
		}
	}
}

func (m *LockController) Run(ctx context.Context) {
	m.logger.Info("LockController", zap.String("status", "start"))
	go func() {
		for {
			m.logger.Info("LockController", zap.String("status", "Apply"))
			lock, err := m.Apply(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				m.logger.Info("LockController", zap.String("Run", "Apply"), zap.Error(err))
			}
			if errors.Is(err, context.Canceled) {
				return
			}
			m.l = lock
			m.startCond.L.Lock()
			m.startCond.Broadcast()
			m.startCond.L.Unlock()
			err = m.AutoRefresh(ctx, lock, time.Millisecond*300)
			// 续约失败/手动停止
			if err != nil {
				if errors.Is(err, ErrRefreshFailed) {
					m.logger.Warn("LockController", zap.String("refresh", "failed"), zap.Error(err))
				}
				m.stopCond.L.Lock()
				m.stopCond.Broadcast()
				m.stopCond.L.Unlock()
				_ = m.l.Unlock(context.WithoutCancel(ctx))
				m.l = nil
			}
			if errors.Is(err, context.Canceled) {
				m.logger.Info("LockController", zap.String("status", "closed"))
				return
			}
		}
	}()
}

// Stop 手动停止抢锁，一般是在服务关闭时调用
func (m *LockController) Stop() {
	if m.l != nil {
		_ = m.l.Unlock(context.Background())
	}
}
