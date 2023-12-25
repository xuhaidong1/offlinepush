package component

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/xuhaidong1/go-generic-tools/pluginsx/saramax"
	"github.com/xuhaidong1/offlinepush/internal/prometheus"
	"github.com/xuhaidong1/offlinepush/internal/repository/dao"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/pkg/registry"

	"github.com/IBM/sarama"
	"github.com/xuhaidong1/go-generic-tools/container/slice"
	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"github.com/xuhaidong1/offlinepush/internal/repository"
	"golang.org/x/sync/errgroup"
)

type Producer struct {
	// cron 写 producer读
	// notifyProducer <-chan pushconfig.PushConfig
	repo             repository.Repository
	producerClient   sarama.SyncProducer
	responsibleTypes []string
	hasher           *ConsistentHash
	cron             Croner
	counter          *Counter
	rg               registry.Registry
	// interceptor *interceptor.Interceptor
	// CancelFuncs *sync.Map
	logger logx.Logger
}

func NewProducer(repo repository.Repository, producerClient sarama.SyncProducer,
	hasher *ConsistentHash, cron Croner, counter *Counter, rg registry.Registry,
) *Producer {
	p := &Producer{
		repo:           repo,
		producerClient: producerClient,
		hasher:         hasher,
		cron:           cron,
		counter:        counter,
		rg:             rg,
		logger:         ioc.Loggerx.With(logx.Field{Key: "component", Value: "Producer"}),
	}
	p.responsibleTypes = hasher.GetResponsibleKeys()
	return p
}

func (p *Producer) Run(ctx context.Context) {
	p.watchResponsibleTypeCh(ctx, p.hasher)
	p.watchTaskCh(ctx, p.cron)
}

func (p *Producer) watchResponsibleTypeCh(ctx context.Context, hasher *ConsistentHash) {
	ch := hasher.Subscribe(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case types := <-ch:
				p.responsibleTypes = types
			}
		}
	}()
}

func (p *Producer) watchTaskCh(ctx context.Context, cron Croner) {
	ch := cron.Subscribe(ctx)
	manualCh, err := p.rg.Subscribe(config.StartConfig.Register.ManualTaskKey)
	if err != nil {
		p.logger.Error("subscribe manual taskkey err", logx.Error(err))
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case cfg := <-ch:
				go p.doTask(ctx, cfg)
			case evt := <-manualCh:
				var cfg pushconfig.PushConfig
				er := json.Unmarshal([]byte(evt.Instance.Note), &cfg)
				if er != nil {
					p.logger.Error("unmarshal manual task err", logx.Error(er))
					continue
				}
				go p.doTask(ctx, cfg)
			}
		}
	}()
}

func (p *Producer) doTask(ctx context.Context, cfg pushconfig.PushConfig) {
	//produceCtx, cancel := context.WithCancel(ctx)
	//_, loaded := p.CancelFuncs.LoadOrStore(cfg.Topic.Name, cancel)
	//// load到说明在生产了，不需要再另开goroutine生产
	//if loaded {
	//	return
	//}
	start := time.Now()
	produceTypes := slice.InterSet[string](p.responsibleTypes, cfg.DeviceTypeList)
	eg := &errgroup.Group{}
	for _, typ := range produceTypes {
		t := typ
		eg.Go(func() error {
			return p.ReadAndSend(ctx, cfg.Topic, t, cfg.Num)
		})
	}
	if err := eg.Wait(); err != nil {
		p.logger.Error("producer pool send失败", logx.Error(err))
	} else {
		p.counter.AddHopeCount(int64(len(produceTypes) * cfg.Num))
		p.logger.Info("完成生产", logx.String("topic", cfg.Topic.Name),
			logx.String("生产用时", time.Since(start).String()),
			logx.Int64("消息数量", int64(len(produceTypes)*cfg.Num)))
	}

	// p.CancelFuncs.Delete(cfg.Topic.Name)
	// cancel()
}

type ProducerPoolArgs struct {
	Topic     domain.Topic
	DeviceTyp string
	Limit     int
	Wg        *sync.WaitGroup
}

