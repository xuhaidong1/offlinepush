package producer

import (
	"context"
	"log"
	"sync"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
)

type ProduceController struct {
	startCond *cond.CondAtomic
	stopCond  *cond.CondAtomic
	producer  *Producer
	repo      repository.ProducerRepository
}

func NewProduceController(ctx context.Context, start, stop *cond.CondAtomic, repo repository.ProducerRepository) *ProduceController {
	p := &ProduceController{
		startCond: start,
		stopCond:  stop,
		repo:      repo,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.ListenStartCond(ctx, wg)
	go p.ListenStopCond(ctx, wg)
	wg.Wait()
	return p
}

func (p *ProduceController) ListenStartCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	for {
		p.startCond.L.Lock()
		err := p.startCond.WaitWithTimeout(ctx)
		p.startCond.L.Unlock()
		if err != nil {
			log.Println("producer listenStartCond closing")
			return
		}
		if p.producer == nil {
			p.producer = NewProducer(ctx, p.repo)
		}
	}
}

// ListenStopCond 这边收到了停止信号，是不知道什么原因让停止的 /没拿到锁 应该传黑匣子/手动停止 应该传黑匣子/服务关闭 应该传黑匣子--统一了 不需要知道原因
func (p *ProduceController) ListenStopCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	for {
		p.stopCond.L.Lock()
		err := p.stopCond.WaitWithTimeout(ctx)
		p.stopCond.L.Unlock()
		if err != nil {
			log.Println("producer listenStopCond closing")
			return
		}
		if p.producer != nil {
			p.producer.Stop()
			p.producer = nil
		}
	}
}
