package interceptor

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
	"go.uber.org/zap"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"
)

var (
	ErrNobiz             = errors.New("没这个业务")
	ErrRepeatedOperation = errors.New("重复操作")
)

type Interceptor struct {
	// 一个开关集合，key：biz，val：是否执行任务；可被http操控，默认为true
	Switch map[string]bool
	// 开始推送，如果有剩余任务，应该立即通知消费者开始消费；然而不需要立即通知生产者，生产者按时间表来
	mutex  *sync.RWMutex
	rg     registry.Registry
	logger *zap.Logger
}

func NewInterceptor(ctx context.Context, rg registry.Registry, logger *zap.Logger) *Interceptor {
	mp := make(map[string]bool)
	for k := range pushconfig.PushMap {
		mp[k] = true
	}
	i := &Interceptor{
		Switch: mp,
		mutex:  &sync.RWMutex{},
		rg:     rg,
		logger: logger,
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go i.watchCh(ctx, wg)
	wg.Wait()
	return i
}

func (i *Interceptor) ResumeBiz(ctx context.Context, biz string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	_, ok := i.Switch[biz]
	if !ok {
		return ErrNobiz
	}
	if i.Switch[biz] == false {
		// i.Switch[biz] = true
		js, _ := json.Marshal(pair{
			Biz:    biz,
			Status: true,
		})
		return i.rg.Register(ctx, registry.ServiceInstance{
			Address:     "interceptor",
			ServiceName: config.StartConfig.Register.InterceptorKey,
			Weight:      0,
			Note:        string(js),
		})
	}
	return ErrRepeatedOperation
}

func (i *Interceptor) PauseBiz(ctx context.Context, biz string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	_, ok := i.Switch[biz]
	if !ok {
		return ErrNobiz
	}
	if i.Switch[biz] == true {
		js, _ := json.Marshal(pair{
			Biz:    biz,
			Status: false,
		})
		return i.rg.Register(ctx, registry.ServiceInstance{
			Address:     "interceptor",
			ServiceName: config.StartConfig.Register.InterceptorKey,
			Weight:      0,
			Note:        string(js),
		})
	}
	return ErrRepeatedOperation
}

// Permit 对于生产者，已经在生产的就不取消了，以后这个业务的生产信号会被拦截，不能开始生产；
// 对于消费者，已经在消费的要停止消费，接受到了新任务也不会开始消费。
func (i *Interceptor) Permit(biz string) bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.Switch[biz]
}

func (i *Interceptor) GetMap() map[string]bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.Switch
}

// 监听etcd的interceptor配置变更
func (i *Interceptor) watchCh(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()
	ch, err := i.rg.Subscribe(config.StartConfig.Register.InterceptorKey)
	if err != nil {
		panic(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			if event.Type == registry.EventTypePut {
				pairStr := event.Instance.Note
				var p pair
				_ = json.Unmarshal([]byte(pairStr), &p)
				i.mutex.Lock()
				i.Switch[p.Biz] = p.Status
				i.mutex.Unlock()
				i.logger.Info("Interceptor", zap.Bool(p.Biz, p.Status))
			}
		}
	}
}

type pair struct {
	Biz    string `json:"biz"`
	Status bool   `json:"status"`
}
