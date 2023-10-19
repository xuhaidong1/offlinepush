package consumer

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
)

type ConsumeController struct {
	notifyChan <-chan registry.Event
	// consumeChan 传递的是business name
	consumeChan chan string
	consumers   *sync.Pool
	repo        repository.ConsumerRepository
}

func NewConsumeController(ctx context.Context, ch <-chan registry.Event, repo repository.ConsumerRepository) *ConsumeController {
	p := &ConsumeController{
		notifyChan:  ch,
		consumeChan: make(chan string, 10), // 这里应该根据有多少种推送业务来确定容量
		consumers: &sync.Pool{New: func() any {
			return NewConsumer(repo)
		}},
		repo: repo,
	}
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go p.WatchChan(ctx, wg)
	go p.WatchLeftMessage(ctx, wg, "leftMsg", time.Second)
	go p.Schedule(ctx, wg)
	wg.Wait()
	return p
}

func (c *ConsumeController) WatchChan(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	log.Println("ConsumeController WatchChan start")
	for {
		select {
		case event := <-c.notifyChan:
			if event.Type == registry.EventTypePut {
				taskKey := event.Instance.Note
				// getTask 完成取消息，存到本地缓存
				err := c.repo.GetTask(ctx, taskKey)
				if err != nil {
					log.Fatal(err)
				}
				c.consumeChan <- taskKey
			}
		case <-ctx.Done():
			log.Println("ConsumeController WatchChan closing")
			return
		}
	}
}

func (c *ConsumeController) WatchLeftMessage(ctx context.Context, wg *sync.WaitGroup, key string, interval time.Duration) {
	wg.Done()
	log.Println("ConsumeController WatchLeftMessage start")
	for {
		select {
		case <-ctx.Done():
			log.Println("ConsumeController WatchLeftMessage closing")
			return
		default:
			msg, err := c.repo.GetLeftMessage(ctx, key, interval)
			if err != nil && !errors.Is(err, redis.Nil) {
				log.Fatal(err)
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Println(msg)
			// msg 塞到缓存里面
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
		case biz := <-c.consumeChan:
			go c.Assign(ctx, biz)
		}
	}
}

// Assign 需要开启goroutine
func (c *ConsumeController) Assign(ctx context.Context, bizName string) {
	consumer := c.consumers.Get().(*Consumer)
	consumer.Consume(ctx, bizName)
	c.consumers.Put(consumer)
}
