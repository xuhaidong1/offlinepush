package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
)

type ConsumeController struct {
	// 注册中心写 consumeController读出任务来
	notifyChan <-chan registry.Event
	consumers  *sync.Pool
	repo       repository.ConsumerRepository
	registry   registry.Registry
	logger     *zap.Logger
}

func NewConsumeController(ctx context.Context, ch <-chan registry.Event, repo repository.ConsumerRepository, rg registry.Registry) *ConsumeController {
	p := &ConsumeController{
		notifyChan: ch,
		consumers: &sync.Pool{New: func() any {
			return NewConsumer(repo)
		}},
		repo:     repo,
		registry: rg,
		logger:   ioc.Logger,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.WatchLeftMessage(ctx, wg, config.StartConfig.Redis.ConsumerLeftMessageKey, time.Second*3)
	go p.Schedule(ctx, wg)
	wg.Wait()
	return p
}

func (c *ConsumeController) WatchLeftMessage(ctx context.Context, wg *sync.WaitGroup, key string, interval time.Duration) {
	wg.Done()
	c.logger.Info("ConsumeController", zap.String("WatchLeftMessage", "start"))
	// 每隔interval询问一次
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("ConsumeController", zap.String("WatchLeftMessage", "closed"))
			return
		case <-ticker.C:
			biz, err := c.repo.GetLeftTask(ctx, key)
			if err != nil && !errors.Is(err, redis.Nil) {
				c.logger.Error("ConsumeController", zap.Error(err))
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			go c.Assign(ctx, biz)
		}
	}
}

// Schedule 需要开启goroutine
func (c *ConsumeController) Schedule(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	c.logger.Info("ConsumeController", zap.String("Schedule", "start"))
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("ConsumeController", zap.String("Schedule", "closed"))
			return
		case event := <-c.notifyChan:
			if event.Type == registry.EventTypePut {
				biz := event.Instance.Note
				go c.Assign(ctx, biz)
			}
		}
	}
}

// Assign 需要开启goroutine
func (c *ConsumeController) Assign(ctx context.Context, bizName string) {
	c.logger.Info(bizName, zap.String("push", "start"))
	eg, egCtx := errgroup.WithContext(ctx)
	// 根据qps调整goroutine数量
	num := c.CalGoroutineNum(pushconfig.PushMap[bizName].Qps)
	for i := 0; i < num; i++ {
		eg.Go(func() error {
			consumer := c.consumers.Get().(*Consumer)
			defer c.consumers.Put(consumer)
			err := consumer.Consume(egCtx, bizName)
			return err
		})
	}
	// 等待所有消费完成
	// todo 排查为什么取消没劲道err分支。。 写回需要落日志
	if err := eg.Wait(); err != nil {
		c.logger.Info("Consumer", zap.String("status", "canceled"))
		// 能走到这说明外面ctx已经取消了，写回被打断的biz需要另起ctx
		er := c.repo.WriteBackLeftTask(context.Background(), bizName)
		if er != nil {
			c.logger.Error("ConsumeController", zap.String("Assign", "writeback"), zap.Error(er))
		}
		c.logger.Info("ConsumeController", zap.String("WriteBackLeftTask", "ok"))
		return
	} else {
		c.logger.Info(bizName, zap.String("push", "completed"))
	}
	service, err := c.registry.ListService(ctx, config.StartConfig.Register.ServiceName+config.StartConfig.Register.PodName)
	if err != nil {
		c.logger.Error("ConsumeController", zap.String("Assign", "registry"), zap.Error(err))
		return
	}
	if len(service) != 1 {
		c.logger.Error("ConsumeController", zap.String("Assign", "registry"), zap.Error(err))
		return
	}
	ins := service[0]
	ins.Weight -= pushconfig.PushMap[bizName].Weight
	err = c.registry.Register(ctx, ins)
	if err != nil {
		c.logger.Error("ConsumeController", zap.String("Assign", "registry"), zap.Error(err))
		return
	}
}

func (c *ConsumeController) CalGoroutineNum(qps int) int {
	// 假设默认网络延迟50ms
	const NetWorkDelay = 50
	// 一个goroutine在1s能处理的任务数量
	v := 1000 / NetWorkDelay
	return qps / v
}
