package component

import (
	"sync"
	"time"

	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type Worker interface {
	Push(msg domain.Message) error
	Work(args Args)
}

type worker struct {
	*Counter
}

func NewWorker(counter *Counter) Worker {
	return &worker{counter}
}

func (w *worker) Work(args Args) {
	err := w.Push(args.Msg)
	w.Counter.IncrPushedCount()
	args.Wg.Done()
	if err != nil {
		// todo 重试逻辑 同步转异步
	}
}

func (w *worker) Push(msg domain.Message) error {
	time.Sleep(200 * time.Millisecond)
	// ioc.PushLogger.Printf("push %v\n", msg)
	return nil
}

type Args struct {
	Msg domain.Message
	Wg  *sync.WaitGroup
}
