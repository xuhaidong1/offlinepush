package repository

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/go-generic-tools/ecache"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

var NoMessage = errors.New("没有消息")

type ConsumerRepository interface {
	// GetMessage 从本地缓存中取出消息消费
	GetMessage(ctx context.Context, bizName string) (domain.Message, error)
	// GetTask 从存储介质拿任务放到本地缓存中
	GetTask(ctx context.Context, key string) error
	// WriteBackLeftMessage 如果消费被打断，要把本地缓存里面的所有消息刷回存储介质中，由其它消费者消费
	WriteBackLeftMessage(ctx context.Context) error
	// GetLeftMessage 没有遗留消息会阻塞，故需要goroutine调用
	GetLeftMessage(ctx context.Context, key string, interval time.Duration) ([]string, error)
}

type consumerRepository struct {
	rdb   redis.Cmdable
	cache ecache.Cache
}

func NewConsumerRepository(rdb redis.Cmdable, cache ecache.Cache) ConsumerRepository {
	return &consumerRepository{rdb: rdb, cache: cache}
}

func (c *consumerRepository) GetMessage(ctx context.Context, bizName string) (domain.Message, error) {
	// TODO implement me
	panic("implement me")
}

func (c *consumerRepository) GetTask(ctx context.Context, key string) error {
	// TODO implement me
	panic("implement me")
}

func (c *consumerRepository) WriteBackLeftMessage(ctx context.Context) error {
	// TODO implement me
	panic("implement me")
}

func (c *consumerRepository) GetLeftMessage(ctx context.Context, key string, interval time.Duration) ([]string, error) {
	return c.rdb.BRPop(ctx, interval, key).Result()
}
