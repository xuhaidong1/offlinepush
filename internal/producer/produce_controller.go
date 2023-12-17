package producer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xuhaidong1/offlinepush/internal/interceptor"

	"github.com/xuhaidong1/offlinepush/config"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"go.uber.org/zap"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
)

type ProduceController struct {
	// cron 写 producer读
	notifyProducer     <-chan pushconfig.PushConfig
	notifyLoadBalancer chan<- pushconfig.PushConfig
	// 成为/卸任生产者的cond，由分布式锁决定是，用户无需关心
	engageCond  *cond.CondAtomic
	dismissCond *cond.CondAtomic
	producers   *sync.Pool
	repo        repository.ProducerRepository
	interceptor *interceptor.Interceptor
	isEngaged   int32
	CancelFuncs *sync.Map
	logger      *zap.Logger
}

func NewProduceController(ctx context.Context, notifyProducer <-chan pushconfig.PushConfig,
	notifyLoadBalancer chan<- pushconfig.PushConfig,
	start, stop *cond.CondAtomic, repo repository.ProducerRepository,
	interceptor *interceptor.Interceptor,
) *ProduceController {
	p := &ProduceController{
		notifyProducer:     notifyProducer,
		notifyLoadBalancer: notifyLoadBalancer,
		engageCond:         start,
		dismissCond:        stop,
		producers: &sync.Pool{New: func() any {
			return NewProducer(repo)
		}},
		repo:        repo,
		interceptor: interceptor,
		isEngaged:   int32(0),
		CancelFuncs: &sync.Map{},
		logger:      ioc.Logger,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.ListenEngageCond(ctx, wg)
	go p.ListenDismissCond(ctx, wg)
	wg.Wait()
	return p
}

func (p *ProduceController) ListenEngageCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	p.logger.Info("ProduceController", zap.String("ListenEngageCond", "start"))
	for {
		p.engageCond.L.Lock()
		err := p.engageCond.WaitWithTimeout(ctx)
		p.engageCond.L.Unlock()
		if err != nil {
			p.logger.Info("ProduceController", zap.String("ListenEngageCond", "closed"))
			return
		}
		ok := atomic.CompareAndSwapInt32(&p.isEngaged, int32(0), int32(1))
		p.logger.Info("ProduceController", zap.Bool("isProducer", true))
		if !ok {
			p.logger.Warn("ProduceController", zap.String("ListenEngageCond", "change status failed"))
		}
		wgg := &sync.WaitGroup{}
		wgg.Add(2)
		go p.WatchTask(ctx, wgg)
		go p.WatchLeftTask(ctx, wgg, config.StartConfig.Redis.ProducerLeftTaskKey, time.Second)
		wgg.Wait()
	}
}

// ListenDismissCond 这边收到了停止信号，是不知道什么原因让停止的 /没拿到锁 应该传黑匣子/手动停止 应该传黑匣子/服务关闭 应该传黑匣子--统一了 不需要知道原因
func (p *ProduceController) ListenDismissCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	p.logger.Info("ProduceController", zap.String("ListenDismissCond", "start"))
	for {
		p.dismissCond.L.Lock()
		err := p.dismissCond.WaitWithTimeout(ctx)
		p.dismissCond.L.Unlock()
		// 能出来说明收到了停止信号
		p.CancelProduce()
		atomic.StoreInt32(&p.isEngaged, int32(0))
		p.logger.Info("ProduceController", zap.Bool("isProducer", false))
		if err != nil {
			p.logger.Info("ProduceController", zap.String("ListenDismissCond", "closed"))
			return
		}

	}
}

func (p *ProduceController) WatchTask(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	watchCtx, cancel := context.WithCancel(ctx)
	_, loaded := p.CancelFuncs.LoadOrStore("WatchTask", cancel)
	// load到说明有问题，不是生产者却开始WatchLeftTask
	if loaded {
		p.logger.Warn("WatchLeftTask", zap.String("LoadOrStore CancelFunc", "err"))
		return
	}
	p.logger.Info("ProduceController", zap.String("WatchTask", "start"))
	for {
		select {
		case <-watchCtx.Done():
			p.logger.Info("ProduceController", zap.String("WatchTask", "canceled"))
			return
		case cfg := <-p.notifyProducer:
			if atomic.LoadInt32(&p.isEngaged) == int32(1) {
				if !p.interceptor.Permit(cfg.Topic.Name) {
					p.logger.Info("ProduceController", zap.String(cfg.Topic.Name, "stopped"))
					continue
				}
				go p.Assign(watchCtx, cfg)
			}
		}
	}
}

func (p *ProduceController) WatchLeftTask(ctx context.Context, wg *sync.WaitGroup, key string, interval time.Duration) {
	wg.Done()
	watchCtx, cancel := context.WithCancel(ctx)
	_, loaded := p.CancelFuncs.LoadOrStore("WatchLeftTask", cancel)
	// load到说明有问题，不是生产者却开始WatchLeftTask
	if loaded {
		p.logger.Warn("WatchLeftTask", zap.String("LoadOrStore CancelFunc", "err"))
		return
	}
	p.logger.Info("ProduceController", zap.String("WatchLeftTask", "start"))
	// 每隔interval询问一次
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-watchCtx.Done():
			p.logger.Info("ProduceController", zap.String("WatchLeftTask", "canceled"))
			return
		case <-ticker.C:
			if atomic.LoadInt32(&p.isEngaged) != int32(1) {
				continue
			}
			cfg, err := p.repo.GetLeftTask(watchCtx)
			if err != nil && !errors.Is(err, redis.Nil) {
				p.logger.Error("ProduceController", zap.Error(err))
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			if !p.interceptor.Permit(cfg.Topic.Name) {
				p.logger.Info("ProduceController", zap.String(cfg.Topic.Name, "stopped"))
				continue
			}
			go p.Assign(watchCtx, cfg)
		}
	}
}

func (p *ProduceController) Assign(ctx context.Context, cfg pushconfig.PushConfig) {
	produceCtx, cancel := context.WithCancel(ctx)
	_, loaded := p.CancelFuncs.LoadOrStore(cfg.Topic.Name, cancel)
	// load到说明在生产了，不需要再另开goroutine生产
	if loaded {
		return
	}
	producer := p.producers.Get().(*Producer)
	producer.Produce(produceCtx, cfg)
	p.producers.Put(producer)
	p.CancelFuncs.Delete(cfg.Topic.Name)
	cancel()
	p.notifyLoadBalancer <- cfg
}

func (p *ProduceController) CancelProduce() {
	p.CancelFuncs.Range(func(key, value any) bool {
		cancelAny, ok := p.CancelFuncs.LoadAndDelete(key)
		if !ok {
			p.logger.Error("ProduceController", zap.String("CancelProduce", "LoadAndDelete CancelFuncs failed"))
		}
		cancel := cancelAny.(context.CancelFunc)
		cancel()
		return true
	})
}
