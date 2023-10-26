package main

import (
	"context"
	"time"

	"github.com/xuhaidong1/offlinepush/internal/cron"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	"go.uber.org/zap"
)

type GracefulShutdown struct {
	Cancel         context.CancelFunc
	Registry       registry.Registry
	instance       registry.ServiceInstance
	consumeIns     registry.ServiceInstance
	cronController *cron.CronController
	logger         *zap.Logger
}

func NewGracefulShutdown(cancel context.CancelFunc, registry registry.Registry,
	instance registry.ServiceInstance, consumeIns registry.ServiceInstance,
	cronController *cron.CronController, logger *zap.Logger,
) *GracefulShutdown {
	return &GracefulShutdown{
		Cancel:         cancel,
		Registry:       registry,
		instance:       instance,
		consumeIns:     consumeIns,
		cronController: cronController,
		logger:         logger,
	}
}

func (s *GracefulShutdown) Shutdown() {
	s.Cancel()
	s.logger.Info("GracefulShutdown", zap.String("GracefulShutdown", "UnRegister"))
	err := s.Registry.UnRegister(context.Background(), s.instance)
	if err != nil {
		s.logger.Error("GracefulShutdown", zap.Error(err))
	}
	err = s.Registry.UnRegister(context.Background(), s.consumeIns)
	if err != nil {
		s.logger.Error("GracefulShutdown", zap.Error(err))
	}
	err = s.Registry.Close()
	if err != nil {
		s.logger.Error("GracefulShutdown", zap.Error(err))
	}
	s.cronController.Stop()
	time.Sleep(time.Second)
}
