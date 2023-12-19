package component

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
)

type Counter struct {
	hopeCount   int64
	pushedCount int64
	rg          registry.Registry
	logger      logx.Logger
}

func NewCounter(rg registry.Registry) *Counter {
	return &Counter{
		hopeCount:   0,
		pushedCount: 0,
		rg:          rg,
		logger:      ioc.Loggerx.With(logx.Field{Key: "component", Value: "counter"}),
	}
}

func (c *Counter) ListenReset(ctx context.Context) {
	events, err := c.rg.Subscribe(config.StartConfig.Register.WriteCountKey)
	if err != nil {
		c.logger.Error("订阅reset key err", logx.Error(err))
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-events:
				atomic.StoreInt64(&c.hopeCount, 0)
				atomic.StoreInt64(&c.pushedCount, 0)
			}
		}
	}()
}

func (c *Counter) IncrPushedCount() {
	atomic.AddInt64(&c.pushedCount, 1)
}

func (c *Counter) GetCount() (int64, int64) {
	return atomic.LoadInt64(&c.hopeCount), atomic.LoadInt64(&c.pushedCount)
}

func (c *Counter) AddHopeCount(delta int64) {
	atomic.AddInt64(&c.hopeCount, delta)
}

func (c *Counter) Upload(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hope, rel := c.GetCount()
				err := c.rg.Register(ctx, registry.ServiceInstance{
					Address:     config.StartConfig.Register.PodName,
					ServiceName: config.StartConfig.Register.ReadCountKey,
					Note:        fmt.Sprintf("hopeCount:%d,realCount:%d", hope, rel),
				})
				if err != nil {
					c.logger.Error("上报消费数量失败", logx.Error(err))
				}
			}
		}
	}()
}

func (c *Counter) Run(ctx context.Context) {
	c.Upload(ctx)
	c.ListenReset(ctx)
}
