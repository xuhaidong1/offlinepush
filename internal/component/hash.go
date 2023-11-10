package component

import (
	"context"
	"sync"
	"time"

	"github.com/serialx/hashring"
	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/pkg/registry"
)

type ConsistentHash struct {
	serviceName string
	podName     string
	ring        *hashring.HashRing
	// 储存所有设备类型
	types []string
	// 本实例负责的设备类型
	responsibleKeysCh chan []string
	// 储存所有的服务节点
	nodes    map[string]struct{}
	registry registry.Registry
	logger   logx.Logger
}

func NewConsistentHash(serviceName, podName string, types []string, registry registry.Registry) *ConsistentHash {
	c := &ConsistentHash{
		serviceName:       serviceName,
		podName:           podName,
		types:             types,
		responsibleKeysCh: make(chan []string),
		registry:          registry,
		logger:            ioc.Loggerx.With(logx.Field{Key: "component", Value: "ConsistentHash"}),
	}
	ctx := context.Background()
	nodes, err := c.listNodes(ctx)
	if err != nil {
		panic(err)
	}
	c.ring = hashring.New(nodes)
	c.setNodes(nodes)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go c.watchNodes(ctx, serviceName, wg)
	wg.Wait()

	time.AfterFunc(time.Minute*2, func() {
		ns, er := c.listNodes(ctx)
		if er != nil {
			c.logger.Error("listNode失败", logx.Error(err))
		}
		c.setNodes(ns)
	})
	return c
}

func (c *ConsistentHash) Subscribe(ctx context.Context) <-chan []string {
	ch := make(chan []string)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case keys := <-c.responsibleKeysCh:
				ch <- keys
			}
		}
	}()
	return ch
}

func (c *ConsistentHash) GetResponsibleKeys() []string {
	return c.hash(c.podName)
}

func (c *ConsistentHash) watchNodes(ctx context.Context, servicename string, wg *sync.WaitGroup) {
	wg.Done()
	ch, err := c.registry.Subscribe(servicename)
	if err != nil {
		panic(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			if event.Type == registry.EventTypePut {
				name := event.Instance.Address
				c.OnAddNode(name)
			}
			if event.Type == registry.EventTypeDelete {
				name := event.Instance.Address
				c.OnDelNode(name)
			}
		}
	}
}

func (c *ConsistentHash) OnAddNode(name string) {
	c.nodes[name] = struct{}{}
	ring := c.ring.AddNode(name)
	c.ring = ring
	select {
	case c.responsibleKeysCh <- c.hash(c.podName):
	default:
	}
}

func (c *ConsistentHash) OnDelNode(name string) {
	delete(c.nodes, name)
	ring := c.ring.RemoveNode(name)
	c.ring = ring
	select {
	case c.responsibleKeysCh <- c.hash(c.podName):
	default:
	}
}

func (c *ConsistentHash) OnSetNodes(nodes []string) {
	ring := hashring.New(nodes)
	c.ring = ring
	select {
	case c.responsibleKeysCh <- c.hash(c.podName):
	default:
	}
}

// 哈希，返回pod实例需要负责的设备类型
func (c *ConsistentHash) hash(pod string) []string {
	res := make([]string, 0)
	for _, t := range c.types {
		node, _ := c.ring.GetNode(t)
		if node == pod {
			res = append(res, t)
		}
	}
	return res
}

func (c *ConsistentHash) getNodes() []string {
	keys := make([]string, 0, len(c.nodes))
	for k := range c.nodes {
		keys = append(keys, k)
	}
	return keys
}

func (c *ConsistentHash) setNodes(nodes []string) {
	nodesSet := make(map[string]struct{})
	for _, n := range nodes {
		nodesSet[n] = struct{}{}
	}
	c.nodes = nodesSet
	c.OnSetNodes(nodes)
}

func (c *ConsistentHash) listNodes(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	instances, err := c.registry.ListService(ctx, c.serviceName)
	if err != nil {
		return []string{}, err
	}
	nodes := make([]string, 0, len(instances))
	for _, ins := range instances {
		nodes = append(nodes, ins.Address)
	}
	return nodes, nil
}
