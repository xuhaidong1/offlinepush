package consumer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

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
}

func NewConsumeController(ctx context.Context, ch <-chan registry.Event, repo repository.ConsumerRepository, rg registry.Registry) *ConsumeController {
	p := &ConsumeController{
		notifyChan: ch,
		consumers: &sync.Pool{New: func() any {
			return NewConsumer(repo)
		}},
		repo:     repo,
		registry: rg,
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.WatchLeftMessage(ctx, wg, "leftMsg", time.Second*3)
	go p.Schedule(ctx, wg)
	wg.Wait()
	return p
}

func (c *ConsumeController) WatchLeftMessage(ctx context.Context, wg *sync.WaitGroup, key string, interval time.Duration) {
	wg.Done()
	log.Println("ConsumeController WatchLeftMessage start")
	// 每隔interval询问一次
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("ConsumeController WatchLeftMessage closing")
			return
		case <-ticker.C:
			biz, err := c.repo.GetLeftTask(ctx, key)
			if err != nil && !errors.Is(err, redis.Nil) {
				log.Fatal(err)
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
	log.Println("ConsumeController Schedule start")
	for {
		select {
		case <-ctx.Done():
			log.Println("ConsumeConcroller schedule closing")
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
	eg, egCtx := errgroup.WithContext(ctx)
	// todo 根据qps调整goroutine数量
	for i := 0; i < 5; i++ {
		eg.Go(func() error {
			consumer := c.consumers.Get().(*Consumer)
			defer c.consumers.Put(consumer)
			err := consumer.Consume(egCtx, bizName)
			return err
		})
	}
	// 等待所有消费完成
	if err := eg.Wait(); err != nil {
		er := c.repo.WriteBackLeftTask(ctx, bizName)
		if er != nil {
			log.Fatalln(er)
		}
	} else {
		fmt.Printf("%s 推送任务完成\n", bizName)
	}

	service, err := c.registry.ListService(ctx, config.StartConfig.Register.ServiceName+config.StartConfig.Register.PodName)
	if err != nil {
		log.Fatalln(err)
		return
	}
	if len(service) != 1 {
		log.Fatalln("查找本实例修改weight失败")
		return
	}
	ins := service[0]
	ins.Weight -= pushconfig.PushMap[bizName].Weight
	err = c.registry.Register(ctx, ins)
	if err != nil {
		log.Fatalln(err)
		return
	}
}
