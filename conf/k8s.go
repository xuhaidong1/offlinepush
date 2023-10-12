//go:build k8s

package conf

var StartConfig = Config{
	Redis: RedisConfig{Addr: "offlinepush-redis:6579", Password: ""},
	Etcd:  EtcdConfig{Addr: "offlinepush-etcd:2379"},
}

type Config struct {
	Redis RedisConfig
	Etcd  EtcdConfig
}

type EtcdConfig struct {
	Addr string
}

type RedisConfig struct {
	Addr     string
	Password string
}
