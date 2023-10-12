//go:build !k8s

package conf

var StartConfig = Config{
	Redis: RedisConfig{Addr: "localhost:30379", Password: ""},
	Etcd:  EtcdConfig{Addr: "localhost:32379"},
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
