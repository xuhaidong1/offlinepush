package pusher

import (
	"context"

	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type Pusher interface {
	Push(ctx context.Context, msg domain.Message) error
}

type FailoverPusher struct {
	svcs []Pusher
}

func (f *FailoverPusher) Push(ctx context.Context, msg domain.Message) error {
	for _, svc := range f.svcs {
		err := svc.Push(ctx, msg)
		if err == nil {
			return nil
		}
		// log 监控
	}
	// 全都fail
	return nil
}
