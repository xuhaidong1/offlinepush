package producer

import (
	"context"
	"log"
	"strconv"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
)

type Producer struct {
	repo repository.ProducerRepository
}

func NewProducer(repo repository.ProducerRepository) *Producer {
	p := &Producer{
		repo: repo,
	}
	return p
}

func (p *Producer) Produce(ctx context.Context, cfg pushconfig.PushConfig) {
	log.Println("producer working")
	for _, deviceType := range cfg.DeviceTypeList {
		for i := 1; i <= 300; i++ {
			select {
			case <-ctx.Done():
				log.Println("producer work结束")
				err := p.repo.WriteBackLeftTask(context.Background(), cfg.Business.Name)
				if err != nil {
					log.Fatalln("producer 写回受影响任务失败")
				}
				return
			default:
				msg := domain.Message{
					Business: cfg.Business,
					Device: domain.Device{
						Type: deviceType,
						ID:   deviceType + strconv.Itoa(i),
					},
				}
				err := p.repo.Store(ctx, msg)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}
	}
	err := p.repo.WriteBack(ctx, cfg.Business.Name)
	if err != nil {
		log.Fatalln("producer writeback fail")
	}
}
