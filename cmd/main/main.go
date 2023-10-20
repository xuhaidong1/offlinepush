package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/cron"

	"github.com/hashicorp/golang-lru/v2/simplelru"
	cond "github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/go-generic-tools/ecache/memory/lru"
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
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 运行要加上 --config=conf/dev.toml
// 并且可以开启环境变量 EGO_DEBUG=true
func main() {
	//--初始化三方依赖
	etcd := ioc.InitEtcd()
	rdb := ioc.InitRedis()
	PingRedis(rdb)
	PingEtcd(etcd)
	ctx, cancel := context.WithCancel(context.Background())
	simpleLru, err := simplelru.NewLRU[string, any](100000, func(key string, value any) {})
	localCache := lru.NewCache(simpleLru)

	//----实例注册-------
	podName := GetPodName()
	rg, err := etcd2.NewRegistry(etcd)
	if err != nil {
		panic(err)
	}
	ins, err := RegisterService(ctx, rg)
	if err != nil {
		panic(err)
	}
	// 消费者只订阅自己
	consumeInsName := config.StartConfig.Register.ServiceName + GetPodName()
	notifyChan, err := rg.Subscribe(consumeInsName)
	if err != nil {
		panic(err)
	}

	//----消费者初始化-----------------
	consumeRepo := consumerrepo.NewConsumerRepository(rdb, localCache)
	consumer.NewConsumeController(ctx, notifyChan, consumeRepo)

	//----分布式锁管理----
	lockClient := redis_lock.NewClient(rdb)
	// cond用于管理生产者/负载均衡组件的启动停止
	startCond := cond.NewCondAtomic(&sync.Mutex{})
	stopCond := cond.NewCondAtomic(&sync.Mutex{})
	//----负载均衡器初始化----
	loadbalancer.NewLoadBalanceController(ctx, startCond, stopCond)
	//----生产者控制器初始化------
	producerRepo := producerrepo.NewProducerRepository(rdb, localCache)
	taskChan := make(chan pushconfig.PushConfig, 10)
	producer.NewProduceController(ctx, taskChan, startCond, stopCond, producerRepo)
	//-----定时任务控制器初始化----
	cronController := cron.NewCronController(taskChan)
	//-----锁控制器初始化--------
	lockController := lock.NewLockController(lockClient, podName, startCond, stopCond, config.StartConfig.Lock)
	go lockController.Start(ctx)
	//----优雅关闭初始化------
	gs := &GracefulClose{
		Cancel:         cancel,
		Registry:       rg,
		instance:       ins,
		cronController: cronController,
	}
	log.Println("alive")
	time.Sleep(time.Second * 5)
	gs.Shutdown()
	time.Sleep(time.Second)
}

type GracefulClose struct {
	Cancel         context.CancelFunc
	Registry       registry.Registry
	instance       registry.ServiceInstance
	cronController *cron.CronController
}

func (s *GracefulClose) Shutdown() {
	s.Cancel()
	err := s.Registry.UnRegister(context.Background(), s.instance)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Registry.Close()
	if err != nil {
		log.Fatal(err)
	}
	s.cronController.Stop()
}

func GetPodName() string {
	podName, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprintf("Error getting Pod name: %v", err))
	}
	return podName
}

func PingRedis(redisCmd redis.Cmdable) {
	redisCmd.Set(context.Background(), "key1", "val1", time.Minute)
	result, err := redisCmd.Get(context.Background(), "key1").Result()
	if err != nil {
		panic(err)
	}
	if result != "val1" {
		panic("值不对")
	}
}

func PingEtcd(etcd *clientv3.Client) {
	// 创建一个键值对的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 执行 SET 操作
	_, err := etcd.Put(ctx, "mykey", "myvalue")
	if err != nil {
		log.Fatal(err)
	}

	// 执行 GET 操作
	resp, err := etcd.Get(ctx, "mykey")
	if err != nil {
		log.Fatal(err)
	}

	// 打印获取到的值
	for _, kv := range resp.Kvs {
		fmt.Printf("Key: %s, Value: %s\n", kv.Key, kv.Value)
	}
}

func RegisterService(ctx context.Context, rg registry.Registry) (registry.ServiceInstance, error) {
	ins := registry.ServiceInstance{
		Address:     GetPodName(),
		ServiceName: config.StartConfig.Register.ServiceName + GetPodName(),
		Weight:      10000,
	}
	// 这个注册的是服务，生产者调用服务列表的key：config.StartConfig.Register.ServiceName，根据服务weight确定通知的实例。
	err := rg.Register(ctx, ins)
	if err != nil {
		return registry.ServiceInstance{}, err
	}
	return ins, err
}
