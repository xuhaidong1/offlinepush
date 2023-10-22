package producer

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
)

type ProduceController struct {
	// cron 写 producer读
	notifyProducer     <-chan pushconfig.PushConfig
	notifyLoadBalancer chan<- pushconfig.PushConfig
	startCond          *cond.CondAtomic
	stopCond           *cond.CondAtomic
	producers          *sync.Pool
	repo               repository.ProducerRepository
	isUse              int32
	CancelFuncs        *sync.Map
}

func NewProduceController(ctx context.Context, notifyProducer <-chan pushconfig.PushConfig, notifyLoadBalancer chan<- pushconfig.PushConfig,
	start, stop *cond.CondAtomic, repo repository.ProducerRepository,
) *ProduceController {
	p := &ProduceController{
		notifyProducer:     notifyProducer,
		notifyLoadBalancer: notifyLoadBalancer,
		startCond:          start,
		stopCond:           stop,
		producers: &sync.Pool{New: func() any {
			return NewProducer(repo)
		}},
		repo:        repo,
		isUse:       int32(0),
		CancelFuncs: &sync.Map{},
	}
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go p.ListenStartCond(ctx, wg)
	go p.ListenStopCond(ctx, wg)
	go p.WatchTask(ctx, wg)
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
		ok := atomic.CompareAndSwapInt32(&p.isUse, int32(0), int32(1))
		log.Println("成为了生产者")
		if !ok {
			log.Fatalln("ProduceController start fail")
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
			p.CancelProduce()
			return
		}
		ok := atomic.CompareAndSwapInt32(&p.isUse, int32(1), int32(0))
		if !ok {
			log.Fatalln("ProduceController stop fail")
		}
		p.CancelProduce()
	}
}

func (p *ProduceController) WatchTask(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Println("ProduceController WatchProduceChan closing")
			return
		case cfg := <-p.notifyProducer:
			if atomic.LoadInt32(&p.isUse) == int32(1) {
				go p.Assign(ctx, cfg)
				log.Println("produce ok")
			}
		}
	}
}

func (p *ProduceController) Assign(ctx context.Context, cfg pushconfig.PushConfig) {
	produceCtx, cancel := context.WithCancel(ctx)
	_, loaded := p.CancelFuncs.LoadOrStore(cfg.Business.Name, cancel)
	if loaded {
		return
	}
	producer := p.producers.Get().(*Producer)
	producer.Produce(produceCtx, cfg)
	p.producers.Put(producer)
	p.CancelFuncs.Delete(cfg.Business.Name)
	cancel()
	p.notifyLoadBalancer <- cfg
}

func (p *ProduceController) CancelProduce() {
	p.CancelFuncs.Range(func(key, value any) bool {
		cancelAny, ok := p.CancelFuncs.LoadAndDelete(key)
		if !ok {
			log.Fatalln("ProduceController CancelProduce load cancelMap fail")
		}
		cancel := cancelAny.(context.CancelFunc)
		cancel()
		return true
	})
}
