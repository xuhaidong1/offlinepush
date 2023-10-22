//go:build k8s

package config

import "time"

var StartConfig = Config{
	Redis: RedisConfig{
		Addr:                   "offlinepush-redis:6579",
		Password:               "",
		ConsumerLeftMessageKey: "offlinepush:consumer:leftmessage",
		ProducerLeftTaskKey:    "offlinepush:producer:lefttask",
	},
	Etcd: EtcdConfig{Addr: "offlinepush-etcd:2379"},
	Register: RegisterConfig{
		ServiceName:    "offlinepush-service",
		ConsumerPrefix: "offlinepush-consumer",
		PodName:        GetPodName(),
	},
	Lock: LockConfig{
		LockKey:         "offlinepush-lock",
		Expiration:      time.Second * 6,
		RefreshInterval: time.Second * 5,
		Timeout:         time.Millisecond * 200,
	},
}
