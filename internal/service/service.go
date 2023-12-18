package service

import (
	"context"
	"os"
	"syscall"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/interceptor"
	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
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
	notifyProducer chan<- pushconfig.PushConfig
	interceptor    *interceptor.Interceptor
	producerRepo   repository.ProducerRepository
	register       registry.Registry
	shutdownCh     chan os.Signal
}

func NewPushService(notifyProducer chan<- pushconfig.PushConfig, interceptor *interceptor.Interceptor,
	producerRepo repository.ProducerRepository,
	register registry.Registry, shutdownCh chan os.Signal,
) *PushService {
	return &PushService{
		notifyProducer: notifyProducer,
		interceptor:    interceptor,
		producerRepo:   producerRepo,
		register:       register,
		shutdownCh:     shutdownCh,
	}
}

func (s *PushService) GetBizStatus() map[string]bool {
	return s.interceptor.GetMap()
}

// AddTask 开始执行某个业务的推送任务[开始生产-消费]
func (s *PushService) AddTask(ctx context.Context, bizName string) error {
	return s.producerRepo.WriteBackLeftTask(ctx, bizName)
}

// Resume 恢复某个业务推送
func (s *PushService) Resume(ctx context.Context, bizName string) error {
	err := s.interceptor.ResumeBiz(ctx, bizName)
	if err != nil {
		return err
	}
	// 写一个遗留任务到redis，由某个消费者开始消费积压的消息，生产者无需通知，生产者在指定时间生产之前会判断是否允许生产
	// err = s.consumerRepo.WriteBackLeftTask(ctx, bizName)
	return err
}

// Pause 暂停某个业务推送，生产者；已经在生产的会正常生产完毕，消息存起来；之后到了指定生产时间收到了会收到cron信号，但不会开始生产；人工AddTask也不会生效
// 消费者：正在消费的正常退出，不需要写回遗留任务；再来新的消费任务也会拒绝。
func (s *PushService) Pause(ctx context.Context, bizName string) error {
	return s.interceptor.PauseBiz(ctx, bizName)
}

func (s *PushService) PodList(ctx context.Context) ([]registry.ServiceInstance, error) {
	service, err := s.register.ListService(ctx, config.StartConfig.Register.ServiceName)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (s *PushService) Shutdown() {
	s.shutdownCh <- syscall.SIGINT
}
