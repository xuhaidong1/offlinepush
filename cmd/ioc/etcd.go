package ioc

import (
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
	})
	return etcdClient
}
