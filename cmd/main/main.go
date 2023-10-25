package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/xuhaidong1/offlinepush/internal/consumer/repository/cache"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/cron"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/go-generic-tools/redis_lock"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/internal/consumer"
	consumerrepo "github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/internal/loadbalancer"
	"github.com/xuhaidong1/offlinepush/internal/lock"
	"github.com/xuhaidong1/offlinepush/internal/producer"
	producerrepo "github.com/xuhaidong1/offlinepush/internal/producer/repository"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	etcd2 "github.com/xuhaidong1/offlinepush/pkg/registry/etcd"
)

// 运行要加上 --config=conf/dev.toml
// 并且可以开启环境变量 EGO_DEBUG=true
func main() {
	//--初始化三方依赖
	etcd := ioc.InitEtcd()
	rdb := ioc.InitRedis()
	ctx, cancel := context.WithCancel(context.Background())
	localCache := cache.NewLocalCache()
	//----实例注册-------
	podName := config.GetPodName()
	rg, err := etcd2.NewRegistry(etcd)
	if err != nil {
		panic(err)
	}
	ins, consumeIns, err := RegisterService(ctx, rg)
	if err != nil {
		panic(err)
	}
	//-------通道创建------
	// 消费者只订阅自己
	notifyConsumerChan, err := rg.Subscribe(consumeIns.ServiceName)
	if err != nil {
		panic(err)
	}
	notifyProducerChan := make(chan pushconfig.PushConfig, 10)
	notifyLoadBalancerChan := make(chan pushconfig.PushConfig, 10)
	//----消费者初始化-----------------
	consumeRepo := consumerrepo.NewConsumerRepository(localCache, rdb)
	consumer.NewConsumeController(ctx, notifyConsumerChan, consumeRepo, rg)

	//----分布式锁管理----
	lockClient := redis_lock.NewClient(rdb)
	// cond用于管理生产者/负载均衡组件的启动停止
	startCond := cond.NewCondAtomic(&sync.Mutex{})
	stopCond := cond.NewCondAtomic(&sync.Mutex{})
	//----负载均衡控制器初始化----
	loadbalancer.NewLoadBalanceController(ctx, notifyLoadBalancerChan, startCond, stopCond, rg)
	//----生产者控制器初始化------
	producerRepo := producerrepo.NewProducerRepository(localCache, rdb)
	producer.NewProduceController(ctx, notifyProducerChan, notifyLoadBalancerChan, startCond, stopCond, producerRepo)
	//-----定时任务控制器初始化----
	cronController := cron.NewCronController(notifyProducerChan)
	//-----锁控制器初始化--------
	lockController := lock.NewLockController(lockClient, podName, startCond, stopCond, config.StartConfig.Lock)
	lockController.Run(ctx)
	//----优雅关闭初始化------
	gs := &GracefulShutdown{
		Cancel:         cancel,
		Registry:       rg,
		instance:       ins,
		consumeIns:     consumeIns,
		cronController: cronController,
		logger:         ioc.Logger,
	}
	time.Sleep(time.Second * 2)
	go func() {
		notifyProducerChan <- pushconfig.PushMap["reboot"]
		// time.Sleep(time.Second)
		// notifyProducerChan <- pushconfig.PushMap["weather"]
		// time.Sleep(time.Second*3)
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	gs.Shutdown()
}

type GracefulShutdown struct {
	Cancel         context.CancelFunc
	Registry       registry.Registry
	instance       registry.ServiceInstance
	consumeIns     registry.ServiceInstance
	cronController *cron.CronController
	logger         *zap.Logger
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

func RegisterService(ctx context.Context, rg registry.Registry) (ins, consumeIns registry.ServiceInstance, err error) {
	ins = registry.ServiceInstance{
		Address:     config.StartConfig.Register.PodName,
		ServiceName: config.StartConfig.Register.ServiceName + config.StartConfig.Register.PodName,
		Weight:      10000,
	}
	// 这个注册的是服务，生产者调用服务列表的key：config.StartConfig.Register.ServiceName，根据服务weight确定通知的实例。
	err = rg.Register(ctx, ins)
	if err != nil {
		return registry.ServiceInstance{}, registry.ServiceInstance{}, err
	}
	consumeIns = registry.ServiceInstance{
		Address:     config.StartConfig.Register.PodName,
		ServiceName: config.StartConfig.Register.ConsumerPrefix + config.StartConfig.Register.PodName,
	}
	// 这个注册的是消费者抽象，生产者准备好消息-负载均衡-通知这个抽象-消费者收到通知
	err = rg.Register(ctx, consumeIns)
	if err != nil {
		return registry.ServiceInstance{}, registry.ServiceInstance{}, err
	}
	return ins, consumeIns, err
}
