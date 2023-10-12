package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xuhaidong1/offlinepush/cmd/ioc"
)

// 运行要加上 --config=conf/dev.toml
// 并且可以开启环境变量 EGO_DEBUG=true
func main() {
	etcd := ioc.InitEtcd()
	redisCmd := ioc.InitRedis()
	log.Println("OfflinePush启动")
	log.Println("14")
	redisCmd.Set(context.Background(), "key1", "val1", time.Minute)
	result, err := redisCmd.Get(context.Background(), "key1").Result()
	if err != nil {
		panic(err)
	}
	if result != "val1" {
		panic("值不对")
	}

	// 创建一个键值对的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 执行 SET 操作
	_, err = etcd.Put(ctx, "mykey", "myvalue")
	if err != nil {
		log.Fatal(err)
	}

	// 执行 GET 操作
	resp, err := etcd.Get(ctx, "mykey")
	if err != nil {
		log.Fatal(err)
	}

	// 打印获取到的值
	for _, kv := range resp.Kvs {
		fmt.Printf("Key: %s, Value: %s\n", kv.Key, kv.Value)
	}
	for {
		time.Sleep(time.Second * 5)
		log.Println("alive")
	}
}
