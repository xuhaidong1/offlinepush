//go:build !k8s

package config

import "time"

var StartConfig = Config{
	Redis: RedisConfig{
		Addr:                "localhost:6579",
		Password:            "",
		ConsumerLeftTaskKey: "offlinepush:consumer:leftmessage",
		ProducerLeftTaskKey: "offlinepush:producer:lefttask",
	},
	Etcd: EtcdConfig{Addr: "localhost:2379"},
	Register: RegisterConfig{
		ServiceName:    "offlinepush-local",
		ConsumerPrefix: "offlinepush-consumer",
		InterceptorKey: "offlinepush-interceptor",
		ManualTaskKey:  "offlinepush-manual",
		ReadCountKey:   "offlinepush-msg-count",
		WriteCountKey:  "offlinepush-msg-count-write",
		PodName:        GetPodName(),
	},
	Lock: LockConfig{
		LockKey:         "offlinepush-lock",
		Expiration:      time.Second * 6,
		RefreshInterval: time.Second * 5,
		Timeout:         time.Millisecond * 200,
	},
	Kafka: KafkaConfig{Addr: "localhost:9094"},
	MySQL: MySQLConfig{DSN: "root:root@tcp(localhost:30306)/offlinepush"},
}
