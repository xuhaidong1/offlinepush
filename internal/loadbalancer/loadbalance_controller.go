package loadbalancer

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"go.uber.org/zap"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/pkg/registry"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
)

// LoadBalanceController 主要是对消费者的负载均衡
type LoadBalanceController struct {
	// producer写 loadbalancer读
	notifyChan   <-chan pushconfig.PushConfig
	engageCond   *cond.CondAtomic
	dismissCond  *cond.CondAtomic
	loadBalancer *LoadBalancer // 一个loadbalancer就够了
	isUse        int32
	cancel       context.CancelFunc
	registry     registry.Registry
	logger       *zap.Logger
}

func NewLoadBalanceController(ctx context.Context, notifyChan <-chan pushconfig.PushConfig, start, stop *cond.CondAtomic, registry registry.Registry) *LoadBalanceController {
	l := &LoadBalanceController{
		notifyChan:   notifyChan,
		engageCond:   start,
		dismissCond:  stop,
		loadBalancer: NewLoadBalancer(registry),
		isUse:        int32(0),
		registry:     registry,
		logger:       ioc.Logger,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go l.ListenEngageCond(ctx, wg)
	go l.ListenDismissCond(ctx, wg)
	wg.Wait()
	return l
}

func (l *LoadBalanceController) ListenEngageCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	l.logger.Info("LoadBalanceController", zap.String("ListenEngageCond", "start"))
	for {
		l.engageCond.L.Lock()
		err := l.engageCond.WaitWithTimeout(ctx)
		l.engageCond.L.Unlock()
		if err != nil {
			l.logger.Info("LoadBalanceController", zap.String("ListenEngageCond", "closed"))
			return
		}
		ok := atomic.CompareAndSwapInt32(&l.isUse, int32(0), int32(1))
		if !ok {
			l.logger.Error("LoadBalanceController", zap.String("ListenEngageCond", "change status failed"))
		}
		l.logger.Info("LoadBalanceController", zap.Bool("isLoadbalancer", true))
		go l.WatchNotifyChan(ctx)
	}
}

func (l *LoadBalanceController) ListenDismissCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	l.logger.Info("LoadBalanceController", zap.String("ListenDismissCond", "start"))
	for {
		l.dismissCond.L.Lock()
		err := l.dismissCond.WaitWithTimeout(ctx)
		l.dismissCond.L.Unlock()
		// 能出来说明收到了停止信号
		l.CancelLoadBalance()
		atomic.StoreInt32(&l.isUse, int32(0))
		l.logger.Info("LoadBalanceController", zap.Bool("isLoadbalancer", false))
		if err != nil {
			l.logger.Info("LoadBalanceController", zap.String("ListenDismissCond", "closed"))
			return
		}
	}
}

func (l *LoadBalanceController) WatchNotifyChan(ctx context.Context) {
	l.logger.Info("LoadBalanceController", zap.String("WatchNotifyChan", "start"))
	watchCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	for {
		select {
		case <-watchCtx.Done():
			l.logger.Info("LoadBalanceController", zap.String("WatchNotifyChan", "canceled"))
			return
		case cfg := <-l.notifyChan:
			if atomic.LoadInt32(&l.isUse) == int32(1) {
				l.loadBalancer.SelectConsumer(cfg)
			}
		}
	}
}

func (l *LoadBalanceController) CancelLoadBalance() {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
}
