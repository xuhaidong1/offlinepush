package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/component"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
)

// PushService 1. Post添加业务推送配置--no
//  2. Get获取业务推送配置
//  3. Post开始业务正常推送
//  4. Post手动暂停业务推送
//  5. 查保存进度
//  6. 查业务生产的消息
//  7. 删业务的消息队列
//  8. 查保存进度
//  9. 删保存进度
type PushService struct {
	notifyProducer chan pushconfig.PushConfig
	interceptor    *component.Interceptor
	register       registry.Registry
	counter        *component.Counter
	// shutdownCh     chan os.Signal
}

func NewPushService(interceptor *component.Interceptor,
	register registry.Registry, counter *component.Counter,
) *PushService {
	return &PushService{
		notifyProducer: make(chan pushconfig.PushConfig),
		interceptor:    interceptor,
		register:       register,
		counter:        counter,
		// shutdownCh:     shutdownCh,
	}
}

func (s *PushService) GetBizStatus() map[string]bool {
	return s.interceptor.GetMap()
}

func (s *PushService) ResetCounter(ctx context.Context) error {
	return s.register.Register(ctx, registry.ServiceInstance{
		Address:     "",
		ServiceName: config.StartConfig.Register.WriteCountKey,
	})
}

func (s *PushService) GetCount(ctx context.Context) (map[string]string, error) {
	res := make(map[string]string)
	services, err := s.register.ListService(ctx, config.StartConfig.Register.ReadCountKey)
	if err != nil {
		return res, err
	}
	for _, sv := range services {
		res[sv.Address] = sv.Note
	}
	return res, nil
}

// AddTask 开始执行某个业务的推送任务[开始生产-消费]
// todo 幂等性校验 1小时内发送过就不让再发送
func (s *PushService) AddTask(ctx context.Context, topic string, num int) error {
	cfg, ok := pushconfig.PushMap[topic]
	if !ok {
		return errors.New("topic doesn't exist")
	}
	cfg.Num = num
	marshal, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.register.Register(ctx, registry.ServiceInstance{
		Address:     config.StartConfig.Register.PodName,
		ServiceName: config.StartConfig.Register.ManualTaskKey,
		Note:        string(marshal),
	})
}

// Resume 恢复某个业务推送，生产者：批准生产；消费者：重新开始监听kafka
//func (s *PushService) Resume(ctx context.Context, bizName string) error {
//	err := s.interceptor.ResumeBiz(ctx, bizName)
//	if err != nil {
//		return err
//	}
//	// 写一个遗留任务到redis，由某个消费者开始消费积压的消息，生产者无需通知，生产者在指定时间生产之前会判断是否允许生产
//	// err = s.consumerRepo.WriteBackLeftTask(ctx, bizName)
//	return err
//}

// Pause 暂停某个业务推送，生产者；停止生产
// 消费者：sarama停止消费
//func (s *PushService) Pause(ctx context.Context, bizName string) error {
//	return s.interceptor.PauseBiz(ctx, bizName)
//}

func (s *PushService) PodList(ctx context.Context) ([]registry.ServiceInstance, error) {
	service, err := s.register.ListService(ctx, config.StartConfig.Register.ServiceName)
	if err != nil {
		return nil, err
	}
	return service, nil
}

//func (s *PushService) Shutdown() {
//	s.shutdownCh <- syscall.SIGINT
//}
