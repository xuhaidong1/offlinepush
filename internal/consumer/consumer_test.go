package consumer

import (
	"context"
	"log"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/offlinepush/internal/domain"
	"golang.org/x/sync/errgroup"
)

type Cmer struct{}

type Puller struct{}

var q = queue.NewConcurrentBlockingQueue[int](10000)

func (p *Puller) Pull(ctx context.Context, biz string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			//data := redis.lrange
			//if nomessage{
			//	q.Enqueue(ctx,-1)
			//	return
			//}
			data := func() (res []int) {
				for i := 0; i < 40; i++ {
					res = append(res, i)
				}
				res = append(res, -1)
				return
			}()
			for i, d := range data {
				err := q.Enqueue(ctx, d)
				if err != nil {
					WriteBackLeftMessage(data[i:])
					return
				}
			}
		}
	}
}

func WriteBackLeftMessage(data []int) {
	for {
		if q.IsEmpty() {
			break
		}
		re, err := q.Dequeue(context.Background())
		if err != nil {
			return
		}
		if re != -1 {
			log.Println(re)
		}
	}
	for _, d := range data {
		if d != -1 {
			log.Println(d)
		}
	}
}

func (c *Cmer) Cons(ctx, dequeueCtx context.Context, biz string, finished chan struct{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-finished:
			return nil
		default:
			msg, err := q.Dequeue(dequeueCtx)
			if err != nil {
				// 这里不知道是什么原因取消的，
				// 如果是上级取消（服务关闭，暂停）会走入case <-ctx.Done():
				// 如果是消费完了没有消息了，会走入case <-finished:
				continue
			}
			if msg == -1 {
				close(finished)
				continue
			}
			// lg.Println(msg)
		}
	}
}

var (
	cmer   = &Cmer{}
	puller = &Puller{}
)

func TestConsumer_Push(t *testing.T) {
	eg := errgroup.Group{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finish := make(chan struct{})
	dequeueCtx, dCancel := context.WithCancel(ctx)
	go func() {
		<-finish
		dCancel()
	}()
	//go func() {
	//	time.Sleep(time.Millisecond * 5)
	//	cancel()
	//}()
	go puller.Pull(ctx, "biz")
	for i := 0; i < 10; i++ {
		eg.Go(func() error {
			return cmer.Cons(ctx, dequeueCtx, "biz", finish)
		})
	}
	err := eg.Wait()
	if err != nil {
		return
		// 由pull写回leftmsg
	}
}

//var lg = NewPushLogger()
//
//func NewPushLogger() *log.Logger {
//	currentDir, err := os.Getwd()
//	if err != nil {
//		panic(err)
//	}
//	logFilePath := filepath.Join(currentDir, "push.log")
//	// os.O_APPEND
//	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0o666)
//	if err != nil {
//		panic(err)
//	}
//	return log.New(logFile, "pushlogger:", log.LstdFlags)
//}

func dd(cmd redis.Cmdable, biz string, devices []domain.Device) {
	for _, d := range devices {
		taskKey := biz + ":" + d.Type
		cmd.LPush(context.Background(), taskKey, d.ID)
		cmd.SAdd(context.Background(), biz, taskKey)
	}
}
