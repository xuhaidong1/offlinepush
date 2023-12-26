package pusher

import (
	"context"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"github.com/xuhaidong1/offlinepush/pkg/ratelimit"
)

type RateLimitPusher struct {
	limiter ratelimit.Limiter
	svcs    []*FailoverPusher
	logger  logx.Logger
	x       cond.CondAtomic
}

func (r *RateLimitPusher) Push(ctx context.Context, msg domain.Message) error {
	for k := 0; k < 10; k++ {
		svc := r.svcs[k%len(r.svcs)]
		if !svc.Healthy {
			continue
		}
		limited, err := r.limiter.Limit(ctx, svc.Name)
		if err != nil {
			r.logger.Error("限流器err", logx.Error(err))
		}
		if !limited {
			er := svc.Push(ctx, msg)
			switch er {
			case nil:
				return nil
			case context.DeadlineExceeded:
				r.logger.Warn("推送超时", logx.String("服务商", svc.Name), logx.Error(er))
			case PusherError:
				r.logger.Warn("服务商连续err，摘出可用服务列表", logx.String("服务商", svc.Name), logx.Error(er))
			default:
				r.logger.Warn("推送failed", logx.String("服务商", svc.Name), logx.Error(er))
			}
		}
	}
	return PusherError
}
