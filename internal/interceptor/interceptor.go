package interceptor

import (
	"errors"
	"sync"

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
	mutex *sync.RWMutex
}

func NewInterceptor() *Interceptor {
	mp := make(map[string]bool)
	for k := range pushconfig.PushMap {
		mp[k] = true
	}
	return &Interceptor{Switch: mp, mutex: &sync.RWMutex{}}
}

func (i *Interceptor) ResumeBiz(biz string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	_, ok := i.Switch[biz]
	if !ok {
		return ErrNobiz
	}
	if i.Switch[biz] == false {
		i.Switch[biz] = true
		return nil
	}
	return ErrRepeatedOperation
}

func (i *Interceptor) PauseBiz(biz string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	_, ok := i.Switch[biz]
	if !ok {
		return ErrNobiz
	}
	if i.Switch[biz] == true {
		i.Switch[biz] = false
		return nil
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
