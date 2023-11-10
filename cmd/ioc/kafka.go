package ioc

import (
	"github.com/IBM/sarama"
	"github.com/xuhaidong1/offlinepush/config"
)

func InitKafka() sarama.Client {
	addrs := []string{config.StartConfig.Kafka.Addr}

	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.RequiredAcks = sarama.WaitForLocal
	saramaCfg.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Consumer.Return.Errors = true

	client, err := sarama.NewClient(addrs, saramaCfg)
	if err != nil {
		panic(err)
	}
	return client
}

func NewSyncProducer(client sarama.Client) sarama.SyncProducer {
	res, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		panic(err)
	}
	return res
}

// NewConsumers 面临的问题依旧是所有的 Consumer 在这里注册一下
//func NewConsumers(c1 *article.InteractiveReadEventConsumer) []events.Consumer {
//	return []events.Consumer{c1}
//}
