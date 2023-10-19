package loadbalancer

import (
	"context"
	"log"
	"sync"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
)

// LoadBalanceController 主要是对消费者的负载均衡
type LoadBalanceController struct {
	startCond *cond.CondAtomic
	stopCond  *cond.CondAtomic
	worker    *LoadBalancer
}

func NewLoadBalanceController(ctx context.Context, start, stop *cond.CondAtomic) *LoadBalanceController {
	l := &LoadBalanceController{startCond: start, stopCond: stop}
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
			log.Println("loadbalancer listenStartCond closing")
			return
		}
		if l.worker == nil {
			l.worker = NewLoadBalancer(ctx)
		}
	}
}

func (l *LoadBalanceController) ListenStopCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	for {
		l.stopCond.L.Lock()
		err := l.stopCond.WaitWithTimeout(ctx)
		l.stopCond.L.Unlock()
		if err != nil {
			log.Println("loadbalancer listenStopCond closing")
			return
		}
		if l.worker != nil {
			l.worker.Stop()
			l.worker = nil
		}
	}
}
