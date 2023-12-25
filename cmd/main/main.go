package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/xuhaidong1/offlinepush/internal/prometheus"

	"github.com/gin-gonic/gin"
	"github.com/xuhaidong1/offlinepush/config/deviceconfig"
	"github.com/xuhaidong1/offlinepush/internal/component"
	"github.com/xuhaidong1/offlinepush/internal/repository"
	"github.com/xuhaidong1/offlinepush/internal/repository/dao"

	"github.com/xuhaidong1/offlinepush/internal/service"
	"github.com/xuhaidong1/offlinepush/web"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	etcd2 "github.com/xuhaidong1/offlinepush/pkg/registry/etcd"
)

func main() {
	//--初始化三方依赖
	db := ioc.InitDB()
	etcd := ioc.InitEtcd()
	kafka := ioc.InitKafka()
	prometheus.InitPrometheus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//----实例注册-------
	rg, err := etcd2.NewRegistry(etcd)
	if err != nil {
		panic(err)
	}
	err = rg.Register(ctx, registry.ServiceInstance{
		Address:     config.StartConfig.Register.PodName,
		ServiceName: config.StartConfig.Register.ServiceName,
	})
	if err != nil {
		panic(err)
	}
	//-------拦截器创建------
	interceptor := component.NewInterceptor(ctx, rg, ioc.Logger)
	//----消费者初始化-----------------
	counter := component.NewCounter(rg)
	counter.Run(ctx)
	worker := component.NewWorker(counter)
	repo := repository.NewRepository(dao.NewGormDeviceDAO(db))
	cron := component.NewCron()
	hasher := component.NewConsistentHash(config.StartConfig.Register.ServiceName, config.StartConfig.Register.PodName, deviceconfig.Devices, rg)
	kafkaProducer := ioc.NewSyncProducer(kafka)
	producer := component.NewProducer(repo, kafkaProducer, hasher, cron, counter, rg)
	producer.Run(ctx)
	consumer := component.NewConsumer(kafka, worker)
	consumer.Run(ctx)
	//----优雅关闭初始化------
	//gs := NewGracefulShutdown(cancel, rg, ins, ioc.Logger)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	//-----http service初始化----
	pushService := service.NewPushService(interceptor, rg, counter)
	pushHandler := web.NewPushHandler(pushService)
	server := InitWebServer(pushHandler)
	go func() {
		er := server.Run(":8085")
		if er != nil {
			return
		}
	}()
	<-ch
	// gs.Shutdown()
}

func InitWebServer(hdl *web.PushHandler) *gin.Engine {
	server := gin.Default()
	hdl.RegisterRoutes(server)
	return server
}
