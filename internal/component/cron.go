package component

// mockgen -source=cron.go -destination=mocks/mock_cron.go
import (
	"context"

	robfig "github.com/robfig/cron/v3"
	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
)

type Croner interface {
	Subscribe(ctx context.Context) <-chan pushconfig.PushConfig
}

type Cron struct {
	cron        *robfig.Cron
	produceChan chan pushconfig.PushConfig
	logger      logx.Logger
}

func NewCron() Croner {
	c := robfig.New()
	c.Start()
	res := &Cron{
		cron:        c,
		produceChan: make(chan pushconfig.PushConfig),
		logger:      ioc.Loggerx.With(logx.Field{Key: "component", Value: "Cron"}),
	}
	res.InitConfig()
	res.logger.Info("started")
	return res
}

func (c *Cron) AddConfig(crontab string, config pushconfig.PushConfig) {
	_, err := c.cron.AddFunc(crontab, func() {
		select {
		case c.produceChan <- config:
		default:
		}
	})
	if err != nil {
		c.logger.Error("添加定时任务失败", logx.Error(err))
	}
}

func (c *Cron) Stop() {
	c.logger.Info("stopped")
	c.cron.Stop()
}

func (c *Cron) InitConfig() {
	for _, cfg := range pushconfig.PushMap {
		c.AddConfig(cfg.Cron, cfg)
	}
}

func (c *Cron) Subscribe(ctx context.Context) <-chan pushconfig.PushConfig {
	ch := make(chan pushconfig.PushConfig)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case cfg := <-c.produceChan:
				ch <- cfg
			}
		}
	}()
	return ch
}
