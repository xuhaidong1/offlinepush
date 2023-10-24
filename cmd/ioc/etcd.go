package ioc

import (
	"context"
	"github.com/xuhaidong1/offlinepush/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

var (
	etcdClient   *clientv3.Client
	etcdInitOnce sync.Once
)

func InitEtcd() *clientv3.Client {
	var err error
	etcdInitOnce.Do(func() {
		etcdClient, err = clientv3.New(clientv3.Config{
			Endpoints:   []string{config.StartConfig.Etcd.Addr},
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			panic(err)
		}
		PingEtcd(etcdClient)
	})
	return etcdClient
}

func PingEtcd(etcd *clientv3.Client) {
	// 创建一个键值对的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 执行 SET 操作
	_, err := etcd.Put(ctx, "mykey", "myvalue")
	if err != nil {
		panic(err)
	}
	// 执行 GET 操作
	_, err = etcd.Get(ctx, "mykey")
	if err != nil {
		panic(err)
	}
	// 打印获取到的值
	//for _, kv := range resp.Kvs {
	//	fmt.Printf("Key: %s, Value: %s\n", kv.Key, kv.Value)
	//}
}
