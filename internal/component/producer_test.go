package component

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/deviceconfig"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/component/mocks"
	"github.com/xuhaidong1/offlinepush/internal/repository"
	"github.com/xuhaidong1/offlinepush/internal/repository/dao"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	"go.uber.org/mock/gomock"
)

// go install github.com/golang/mock/mockgen@v1.6.0
// mockgen github.com/IBM/sarama SyncProducer >mocks/mock_sync_producer.go

func TestProducer(t *testing.T) {
	servicename := "offlinepush"
	pod := "pod0"
	num := 100000
	reboot := pushconfig.PushMap["reboot"]
	task := pushconfig.PushConfig{
		Topic:          reboot.Topic,
		DeviceTypeList: reboot.DeviceTypeList,
		Cron:           reboot.Cron,
		Num:            num,
	}
	taskCh := make(chan pushconfig.PushConfig)
	ctrl := gomock.NewController(t)
	var msgCount int32
	defer ctrl.Finish()

	rg := func(ctrl *gomock.Controller) registry.Registry {
		r := mocks.NewMockRegistry(ctrl)
		r.EXPECT().ListService(gomock.Any(), gomock.Any()).Return([]registry.ServiceInstance{
			{Address: pod, ServiceName: servicename},
		}, nil)
		ch := make(chan registry.Event)
		r.EXPECT().Subscribe(gomock.Any()).Return(ch, nil)
		return r
	}(ctrl)

	crn := func(ctrl *gomock.Controller) Croner {
		c := mocks.NewMockCroner(ctrl)
		c.EXPECT().Subscribe(gomock.Any()).Return(taskCh)
		return c
	}(ctrl)

	kafkaProducer := func(ctrl *gomock.Controller) sarama.SyncProducer {
		p := mocks.NewMockSyncProducer(ctrl)
		p.EXPECT().SendMessages(gomock.Any()).Do(func(msgs []*sarama.ProducerMessage) {
			for i := 0; i < len(msgs); i++ {
				//var m domain.Message
				//_ = json.Unmarshal(msgs[i].Metadata.([]byte), &m)
				//log.Println(m)
				atomic.AddInt32(&msgCount, 1)
			}
		}).Return(nil).AnyTimes()
		return p
	}(ctrl)
	repo := repository.NewRepository(dao.NewGormDeviceDAO(ioc.InitDB()))
	hasher := NewConsistentHash(servicename, pod, deviceconfig.Devices, rg)
	NewProducer(repo, kafkaProducer, hasher, crn, NewCounter(rg), rg)
	time.Sleep(time.Second * 2)
	taskCh <- task
	time.Sleep(time.Second * 5)
	assert.Equal(t, int32(num*len(task.DeviceTypeList)), msgCount)
}
