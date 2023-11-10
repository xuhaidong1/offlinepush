package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/xuhaidong1/offlinepush/internal/consumer/repository/cache"
	interceptor2 "github.com/xuhaidong1/offlinepush/internal/interceptor"
	"github.com/xuhaidong1/offlinepush/internal/service"
	"github.com/xuhaidong1/offlinepush/web"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/cron"

	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/go-generic-tools/redis_lock"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/internal/consumer"
	consumerrepo "github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/internal/lock"
	"github.com/xuhaidong1/offlinepush/internal/producer"
	producerrepo "github.com/xuhaidong1/offlinepush/internal/producer/repository"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	etcd2 "github.com/xuhaidong1/offlinepush/pkg/registry/etcd"
)

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
	//-------通道/拦截器创建------
	// 消费者只订阅自己
	notifyConsumerChan, err := rg.Subscribe(consumeIns.ServiceName)
	if err != nil {
		panic(err)
	}
	notifyProducerChan := make(chan pushconfig.PushConfig, 10)
	notifyLoadBalancerChan := make(chan pushconfig.PushConfig, 10)
	interceptor := interceptor2.NewInterceptor(ctx, rg, ioc.Logger)
	//----消费者初始化-----------------
	consumeRepo := consumerrepo.NewConsumerRepository(localCache, rdb)
	consumer.NewConsumeController(ctx, notifyConsumerChan, consumeRepo, interceptor, rg)

	//----分布式锁管理----
	lockClient := redis_lock.NewClient(rdb)
	// cond用于管理生产者/负载均衡组件的任命/卸任
	engageCond := cond.NewCondAtomic(&sync.Mutex{})
	dismissCond := cond.NewCondAtomic(&sync.Mutex{})
	//----生产者控制器初始化------
	producerRepo := producerrepo.NewProducerRepository(localCache, rdb)
	producer.NewProduceController(ctx, notifyProducerChan, notifyLoadBalancerChan, engageCond,
		dismissCond, producerRepo, interceptor)
	//-----定时任务控制器初始化----
	cronController := cron.NewCronController(notifyProducerChan)
	//-----锁控制器初始化--------
	lockController := lock.NewLockController(lockClient, podName, engageCond, dismissCond, config.StartConfig.Lock)
	lockController.Run(ctx)
	//----优雅关闭初始化------
	gs := NewGracefulShutdown(cancel, rg, ins, consumeIns, cronController, ioc.Logger)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	//-----http service初始化----
	pushService := service.NewPushService(notifyProducerChan, interceptor, producerRepo, consumeRepo, rg, ch)
	pushHandler := web.NewPushHandler(pushService)
	server := ioc.InitWebServer(pushHandler)
	go func() {
		er := server.Run(":8085")
		if er != nil {
			return
		}
	}()
	<-ch
	gs.Shutdown()
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
