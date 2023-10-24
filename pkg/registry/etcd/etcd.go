package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/xuhaidong1/offlinepush/pkg/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var typesMap = map[mvccpb.Event_EventType]registry.EventType{
	mvccpb.PUT:    registry.EventTypePut,
	mvccpb.DELETE: registry.EventTypeDelete,
}

type Registry struct {
	client *clientv3.Client
	sess   *concurrency.Session
	// 自己退出时，要取消订阅别人
	watchCancel []func()
	mutex       sync.RWMutex
	close       chan struct{}
}

func NewRegistry(client *clientv3.Client) (*Registry, error) {
	sess, err := concurrency.NewSession(client)
	if err != nil {
		return nil, err
	}
	return &Registry{
		client: client,
		sess:   sess,
	}, nil
}

// Register 向注册中心注册一个服务
func (r *Registry) Register(ctx context.Context, ins registry.ServiceInstance) error {
	// 准备key val 和租约
	instanceKey := fmt.Sprintf("%s/%s", ins.ServiceName, ins.Address)
	val, err := json.Marshal(ins)
	if err != nil {
		return err
	}
	_, err = r.client.Put(ctx, instanceKey, string(val), clientv3.WithLease(r.sess.Lease()))
	if err != nil {
		return err
	}
	return nil
}

// UnRegister 取消注册
func (r *Registry) UnRegister(ctx context.Context, ins registry.ServiceInstance) error {
	instanceKey := fmt.Sprintf("%s/%s", ins.ServiceName, ins.Address)
	_, err := r.client.Delete(ctx, instanceKey)
	return err
}

// Subscribe 订阅某key，及时接收该key的变化通知
func (r *Registry) Subscribe(key string) (<-chan registry.Event, error) {
	// 调一次subscribe就会产生一个watchchan 就会产生一个goroutine 要防止泄露和及时关闭
	// watch的是service的所有实例
	ctx, cancel := context.WithCancel(context.Background())
	ctx = clientv3.WithRequireLeader(ctx)
	r.mutex.Lock() // append要保护起来 并发安全
	r.watchCancel = append(r.watchCancel, cancel)
	r.mutex.Unlock()
	// Watch里面的ctx calcel掉， watch就会退出
	watchCh := r.client.Watch(ctx, key, clientv3.WithPrefix())
	res := make(chan registry.Event)
	go func() {
		for resp := range watchCh {
			if resp.Canceled {
				return
			}
			if resp.Err() != nil {
				continue
			}
			for _, event := range resp.Events {
				var ins registry.ServiceInstance
				err := json.Unmarshal(event.Kv.Value, &ins)
				if err != nil {
					// 发一个没有意义的事件 或者是错误事件
					// 这个地方可能阻塞，阻塞了goroutine就泄露掉了
					select {
					case res <- registry.Event{}: // 不能用select-default包裹住这个，因为这个可能会把有用的事件冲掉
					case <-ctx.Done():
						return
					}
					continue
				}
				// 这个地方可能阻塞，阻塞了goroutine就泄露掉了
				select {
				case res <- registry.Event{
					Type:     typesMap[event.Type],
					Instance: ins,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return res, nil
}

// ListService 查看某服务的实例列表
func (r *Registry) ListService(ctx context.Context, serviceName string) ([]registry.ServiceInstance, error) {
	resp, err := r.client.Get(ctx, serviceName, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	instances := make([]registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var ins registry.ServiceInstance
		err = json.Unmarshal(kv.Value, &ins)
		if err != nil {
			// continue  这种做法忽略了部分解析失败 协议版本升级可能会导致这样
			return nil, err
		}
		instances = append(instances, ins)
	}
	return instances, nil
}

// Close 关闭订阅，取消监听
func (r *Registry) Close() error {
	r.mutex.RLock()
	watchCancel := r.watchCancel
	r.mutex.RUnlock()
	for _, cancel := range watchCancel {
		cancel()
	}
	// close(r.close)关闭这个chan 达到广播的效果，监听close的地方就会立即拿到0值 退出
	err := r.sess.Close()
	if err != nil {
		return err
	}
	err = r.client.Close()
	return err
}
