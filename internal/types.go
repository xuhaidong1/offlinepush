package internal

import (
	"context"
	"sync"
)

// Controller 生产/消费/负载均衡worker的控制器，以下每个方法调用时都需要开启goroutine
type Controller interface {
	ListenEngageCond(ctx context.Context, wg *sync.WaitGroup)
	ListenDismissCond(ctx context.Context, wg *sync.WaitGroup)
}

type Worker interface {
	Work(ctx context.Context)
	Stop()
}
