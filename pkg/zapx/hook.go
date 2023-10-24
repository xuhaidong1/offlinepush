package zapx

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ZapRedisHook struct {
	logger *zap.Logger
}

func (h *ZapRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *ZapRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.logger.Debug("Redis command", zap.String("cmd", cmd.Name()))
		return next(ctx, cmd)
	}
}

func (h *ZapRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			h.logger.Debug("Redis pipeline command", zap.String("cmd", cmd.Name()))
		}
		return next(ctx, cmds)
	}
}

func NewZapRedisHook(logger *zap.Logger) *ZapRedisHook {
	return &ZapRedisHook{logger: logger}
}
