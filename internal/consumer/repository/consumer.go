package repository

import (
	"context"
	_ "embed"
	"errors"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/consumer/repository/cache"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

var NoMessage = errors.New("没有消息")

//go:embed lua/get_message.lua
var luaGetMessage string

type ConsumerRepository interface {
	// GetMessage 从本地缓存中取出消息消费
	GetMessage(ctx context.Context, bizName string) (domain.Message, error)
	// WriteBackLeftTask 如果消费被打断，要把受影响的biz追加遗留任务key里面，由其它消费者消费
	WriteBackLeftTask(ctx context.Context, bizName string) error
	// GetLeftTask 从遗留消息队列里面取一个业务
	GetLeftTask(ctx context.Context, key string) (biz string, err error)
}

type consumerRepository struct {
	local cache.LocalCache
	rdb   redis.Cmdable
}

func NewConsumerRepository(local cache.LocalCache, rdb redis.Cmdable) ConsumerRepository {
	return &consumerRepository{rdb: rdb, local: local}
}

func (r *consumerRepository) GetMessage(ctx context.Context, bizName string) (domain.Message, error) {
	res, err := r.rdb.Eval(ctx, luaGetMessage, []string{bizName}).Result()
	if err != nil {
		return domain.Message{}, err
	}
	if values, ok := res.([]any); ok {
		// values 包含了 Lua 脚本中返回的多个值
		if len(values) == 3 {
			deviceType := values[0].(string)
			deviceID := values[1].(string)
			okk := values[2].(string)
			if okk == "ok" {
				return domain.Message{
					Business: pushconfig.PushMap[bizName].Business,
					Device: domain.Device{
						Type: deviceType,
						ID:   deviceID,
					},
				}, nil
			} else {
				return domain.Message{}, NoMessage
			}
		}
	}
	return domain.Message{}, errors.New("lua返回有问题")
	//for {
	//	taskKey, err := r.rdb.SRandMember(ctx, bizName).Result()
	//	if errors.Is(err, redis.Nil) {
	//		r.rdb.Del(ctx, bizName)
	//		return domain.Message{}, NoMessage
	//	}
	//	deviceID, err := r.rdb.RPop(ctx, taskKey).Result()
	//	if errors.Is(err, redis.Nil) {
	//		r.rdb.SRem(ctx, bizName, taskKey)
	//		continue
	//	}
	//	return domain.Message{
	//		Business: pushconfig.PushMap[bizName].Business,
	//		Device: domain.Device{
	//			Type: strings.Split(taskKey, ":")[1],
	//			ID:   deviceID,
	//		},
	//	}, nil
	//}
}

func (r *consumerRepository) WriteBackLeftTask(ctx context.Context, bizName string) error {
	return r.rdb.RPush(ctx, config.StartConfig.Redis.ConsumerLeftMessageKey, bizName).Err()
}

func (r *consumerRepository) GetLeftTask(ctx context.Context, key string) (biz string, err error) {
	return r.rdb.LPop(ctx, key).Result()
}
