package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xuhaidong1/offlinepush/internal/domain"

	"github.com/xuhaidong1/offlinepush/internal/interceptor"

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
	notifyChan  <-chan registry.Event
	consumer    *WorkerV0
	repo        repository.ConsumerRepository
	interceptor *interceptor.Interceptor
	registry    registry.Registry
	rgMutex     *sync.Mutex
	logger      *zap.Logger
}

func NewConsumeController(ctx context.Context, ch <-chan registry.Event, repo repository.ConsumerRepository,
	interceptor *interceptor.Interceptor, rg registry.Registry,
) *ConsumeController {
	p := &ConsumeController{
		notifyChan:  ch,
		consumer:    NewWorkerV0(repo, interceptor),
		repo:        repo,
		registry:    rg,
		rgMutex:     &sync.Mutex{},
		interceptor: interceptor,
		logger:      ioc.Logger,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.WatchLeftTask(ctx, wg, config.StartConfig.Redis.ConsumerLeftTaskKey, time.Second*3)
	go p.Schedule(ctx, wg)
	wg.Wait()
	return p
}

func (c *ConsumeController) WatchLeftTask(ctx context.Context, wg *sync.WaitGroup, key string, interval time.Duration) {
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
			if !c.interceptor.Permit(biz) {
				c.logger.Info("ConsumeController", zap.String(biz, "stopped"))
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
				if !c.interceptor.Permit(biz) {
					c.logger.Info("ConsumeController", zap.String(biz, "paused"))
					continue
				}
				go c.Assign(ctx, biz)
			}
		}
	}
}

// Assign 需要开启goroutine
func (c *ConsumeController) Assign(ctx context.Context, bizName string) {
	c.SubWeight(ctx, bizName)
	c.logger.Info(bizName, zap.String("push", "start"))
	// 消费完成信号，由读到EOF的consume goroutine关闭
	finished, queueReady := make(chan struct{}), make(chan struct{})
	dequeueCtx, cancel := context.WithCancel(ctx)
	// 用于唤醒阻塞在dequeue中的goroutine
	go func() {
		<-finished
		cancel()
	}()
	// 拉取消息到本地缓存，消费者异步消费
	go c.Pull(ctx, bizName, queueReady)
	eg := &errgroup.Group{}
	// 根据qps调整goroutine数量
	num := c.CalGoroutineNum(pushconfig.PushMap[bizName].Qps)
	for i := 0; i < num; i++ {
		eg.Go(func() error {
			err := c.consumer.Work(ctx, dequeueCtx, bizName, finished, queueReady)
			return err
		})
	}
	// 等待所有消费完成
	if err := eg.Wait(); err != nil {
		c.logger.Info("Consumer", zap.Any("status", err))
		// 能走到这说明外面ctx已经取消了，写回被打断的biz需要另起ctx
		newCtx := context.WithoutCancel(ctx)
		// 拿到队列里面所有剩下的消息
		msgs, er := c.repo.GetAllMessage(newCtx, bizName)
		if er != nil {
			c.logger.Error("ConsumeController", zap.String("Assign", "GetAllMessage"), zap.Error(er))
		}
		c.logger.Info("ConsumeController", zap.String("GetAllMessage", "ok"))
		// 把剩下的消息都放回存储
		er = c.repo.WriteBackLeftMessage(newCtx, bizName, msgs)
		if er != nil {
			c.logger.Error("ConsumeController", zap.String("Assign", "WriteBackLeftMessage"), zap.Error(er))
		}
		c.logger.Info("ConsumeController", zap.String("WriteBackLeftMessage", "ok"))
		if errors.Is(err, context.Canceled) {
			// 如果是取消了，通知其它消费者消费；暂停的话实例还在，需要加回去weight
			er = c.repo.WriteBackLeftTask(newCtx, bizName)
			if er != nil {
				c.logger.Error("ConsumeController", zap.String("Assign", "WriteBackLeftTask"), zap.Error(er))
			}
			c.logger.Info("ConsumeController", zap.String("WriteBackLeftTask", "ok"))
			return
		}
	} else {
		c.logger.Info(bizName, zap.String("push", "completed"))
	}
	c.AddWeight(ctx, bizName)
}

