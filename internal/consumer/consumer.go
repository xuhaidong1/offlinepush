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
	return &Consumer{
		repo: repo,
	}
}

func (c *Consumer) Consume(ctx context.Context, bizName string) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("consumer closing")
			return ctx.Err()
		default:
			msg, err := c.repo.GetMessage(ctx, bizName)
			if err != nil && !errors.Is(err, NoMessage) {
				return err
			}
			if errors.Is(err, NoMessage) {
				log.Println("comsume nomessage")
				return nil
			}
			// 在这里mock推送。。
			c.Push(msg)
		}
	}
}

func (c *Consumer) Push(msg domain.Message) {
	log.Printf("pushing %v\n", msg)
	// time.Sleep(50 * time.Millisecond)
}
