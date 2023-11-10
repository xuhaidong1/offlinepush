package consumer

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/xuhaidong1/offlinepush/internal/interceptor"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/internal/consumer/repository"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"go.uber.org/zap"
)

var (
	NoMessage = repository.NoMessage
	Paused    = errors.New("paused")
)

type WorkerV0 struct {
	repo        repository.ConsumerRepository
	interceptor *interceptor.Interceptor
	pushLogger  *log.Logger
	logger      *zap.Logger
}

func NewWorkerV0(repo repository.ConsumerRepository, interceptor *interceptor.Interceptor) *WorkerV0 {
	return &WorkerV0{
		repo:        repo,
		interceptor: interceptor,
		pushLogger:  ioc.PushLogger,
		logger:      ioc.Logger,
	}
}

func (c *WorkerV0) WorkOld(ctx context.Context, bizName string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !c.interceptor.Permit(bizName) {
				return Paused
			}
			// todo 改造这里 分片获取数据 一次获取一片数据：biz：devicetype 到本地缓存，
			// todo 本地缓存消费完成后取下一片数据 如果被pause/cancel 需要把这一片的数据写回redis
			msg, err := c.repo.GetMessageFromStorage(ctx, bizName)
			if err != nil && !errors.Is(err, NoMessage) {
				c.logger.Error("WorkerV0", zap.String("WorkOld", "GetMessage"), zap.Error(err))
				return nil
			}
			if errors.Is(err, NoMessage) {
				return nil
			}
			// 在这里mock推送。。
			c.Push(msg)
		}
	}
}

func (c *WorkerV0) Work(ctx, dequeueCtx context.Context, biz string, finished, queueReady chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-queueReady:
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-finished:
			return nil
		default:
			if !c.interceptor.Permit(biz) {
				return Paused
			}
			msg, err := c.repo.GetMessage(dequeueCtx, biz)
			if err != nil {
				// 这里不知道是什么原因取消的，所以continue去返回原因
				// 如果是上级取消（服务关闭，暂停）会走入case <-ctx.Done():
				// 如果是消费完了没有消息了，会走入case <-finished:
				continue
			}
			if domain.IsEOF(msg) {
				// 后面没消息了，广播告诉其它消费者goroutine退出
				close(finished)
				continue
			}
			c.Push(msg)
		}
	}
}

func (c *WorkerV0) Push(msg domain.Message) {
	time.Sleep(50 * time.Millisecond)
	c.pushLogger.Printf("push %v\n", msg)
}
