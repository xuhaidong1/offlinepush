package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/conf"
	"sync"
)

var (
	redisCmd      redis.Cmdable
	redisInitOnce sync.Once
)

func InitRedis() redis.Cmdable {
	// 这里演示读取特定的某个字段
	redisInitOnce.Do(func() {
		redisCmd = redis.NewClient(&redis.Options{
			Addr: conf.StartConfig.Redis.Addr,
		})
	})
	return redisCmd
}
