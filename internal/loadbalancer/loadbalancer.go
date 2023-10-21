package loadbalancer

import (
	"context"
	"log"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
)

type LoadBalancer struct {
	registry registry.Registry
}

func NewLoadBalancer(registry registry.Registry) *LoadBalancer {
	return &LoadBalancer{
		registry: registry,
	}
}

func (l *LoadBalancer) SelectConsumer(cfg pushconfig.PushConfig) {
	ctx := context.Background()
	insSlice, err := l.registry.ListService(ctx, config.StartConfig.Register.ServiceName)
	if err != nil {
		log.Fatalln(err)
	}
	targetIns := l.Max(insSlice)
	targetIns.Weight -= cfg.Weight
	consumeIns := registry.ServiceInstance{
		Address:     targetIns.Address,
		ServiceName: config.StartConfig.Register.ConsumerPrefix + targetIns.Address,
		Note:        cfg.Business.Name,
	}
	err = l.registry.Register(ctx, targetIns)
	if err != nil {
		log.Fatalln(err)
	}
	err = l.registry.Register(ctx, consumeIns)
	if err != nil {
		log.Fatalln(err)
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
