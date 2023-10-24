package producer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config"
	"go.uber.org/zap"

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
	logger             *zap.Logger
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
		logger:      ioc.Logger,
	}
	wg := &sync.WaitGroup{}
	wg.Add(4)
	go p.ListenStartCond(ctx, wg)
	go p.ListenStopCond(ctx, wg)
	go p.WatchTask(ctx, wg)
	go p.WatchLeftTask(ctx, wg, config.StartConfig.Redis.ProducerLeftTaskKey, time.Second)
	wg.Wait()
	return p
}

func (p *ProduceController) ListenStartCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	p.logger.Info("ProduceController", zap.String("ListenStartCond", "start"))
	for {
		p.startCond.L.Lock()
		err := p.startCond.WaitWithTimeout(ctx)
		p.startCond.L.Unlock()
		if err != nil {
			p.logger.Info("ProduceController", zap.String("ListenStartCond", "closed"))
			return
		}
		ok := atomic.CompareAndSwapInt32(&p.isUse, int32(0), int32(1))
		p.logger.Info("ProduceController", zap.Bool("isProducer", true))
		if !ok {
			p.logger.Error("ProduceController", zap.String("ListenStartCond", "change status failed"))
		}
	}
}

// ListenStopCond 这边收到了停止信号，是不知道什么原因让停止的 /没拿到锁 应该传黑匣子/手动停止 应该传黑匣子/服务关闭 应该传黑匣子--统一了 不需要知道原因
func (p *ProduceController) ListenStopCond(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	p.logger.Info("ProduceController", zap.String("ListenStopCond", "start"))
	for {
		p.stopCond.L.Lock()
		err := p.stopCond.WaitWithTimeout(ctx)
		p.stopCond.L.Unlock()
		// 能出来说明收到了停止信号
		p.CancelProduce()
		atomic.StoreInt32(&p.isUse, int32(0))
		p.logger.Info("ProduceController", zap.Bool("isProducer", false))
		if err != nil {
			p.logger.Info("ProduceController", zap.String("ListenStopCond", "closed"))
			return
		}

	}
}

func (p *ProduceController) WatchTask(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	p.logger.Info("ProduceController", zap.String("WatchTask", "start"))
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("ProduceController", zap.String("WatchTask", "closed"))
			return
		case cfg := <-p.notifyProducer:
			if atomic.LoadInt32(&p.isUse) == int32(1) {
				go p.Assign(ctx, cfg)
			}
		}
	}
}

func (p *ProduceController) WatchLeftTask(ctx context.Context, wg *sync.WaitGroup, key string, interval time.Duration) {
	wg.Done()
	p.logger.Info("ProduceController", zap.String("WatchLeftTask", "start"))
	// 每隔interval询问一次
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("ProduceController", zap.String("WatchLeftTask", "closed"))
			return
		case <-ticker.C:
			if atomic.LoadInt32(&p.isUse) != int32(1) {
				continue
			}
			cfg, err := p.repo.GetLeftTask(ctx)
			if err != nil && !errors.Is(err, redis.Nil) {
				p.logger.Error("ProduceController", zap.Error(err))
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			go p.Assign(ctx, cfg)
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
			p.logger.Error("ProduceController", zap.String("CancelProduce", "LoadAndDelete CancelFuncs failed"))
		}
		cancel := cancelAny.(context.CancelFunc)
		cancel()
		return true
	})
}