func (c *ConsumeController) CalGoroutineNum(qps int) int {
	// 假设默认网络延迟50ms
	const NetWorkDelay = 50
	// 一个goroutine在1s能处理的任务数量
	v := 1000 / NetWorkDelay
	return qps / v
}

// SubWeight 开始消费时，减掉权重，消费完成，权重加回去
func (c *ConsumeController) SubWeight(ctx context.Context, biz string) {
	c.rgMutex.Lock()
	defer c.rgMutex.Unlock()
	service, err := c.registry.ListService(ctx, config.StartConfig.Register.ServiceName+config.StartConfig.Register.PodName)
	if err != nil {
		c.logger.Error("ConsumeController", zap.String("Assign", "SubWeight"), zap.Error(err))
		return
	}
	if len(service) != 1 {
		c.logger.Error("ConsumeController", zap.String("Assign", "SubWeight"), zap.Error(err))
		return
	}
	ins := service[0]
	ins.Weight -= pushconfig.PushMap[biz].Weight
	err = c.registry.Register(ctx, ins)
	if err != nil {
		c.logger.Error("ConsumeController",
			zap.String("Assign", "Register SubWeight"),
			zap.Error(err))
	}
}

func (c *ConsumeController) AddWeight(ctx context.Context, biz string) {
	c.rgMutex.Lock()
	defer c.rgMutex.Unlock()
	service, err := c.registry.ListService(ctx, config.StartConfig.Register.ServiceName+config.StartConfig.Register.PodName)
	if err != nil {
		c.logger.Error("ConsumeController", zap.String("Assign", "AddWeight"), zap.Error(err))
		return
	}
	if len(service) != 1 {
		c.logger.Error("ConsumeController", zap.String("Assign", "AddWeight"), zap.Error(err))
		return
	}
	ins := service[0]
	ins.Weight += pushconfig.PushMap[biz].Weight
	err = c.registry.Register(ctx, ins)
	if err != nil {
		c.logger.Error("ConsumeController",
			zap.String("Assign", "Register AddWeight"),
			zap.Error(err))
	}
}

func (c *ConsumeController) Pull(ctx context.Context, biz string, queueReady chan struct{}) {
	closeOnce := &sync.Once{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !c.interceptor.Permit(biz) {
				return
			}
			msgs, err := c.repo.PullMessage(ctx, biz)
			if err != nil && !errors.Is(err, NoMessage) {
				c.logger.Error("ConsumeController", zap.Any("Pull", err))
				return
			}
			if errors.Is(err, NoMessage) {
				// 存EOF不应该被取消
				err = c.repo.StoreMessage(context.WithoutCancel(ctx), domain.GetEOF(biz))
				return
			}
			for i, msg := range msgs {
				if !c.interceptor.Permit(biz) {
					er := c.WriteBackLeftMessage(context.WithoutCancel(ctx), biz, msgs[i:])
					if er != nil {
						c.logger.Error("ConsumeController", zap.Any("WriteBackLeftMessage before store", er))
					}
					return
				}
				// 耗时操作，可能会在这阻塞
				sErr := c.repo.StoreMessage(ctx, msg)
				// ctx cancel了
				if sErr != nil {
					er := c.WriteBackLeftMessage(context.WithoutCancel(ctx), biz, msgs[i:])
					if er != nil {
						c.logger.Error("ConsumeController", zap.Any("WriteBackLeftMessage", er))
					}
					return
				}
				// 关闭queueReady chan通知到所有消费者，可以开始dequeue，避免出现队列还没建立，就开始出队的问题
				closeOnce.Do(func() {
					close(queueReady)
				})
			}
		}
	}
}

func (c *ConsumeController) WriteBackLeftMessage(ctx context.Context, biz string, msgs []domain.Message) error {
	return c.repo.WriteBackLeftMessage(ctx, biz, msgs)
}
