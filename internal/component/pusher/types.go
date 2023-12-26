package pusher

import (
	"context"

	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type Pusher interface {
	Push(ctx context.Context, msg domain.Message) error
}
