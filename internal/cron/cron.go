package cron

import (
	"log"

	robfig "github.com/robfig/cron"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
)

type CronController struct {
	cron        *robfig.Cron
	produceChan chan<- pushconfig.PushConfig
}

func NewCronController(produceChan chan<- pushconfig.PushConfig) *CronController {
	c := robfig.New()
	c.Start()
	res := &CronController{
		cron:        c,
		produceChan: produceChan,
	}
	res.InitConfig()
	return res
}

func (c *CronController) AddConfig(crontab string, config pushconfig.PushConfig) {
	err := c.cron.AddFunc(crontab, func() {
		c.produceChan <- config
	})
	if err != nil {
		log.Fatalln(err)
	}
}

func (c *CronController) Stop() {
	c.cron.Stop()
}

func (c *CronController) InitConfig() {
	for _, cfg := range pushconfig.PushMap {
		c.AddConfig(cfg.Cron, cfg)
	}
}
