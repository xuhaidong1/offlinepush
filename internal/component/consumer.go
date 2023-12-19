package component

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/panjf2000/ants/v2"
	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/go-generic-tools/pluginsx/saramax"
	"github.com/xuhaidong1/offlinepush/cmd/ioc"
	"github.com/xuhaidong1/offlinepush/config/pushconfig"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type Consumer struct {
	// cron 写 producer读
	// notifyProducer <-chan pushconfig.PushConfig
	workPool *ants.MultiPoolWithFunc
	client   sarama.Client
	worker   Worker
	// interceptor *interceptor.Interceptor
	// CancelFuncs *sync.Map
	logger logx.Logger
}

func NewConsumer(client sarama.Client, worker Worker) *Consumer {
	c := &Consumer{
		client: client,
		worker: worker,
		logger: ioc.Loggerx.With(logx.Field{Key: "component", Value: "Consumer"}),
	}
	// Use the MultiPoolFunc and set the capacity of 20 goroutine pools
	mpf, _ := ants.NewMultiPoolWithFunc(30, -1, func(a interface{}) {
		args := a.(Args)
		c.worker.Work(args)
	}, ants.RoundRobin)
	c.workPool = mpf

	return c
}

func (c *Consumer) Run(ctx context.Context) {
	topics := make([]string, 0, len(pushconfig.PushMap))
	for topic := range pushconfig.PushMap {
		topics = append(topics, topic)
	}
	err := c.StartBatch(ctx, topics)
	if err != nil {
		panic(err)
	}
}

func (c *Consumer) StartBatch(ctx context.Context, topics []string) error {
	cg, err := sarama.NewConsumerGroupFromClient("offlinepush:group", c.client)
	if err != nil {
		return err
	}
	const batchSize = 100
	go func() {
		er := cg.Consume(ctx, topics,
			saramax.NewBatchHandler[domain.Message](c.logger, batchSize, c.BatchConsume,
				saramax.WithSetup[domain.Message](func(session sarama.ConsumerGroupSession) error {
					//partitions := session.Claims()[topic]
					//for _, p := range partitions {
					//	session.MarkOffset(topic, p, 91990, "")
					//	session.ResetOffset(topic, p, 91990, "")
					//}
					return nil
				})))
		if er != nil {
			c.logger.Error("退出了消费循环异常", logx.Error(er))
		}
	}()
	return nil
}

func (c *Consumer) BatchConsume(saramaMsgs []*sarama.ConsumerMessage, msgs []domain.Message) error {
	wg := &sync.WaitGroup{}
	wg.Add(len(msgs))
	for _, msg := range msgs {
		err := c.workPool.Invoke(Args{
			Msg: msg,
			Wg:  wg,
		})
		if err != nil {
			c.logger.Error("workPool Invoke error", logx.Error(err))
		}
	}
	wg.Wait()
	mp := make(map[string]struct{})
	for _, m := range saramaMsgs {
		mp[m.Topic+":"+strconv.Itoa(int(m.Partition))] = struct{}{}
	}
	if len(saramaMsgs) > 0 {
		c.logger.Info("消费完成", logx.Int64("数量", int64(len(saramaMsgs))),
			logx.String("消息信息", func() string {
				res := make([]string, 0, len(mp))
				for k := range mp {
					res = append(res, k)
				}
				return strings.Join(res, ",")
			}()))
	}
	return nil
}
