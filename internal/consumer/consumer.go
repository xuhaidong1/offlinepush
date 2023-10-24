package consumer

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"go.uber.org/zap"
)

var NoMessage = repository.NoMessage

type Consumer struct {
	repo       repository.ConsumerRepository
	pushLogger *log.Logger
	logger     *zap.Logger
}

func NewConsumer(repo repository.ConsumerRepository) *Consumer {
	return &Consumer{
		repo:       repo,
		pushLogger: ioc.PushLogger,
		logger:     ioc.Logger,
	}
}

func (c *Consumer) Consume(ctx context.Context, bizName string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.repo.GetMessage(ctx, bizName)
			if err != nil && !errors.Is(err, NoMessage) {
				c.logger.Error("Consumer", zap.String("Consume", "GetMessage"), zap.Error(err))
				return nil
			}
			if errors.Is(err, NoMessage) {
				return nil
			}
			// 在这里mock推送。。
			c.Push(msg)
		}
	}
}

func (c *Consumer) Push(msg domain.Message) {
	time.Sleep(50 * time.Millisecond)
	c.pushLogger.Printf("push %v\n", msg)
}
