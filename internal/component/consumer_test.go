package component

import (
	"context"
	"log"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/config/deviceconfig"
	"github.com/xuhaidong1/offlinepush/internal/component/mocks"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"github.com/xuhaidong1/offlinepush/internal/repository"
	"github.com/xuhaidong1/offlinepush/internal/repository/dao"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	etcd2 "github.com/xuhaidong1/offlinepush/pkg/registry/etcd"

	"github.com/stretchr/testify/assert"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"go.uber.org/mock/gomock"
)

// go install github.com/golang/mock/mockgen@v1.6.0
// mockgen github.com/IBM/sarama SyncProducer >mocks/mock_sync_producer.go
var pushedCount int32

func TestConsumer(t *testing.T) {
	num := 1000
	reboot := pushconfig.PushMap["reboot"]
	task := pushconfig.PushConfig{
		Topic:          reboot.Topic,
		DeviceTypeList: reboot.DeviceTypeList,
		Cron:           reboot.Cron,
		Num:            num,
	}
	taskCh := make(chan pushconfig.PushConfig)
	ctx := context.Background()
	// 消费者时间计算不好算
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	crn := func(ctrl *gomock.Controller) Croner {
		c := mocks.NewMockCroner(ctrl)
		c.EXPECT().Subscribe(gomock.Any()).Return(taskCh)
		return c
	}(ctrl)

	hopeCount := int32(num * len(task.DeviceTypeList))
	//--初始化三方依赖
	etcd := ioc.InitEtcd()
	kafka := ioc.InitKafka()
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

	mockWorker := NewMockWorker()
	repo := repository.NewRepository(dao.NewGormDeviceDAO(ioc.InitDB()))
	hasher := NewConsistentHash(config.StartConfig.Register.ServiceName, config.StartConfig.Register.PodName, deviceconfig.Devices, rg)
	kafkaProducer := ioc.NewSyncProducer(kafka)
	producer := NewProducer(repo, kafkaProducer, hasher, crn)
	producer.Run(ctx)
	consumer := NewConsumer(kafka, mockWorker)
	consumer.Run(ctx)

	time.Sleep(time.Second * 3)
	taskCh <- task
	now := time.Now()
	quit := make(chan struct{})
	go func() {
		workCtx, cancel := context.WithTimeout(ctx, time.Minute*10)
		defer cancel()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				log.Println("留的时间不够")
				quit <- struct{}{}
				return
			case <-ticker.C:
				if atomic.CompareAndSwapInt32(&pushedCount, hopeCount, hopeCount) {
					log.Println("消费花费时间:", time.Since(now).String())
					quit <- struct{}{}
					return
				}
			}
		}
	}()
	<-quit
	// 消费消息的数量校验
	assert.Equal(t, hopeCount, pushedCount)
	log.Println(pushedCount)
	log.Println("okokokok")
}

type MockWorker struct{}

func NewMockWorker() Worker {
	return &MockWorker{}
}

func (w *MockWorker) Work(args Args) {
	err := w.Push(args.Msg)
	args.Wg.Done()
	if err != nil {
		// todo 重试逻辑 同步转异步
	}
}

func (w *MockWorker) Push(msg domain.Message) error {
	time.Sleep(200 * time.Millisecond)
	atomic.AddInt32(&pushedCount, 1)
	return nil
}

func TestConsumerGroup(t *testing.T) {
	kafka := ioc.InitKafka()
	mockWorker := NewMockWorker()
	consumer := NewConsumer(kafka, mockWorker)
	consumer.Run(context.Background())
	// now := time.Now()
	// var hopeCount int32 = 24000
	quit := make(chan struct{})
	//go func() {
	//	workCtx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	//	defer cancel()
	//	ticker := time.NewTicker(100 * time.Millisecond)
	//	defer ticker.Stop()
	//	for {
	//		select {
	//		case <-workCtx.Done():
	//			log.Println("留的时间不够")
	//			quit <- struct{}{}
	//			return
	//		case <-ticker.C:
	//			if atomic.CompareAndSwapInt32(&pushedCount, hopeCount, hopeCount) {
	//				log.Println("消费花费时间:", time.Since(now).String())
	//				quit <- struct{}{}
	//				return
	//			}
	//		}
	//	}
	//}()
	<-quit
}
