package producer

import (
	"context"
	"strconv"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/deviceconfig"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
	"go.uber.org/zap"
)

type Producer struct {
	repo   repository.ProducerRepository
	logger *zap.Logger
}

func NewProducer(repo repository.ProducerRepository) *Producer {
	p := &Producer{
		repo:   repo,
		logger: ioc.Logger,
	}
	return p
}

func (p *Producer) Produce(ctx context.Context, cfg pushconfig.PushConfig) {
	p.logger.Info("Producer",
		zap.String("biz", cfg.Business.Name),
		zap.String("status", "start"))
	for _, deviceType := range cfg.DeviceTypeList {
		n := deviceconfig.DeviceMap[deviceType]
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				p.logger.Info("Producer",
					zap.String("biz", cfg.Business.Name),
					zap.String("status", "canceled"))
				err := p.repo.WriteBackLeftTask(context.Background(), cfg.Business.Name)
				if err != nil {
					p.logger.Error("Producer", zap.String("Produce", "WriteBackLeftTask"), zap.Error(err))
				}
				p.logger.Info("Producer", zap.String("WriteBackLeftTask", "ok"))
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
					p.logger.Error("Producer", zap.String("Produce", "Store"), zap.Error(err))
				}
			}
		}
	}
	err := p.repo.WriteBack(ctx, cfg.Business.Name)
	if err != nil {
		p.logger.Error("Producer", zap.String("Produce", "WriteBack"), zap.Error(err))
	}
	p.logger.Info("Producer",
		zap.String("biz", cfg.Business.Name),
		zap.String("status", "completed"))
}
