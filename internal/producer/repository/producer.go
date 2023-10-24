package repository

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/consumer/repository/cache"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

var (
	ErrKeyNotExist = redis.Nil
	NoMessage      = cache.ErrNoMessage
)

type ProducerRepository interface {
	// Store 暂存消息到本地缓存中
	Store(ctx context.Context, msg domain.Message) error
	// WriteBack 刷新本地缓存到存储介质中
	WriteBack(ctx context.Context, businessName string) error
	// WriteBackLeftTask 如果生产被打断，只把受影响的任务写到黑匣子，由下一任生产者启动时先根据黑匣子生产遗留任务
	WriteBackLeftTask(ctx context.Context, businessName string) error
	GetLeftTask(ctx context.Context) (pushconfig.PushConfig, error)
}

type producerRepository struct {
	local cache.LocalCache
	rdb   redis.Cmdable
}

func NewProducerRepository(local cache.LocalCache, rdb redis.Cmdable) ProducerRepository {
	return &producerRepository{rdb: rdb, local: local}
}

func (r *producerRepository) Store(ctx context.Context, msg domain.Message) error {
	r.local.RPush(ctx, msg)
	return nil
}

func (r *producerRepository) WriteBack(ctx context.Context, businessName string) error {
	task := NewTask(businessName)
	for {
		msg, err := r.local.LPop(ctx, businessName)
		if err != nil && !errors.Is(err, NoMessage) {
			return err
		}
		if errors.Is(err, NoMessage) {
			break
		}
		task.Add(msg)
	}
	_, err := r.rdb.SAdd(ctx, businessName, task.Keys(businessName)).Result()
	if err != nil {
		return err
	}
	for deviceType, idList := range task.DeviceMap {
		redisKey := businessName + ":" + deviceType
		_, err = r.rdb.LPush(ctx, redisKey, idList).Result()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *producerRepository) WriteBackLeftTask(ctx context.Context, businessName string) error {
	return r.rdb.RPush(ctx, config.StartConfig.Redis.ProducerLeftTaskKey, businessName).Err()
}

func (r *producerRepository) GetLeftTask(ctx context.Context) (pushconfig.PushConfig, error) {
	biz, err := r.rdb.LPop(ctx, config.StartConfig.Redis.ProducerLeftTaskKey).Result()
	if err != nil && errors.Is(err, ErrKeyNotExist) {
		return pushconfig.PushConfig{}, ErrKeyNotExist
	}
	return pushconfig.PushMap[biz], nil
}

type Task struct {
	BusinessName string
	// map key: deviceType val:id列表
	DeviceMap map[string][]string
	// 推送qps 根据设备数量，限制时间计算得出
	// Qps int `json:"qps"`
}

func NewTask(businessName string) *Task {
	return &Task{
		BusinessName: businessName,
		DeviceMap:    make(map[string][]string),
	}
}

func (t *Task) Add(msg domain.Message) {
	if m, ok := t.DeviceMap[msg.Device.Type]; ok {
		m = append(m, msg.Device.ID)
		t.DeviceMap[msg.Device.Type] = m
	} else {
		mp := make([]string, 0, 1024)
		mp = append(mp, msg.Device.ID)
		t.DeviceMap[msg.Device.Type] = mp
	}
}

func (t *Task) Keys(biz string) (res []string) {
	for k := range t.DeviceMap {
		res = append(res, biz+":"+k)
	}
	return
}
