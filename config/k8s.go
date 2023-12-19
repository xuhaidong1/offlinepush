//go:build k8s

package config

import "time"

var StartConfig = Config{
	Redis: RedisConfig{
		Addr:                "offlinepush-redis:6579",
		Password:            "",
		ConsumerLeftTaskKey: "k8s-offlinepush:consumer:leftmessage",
		ProducerLeftTaskKey: "k8s-offlinepush:producer:lefttask",
	},
	Etcd: EtcdConfig{Addr: "offlinepush-etcd:2379"},
	Register: RegisterConfig{
		ServiceName:    "k8s-offlinepush-service",
		ConsumerPrefix: "k8s-offlinepush-consumer",
		InterceptorKey: "k8s-offlinepush-interceptor",
		ManualTaskKey:  "k8s-offlinepush-manual",
		ReadCountKey:   "k8s-offlinepush-msg-count",
		WriteCountKey:  "k8s-offlinepush-msg-count-write",
		PodName:        GetPodName(),
	},
	Lock: LockConfig{
		LockKey:         "k8s-offlinepush-lock",
		Expiration:      time.Second * 6,
		RefreshInterval: time.Second * 5,
		Timeout:         time.Millisecond * 200,
	},

	Kafka: KafkaConfig{Addr: "offlinepush-kafka:9094"},
	MySQL: MySQLConfig{DSN: "root:root@tcp(offlinepush-mysql-k8s:3316)/offlinepush"},
}
