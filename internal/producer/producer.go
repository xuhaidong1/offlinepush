package producer

import (
	"context"
	"log"
	"sync"

	"github.com/xuhaidong1/offlinepush/internal/producer/repository"
)

type Producer struct {
	repo   repository.ProducerRepository
	cancel context.CancelFunc
}

func NewProducer(ctx context.Context, repo repository.ProducerRepository) *Producer {
	produceCtx, cancel := context.WithCancel(ctx)
	p := &Producer{
		cancel: cancel,
		repo:   repo,
	}
	go p.Produce(produceCtx)
	return p
}

func (w *Producer) Produce(ctx context.Context) {
	once := sync.Once{}
	for {
		select {
		case <-ctx.Done():
			log.Println("producer work结束")
			return
		default:
			once.Do(func() {
				log.Println("producer working")
			})

		}
	}
}

func (w *Producer) Stop() {
	w.cancel()
}
