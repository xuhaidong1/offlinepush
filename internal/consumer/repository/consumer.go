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
	produceRepo "github.com/xuhaidong1/offlinepush/internal/producer/repository"
)

var NoMessage = errors.New("没有消息")

//go:embed lua/get_message.lua
var luaGetMessage string

//go:embed lua/get_message_slice.lua
var luaGetMessageSlice string

type ConsumerRepository interface {
	// GetMessage 从本地缓存中取出消息消费,没有消息会阻塞
	GetMessage(ctx context.Context, bizName string) (domain.Message, error)
	// WriteBackLeftTask 如果消费被打断，要把受影响的biz追加遗留任务key里面，由其它消费者消费
	WriteBackLeftTask(ctx context.Context, bizName string) error
	// GetLeftTask 从遗留消息队列里面取一个业务
	GetLeftTask(ctx context.Context, key string) (biz string, err error)
	// PullMessage 从消息存储批量拉取消息
	PullMessage(ctx context.Context, biz string) (msgs []domain.Message, err error)
	// StoreMessage 存储1条消息消息到本地缓存（搭配PullMessage使用）
	StoreMessage(ctx context.Context, msg domain.Message) error
	// WriteBackLeftMessage 当ctx取消时，消费取消，写回在本地缓存&还没来得及放在本地缓存的消息到消息存储
	WriteBackLeftMessage(ctx context.Context, biz string, msgs []domain.Message) error
	// GetMessageFromStorage 从消息存储获取1条消息【弃用】
	GetMessageFromStorage(ctx context.Context, bizName string) (domain.Message, error)
	// GetAllMessage 从消费者的本地缓存中取出所有的消息，在cancel需要写回的时候调用
	GetAllMessage(ctx context.Context, biz string) ([]domain.Message, error)
}

type consumerRepository struct {
	local cache.LocalCache
	rdb   redis.Cmdable
}

func NewConsumerRepository(local cache.LocalCache, rdb redis.Cmdable) ConsumerRepository {
	return &consumerRepository{rdb: rdb, local: local}
}

func (r *consumerRepository) GetMessageFromStorage(ctx context.Context, bizName string) (domain.Message, error) {
	res, err := r.rdb.Eval(ctx, luaGetMessage, []string{bizName}).Result()
	if err != nil {
		return domain.Message{}, err
	}
	if values, ok := res.([]any); ok {
		// values 包含了 Lua 脚本中返回的多个值
		if len(values) == 3 {
			deviceType := values[0].(string)
			okk := values[2].(string)
			if okk == "ok" {
				return domain.Message{
					Topic: pushconfig.PushMap[bizName].Topic,
					Device: domain.Device{
						Type: deviceType,
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
	//		Topic: pushconfig.PushMap[bizName].Topic,
	//		Device: domain.Device{
	//			Type: strings.Split(taskKey, ":")[1],
	//			ID:   deviceID,
	//		},
	//	}, nil
	//}
}

func (r *consumerRepository) WriteBackLeftTask(ctx context.Context, bizName string) error {
	return r.rdb.RPush(ctx, config.StartConfig.Redis.ConsumerLeftTaskKey, bizName).Err()
}

func (r *consumerRepository) GetLeftTask(ctx context.Context, key string) (biz string, err error) {
	return r.rdb.LPop(ctx, key).Result()
}

func (r *consumerRepository) GetMessage(ctx context.Context, bizName string) (domain.Message, error) {
	return r.local.BLPop(ctx, bizName)
}

func (r *consumerRepository) PullMessage(ctx context.Context, biz string) (msgs []domain.Message, err error) {
	res, err := r.rdb.Eval(ctx, luaGetMessageSlice, []string{biz}).Result()
	if err != nil {
		return nil, err
	}
	if values, ok := res.([]any); ok {
		// values 包含了 Lua 脚本中返回的多个值
		if len(values) == 3 {
			okk := values[2].(string)
			if okk == "ok" {
				return msgs, nil
			} else {
				return nil, NoMessage
			}
		}
	}
	return nil, errors.New("lua返回有问题")
}

func (r *consumerRepository) StoreMessage(ctx context.Context, msg domain.Message) error {
	return r.local.BRPush(ctx, msg)
}

func (r *consumerRepository) WriteBackLeftMessage(ctx context.Context, biz string, msgs []domain.Message) error {
	task := produceRepo.NewTask(biz)
	for _, msg := range msgs {
		task.Add(msg)
	}
	_, err := r.rdb.SAdd(ctx, biz, task.Keys(biz)).Result()
	if err != nil {
		return err
	}
	for deviceType, idList := range task.DeviceMap {
		redisKey := biz + ":" + deviceType
		_, err = r.rdb.LPush(ctx, redisKey, idList).Result()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *consumerRepository) GetAllMessage(ctx context.Context, biz string) ([]domain.Message, error) {
	return r.local.LPopAll(ctx, biz)
}
