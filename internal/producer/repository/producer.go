package repository

import (
	"context"

	"github.com/ecodeclub/ecache"
	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type ProducerRepository interface {
	// Store 暂存消息到本地缓存中
	Store(ctx context.Context, msg domain.Message) error
	// WriteBack 刷新本地缓存到存储介质中
	WriteBack(ctx context.Context) error
	// WriteBackLeftTask 如果生产被打断，只把受影响的任务写到黑匣子，由下一任生产者启动时先根据黑匣子生产遗留任务
	WriteBackLeftTask(ctx context.Context, cfg pushconfig.PushConfig) error
	GetLeftTask(ctx context.Context) (pushconfig.PushConfig, error)
}

type producerRepository struct {
	rdb        redis.Cmdable
	localCache ecache.Cache
}

func NewProducerRepository(rdb redis.Cmdable, localCache ecache.Cache) ProducerRepository {
	return &producerRepository{rdb: rdb, localCache: localCache}
}

func (p *producerRepository) Store(ctx context.Context, msg domain.Message) error {
	// TODO implement me
	panic("implement me")
}

func (p *producerRepository) WriteBack(ctx context.Context) error {
	// TODO implement me
	panic("implement me")
}

func (p *producerRepository) WriteBackLeftTask(ctx context.Context, cfg pushconfig.PushConfig) error {
	// TODO implement me
	panic("implement me")
}

func (p *producerRepository) GetLeftTask(ctx context.Context) (pushconfig.PushConfig, error) {
	// TODO implement me
	panic("implement me")
}

type TaskEntity struct {
	BusinessName string   `json:"business_name"`
	DeviceType   string   `json:"device_type"`
	DeviceIDList []string `json:"device_id_list"`
	Qps          int      `json:"qps"`
}
