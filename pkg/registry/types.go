package registry

import "context"

type Registry interface {
	// Register 向注册中心注册一个服务
	Register(ctx context.Context, ins ServiceInstance) error
	// UnRegister 取消注册
	UnRegister(ctx context.Context, ins ServiceInstance) error
	// Subscribe 订阅某服务，及时接收该服务的变化通知
	Subscribe(serviceName string) (<-chan Event, error)
	// ListService 查看某服务的实例列表
	ListService(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	// Close 关闭订阅，取消监听
	Close() error
}

// ServiceInstance 代表一个实例
type ServiceInstance struct {
	Address     string `json:"address"`
	ServiceName string `json:"service_name"`
	Weight      uint32 `json:"weight"`
	Note        string `json:"note"`
}

type EventType int

const (
	EventTypeUnknown EventType = iota
	EventTypePut
	EventTypeDelete
	EventTypeUpdate
)

type Event struct {
	Type     EventType
	Instance ServiceInstance
}
