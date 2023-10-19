package loadbalancer

import (
	"context"
	"log"
	"sync"
)

type LoadBalancer struct {
	cancel context.CancelFunc
}

func NewLoadBalancer(ctx context.Context) *LoadBalancer {
	workerCtx, cancel := context.WithCancel(ctx)
	w := &LoadBalancer{
		cancel: cancel,
	}
	go w.Work(workerCtx)
	return w
}

func (w *LoadBalancer) Work(ctx context.Context) {
	once := sync.Once{}
	for {
		select {
		case <-ctx.Done():
			log.Println("loadbalancer work结束")
			return
		default:
			once.Do(func() {
				log.Println("loadbalancer working")
			})
		}
	}
}

func (w *LoadBalancer) Stop() {
	w.cancel()
}
