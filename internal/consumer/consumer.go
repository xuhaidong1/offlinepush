package consumer

import (
	"context"
	"errors"
	"log"

	"github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

var NoMessage = repository.NoMessage

type Consumer struct {
	repo repository.ConsumerRepository
}

func NewConsumer(repo repository.ConsumerRepository) *Consumer {
	return &Consumer{repo: repo}
}

func (c *Consumer) Consume(ctx context.Context, bizName string) {
	for {
		select {
		case <-ctx.Done():
			log.Println("consumer closing")
		default:
			msg, err := c.repo.GetMessage(ctx, bizName)
			if err != nil || !errors.Is(err, NoMessage) {
				log.Fatal(err)
			}
			if errors.Is(err, NoMessage) {
				return
			}
			// 在这里mock推送。。
			c.Push(msg)
		}
	}
}

func (c *Consumer) Push(msg domain.Message) {
	log.Printf("pushing %v\n", msg)
}