func (p *Producer) ReadAndSend(ctx context.Context, topic domain.Topic, deviceTyp string, num int) error {
	const batchSize = 100
	i := 0
	cursor := &dao.Cursor{}
	for i < num {
		limit := batchSize
		if (num - i) < batchSize {
			limit = num - i
		}
		devices, err := p.repo.FindDevicesPage(ctx, deviceTyp, limit, cursor)
		if err != nil {
			p.logger.Error("查询设备列表失败", logx.Error(err))
			return err
		}
		msgs := make([]*sarama.ProducerMessage, 0, batchSize)
		for j := 0; j < len(devices); j++ {
			msgs = append(msgs, &sarama.ProducerMessage{
				Topic: topic.Name,
				Value: saramax.JSONEncoder{Data: domain.Message{
					Topic:  topic,
					Device: devices[j],
				}},
			})
		}
		err = p.producerClient.SendMessages(msgs)
		if err != nil {
			p.logger.Error("发送消息到kafka失败", logx.Error(err))
			return err
		}
		prometheus.MessageGauge.WithLabelValues(topic.Name).Add(float64(len(msgs)))
		i += batchSize
	}
	return nil
}

//func (p *ProduceController) WatchTask(ctx context.Context, Wg *sync.WaitGroup) {
//	Wg.Done()
//	watchCtx, cancel := context.WithCancel(ctx)
//	_, loaded := p.CancelFuncs.LoadOrStore("WatchTask", cancel)
//	// load到说明有问题，不是生产者却开始WatchLeftTask
//	if loaded {
//		p.logger.Warn("WatchLeftTask", zap.String("LoadOrStore CancelFunc", "err"))
//		return
//	}
//	p.logger.Info("ProduceController", zap.String("WatchTask", "start"))
//	for {
//		select {
//		case <-watchCtx.Done():
//			p.logger.Info("ProduceController", zap.String("WatchTask", "canceled"))
//			return
//		case cfg := <-p.notifyProducer:
//			if atomic.LoadInt32(&p.isEngaged) == int32(1) {
//				if !p.interceptor.Permit(cfg.Topic.Name) {
//					p.logger.Info("ProduceController", zap.String(cfg.Topic.Name, "stopped"))
//					continue
//				}
//				go p.Assign(watchCtx, cfg)
//			}
//		}
//	}
//}

//func (p *ProduceController) WatchLeftTask(ctx context.Context, Wg *sync.WaitGroup, key string, interval time.Duration) {
//	Wg.Done()
//	watchCtx, cancel := context.WithCancel(ctx)
//	_, loaded := p.CancelFuncs.LoadOrStore("WatchLeftTask", cancel)
//	// load到说明有问题，不是生产者却开始WatchLeftTask
//	if loaded {
//		p.logger.Warn("WatchLeftTask", zap.String("LoadOrStore CancelFunc", "err"))
//		return
//	}
//	p.logger.Info("ProduceController", zap.String("WatchLeftTask", "start"))
//	// 每隔interval询问一次
//	ticker := time.NewTicker(interval)
//	for {
//		select {
//		case <-watchCtx.Done():
//			p.logger.Info("ProduceController", zap.String("WatchLeftTask", "canceled"))
//			return
//		case <-ticker.C:
//			if atomic.LoadInt32(&p.isEngaged) != int32(1) {
//				continue
//			}
//			cfg, err := p.repo.GetLeftTask(watchCtx)
//			if err != nil && !errors.Is(err, redis.Nil) {
//				p.logger.Error("ProduceController", zap.Error(err))
//			}
//			if errors.Is(err, redis.Nil) {
//				continue
//			}
//			if !p.interceptor.Permit(cfg.Topic.Name) {
//				p.logger.Info("ProduceController", zap.String(cfg.Topic.Name, "stopped"))
//				continue
//			}
//			go p.Assign(watchCtx, cfg)
//		}
//	}
//}

//func (p *ProduceController) CancelProduce() {
//	p.CancelFuncs.Range(func(key, value any) bool {
//		cancelAny, ok := p.CancelFuncs.LoadAndDelete(key)
//		if !ok {
//			p.logger.Error("ProduceController", zap.String("CancelProduce", "LoadAndDelete CancelFuncs failed"))
//		}
//		cancel := cancelAny.(context.CancelFunc)
//		cancel()
//		return true
//	})
//}
