package loadbalancer

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/pkg/registry"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
)

// LoadBalanceController 主要是对消费者的负载均衡
type LoadBalanceController struct {
	// producer写 loadbalancer读
	notifyChan   <-chan pushconfig.PushConfig
	startCond    *cond.CondAtomic
	stopCond     *cond.CondAtomic
	loadBalancer *LoadBalancer // 一个loadbalancer就够了
	isUse        int32
	cancel       context.CancelFunc
	registry     registry.Registry
}

func NewLoadBalanceController(ctx context.Context, notifyChan <-chan pushconfig.PushConfig, start, stop *cond.CondAtomic, registry registry.Registry) *LoadBalanceController {
	l := &LoadBalanceController{
		notifyChan:   notifyChan,
		startCond:    start,
		stopCond:     stop,
		loadBalancer: NewLoadBalancer(registry),
		isUse:        int32(0),
		registry:     registry,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go l.ListenStartCond(ctx, wg)
	go l.ListenStopCond(ctx, wg)
	wg.Wait()
	return l
}

func (l *LoadBalanceController) ListenStartCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	for {
		l.startCond.L.Lock()
		err := l.startCond.WaitWithTimeout(ctx)
		l.startCond.L.Unlock()
		if err != nil {
			log.Println("LoadBalanceController listenStartCond closing")
			return
		}
		ok := atomic.CompareAndSwapInt32(&l.isUse, int32(0), int32(1))
		if !ok {
			log.Fatalln("LoadBalanceController start fail")
		}
		go l.WatchNotifyChan(ctx)
	}
}

func (l *LoadBalanceController) ListenStopCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	for {
		l.stopCond.L.Lock()
		err := l.stopCond.WaitWithTimeout(ctx)
		l.stopCond.L.Unlock()
		if err != nil {
			log.Println("LoadBalanceController listenStopCond closing")
			l.CancelLoadBalance()
			return
		}
		ok := atomic.CompareAndSwapInt32(&l.isUse, int32(1), int32(0))
		if !ok {
			log.Fatalln("LoadBalanceController stop fail")
		}
		l.CancelLoadBalance()
	}
}

func (l *LoadBalanceController) WatchNotifyChan(ctx context.Context) {
	watchCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	for {
		select {
		case <-watchCtx.Done():
			log.Println("LoadBalanceController WatchNotifyChan closing")
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
