package loadbalancer

import (
	"context"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	"go.uber.org/zap"
)

type LoadBalancer struct {
	registry registry.Registry
	logger   *zap.Logger
}

func NewLoadBalancer(registry registry.Registry) *LoadBalancer {
	return &LoadBalancer{
		registry: registry,
		logger:   ioc.Logger,
	}
}

func (l *LoadBalancer) SelectConsumer(cfg pushconfig.PushConfig) {
	ctx := context.Background()
	insSlice, err := l.registry.ListService(ctx, config.StartConfig.Register.ServiceName)
	if err != nil {
		l.logger.Error("LoadBalancer",
			zap.String("SelectConsumer", "ListService"),
			zap.Error(err))
	}
	l.logger.Info("LoadBalancer", zap.Any("ListService", insSlice))
	targetIns := l.Max(insSlice)
	consumeIns := registry.ServiceInstance{
		Address:     targetIns.Address,
		ServiceName: config.StartConfig.Register.ConsumerPrefix + targetIns.Address,
		Note:        cfg.Business.Name,
	}
	err = l.registry.Register(ctx, consumeIns)
	if err != nil {
		l.logger.Error("LoadBalancer",
			zap.String("SelectConsumer", "Register consumeIns"),
			zap.Error(err))
	}
}

func (l *LoadBalancer) Max(insSlice []registry.ServiceInstance) registry.ServiceInstance {
	res := registry.ServiceInstance{}
	for _, ins := range insSlice {
		if ins.Weight > res.Weight {
			res = ins
		}
	}
	return res
}
