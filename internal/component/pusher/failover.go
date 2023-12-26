package pusher

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

var PusherError = errors.New("PusherError")

type FailoverPusher struct {
	Pusher
	Healthy   bool
	Name      string
	cnt       int32
	threshold int32
	logger    logx.Logger
}

func (f *FailoverPusher) Push(ctx context.Context, msg domain.Message) error {
	err := f.Pusher.Push(ctx, msg)
	switch err {
	case nil:
		atomic.StoreInt32(&f.cnt, 0)
	case context.DeadlineExceeded:
		atomic.AddInt32(&f.cnt, 1)
	default:
		atomic.AddInt32(&f.cnt, 1)
	}
	if atomic.LoadInt32(&f.cnt) > f.threshold {
		err = PusherError
		f.markNotAvailable()
	}
	return err
}

func (f *FailoverPusher) markNotAvailable() {
	f.Healthy = false
	go func() {
		cnt := 0
		for {
			time.Sleep(time.Second * 10)
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
			err := f.Pusher.Push(ctx, PingMsg)
			cancel()
			if err == nil {
				cnt++
			} else {
				cnt = 0
			}
			if cnt >= 10 {
				f.Healthy = true
				return
			}
		}
	}()
}

var PingMsg = domain.Message{
	Topic:  domain.Topic{Name: "Ping"},
	Device: domain.Device{},
}
