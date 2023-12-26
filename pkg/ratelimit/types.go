package ratelimit

import "context"

type Limiter interface {
	// Limit 有没有触发限流，key是限流对象
	// bool代表是否限流，true 就是要限流
	// err限流器有没有错误
	Limit(ctx context.Context, key string) (bool, error)
}
