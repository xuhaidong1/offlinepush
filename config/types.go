package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Redis    RedisConfig
	Etcd     EtcdConfig
	Register RegisterConfig
	Lock     LockConfig
}

type EtcdConfig struct {
	Addr string
}

type RedisConfig struct {
	Addr     string
	Password string
}

type RegisterConfig struct {
	ServiceName    string
	ConsumerPrefix string
	PodName        string
}

type LockConfig struct {
	// 要抢的锁的名字
	LockKey string
	// 持有锁的过期时间
	Expiration time.Duration
	// 续约的时间间隔
	RefreshInterval time.Duration
	// redis lua脚本执行超时时间
	Timeout time.Duration
}

func GetPodName() string {
	podName, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprintf("Error getting Pod name: %v", err))
	}
	return podName
}
