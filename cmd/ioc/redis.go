package ioc

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/config"
	"sync"
	"time"
)

var (
	redisCmd      redis.Cmdable
	redisInitOnce sync.Once
)

func InitRedis() redis.Cmdable {
	// 这里演示读取特定的某个字段
	redisInitOnce.Do(func() {
		redisCmd = redis.NewClient(&redis.Options{
			Addr: config.StartConfig.Redis.Addr,
		})
		PingRedis(redisCmd)
	})
	return redisCmd
}

func PingRedis(redisCmd redis.Cmdable) {
	redisCmd.Set(context.Background(), "key1", "val1", time.Minute)
	result, err := redisCmd.Get(context.Background(), "key1").Result()
	if err != nil {
		panic(err)
	}
	if result != "val1" {
		panic("值不对")
	}
}
