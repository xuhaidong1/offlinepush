package cron

import (
	robfig "github.com/robfig/cron"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"go.uber.org/zap"
)

type CronController struct {
	cron        *robfig.Cron
	produceChan chan<- pushconfig.PushConfig
	logger      *zap.Logger
}

func NewCronController(produceChan chan<- pushconfig.PushConfig) *CronController {
	c := robfig.New()
	c.Start()
	res := &CronController{
		cron:        c,
		produceChan: produceChan,
		logger:      ioc.Logger,
	}
	res.InitConfig()
	res.logger.Info("CronController", zap.String("status", "start"))
	return res
}

func (c *CronController) AddConfig(crontab string, config pushconfig.PushConfig) {
	err := c.cron.AddFunc(crontab, func() {
		c.produceChan <- config
	})
	if err != nil {
		c.logger.Error("CronController", zap.Error(err))
	}
}

func (c *CronController) Stop() {
	c.logger.Info("CronController", zap.String("status", "closed"))
	c.cron.Stop()
}

func (c *CronController) InitConfig() {
	for _, cfg := range pushconfig.PushMap {
		c.AddConfig(cfg.Cron, cfg)
	}
}
