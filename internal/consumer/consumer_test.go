package consumer

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/panjf2000/ants/v2"
	"github.com/serialx/hashring"
	"github.com/stretchr/testify/assert"
	"github.com/xuhaidong1/go-generic-tools/container/queue"
	"github.com/xuhaidong1/offlinepush/internal/domain"
)

type Cmer struct{}

type Puller struct{}

// var q = queue.NewConcurrentBlockingQueue[int](10000)
//
//	func (p *Puller) Pull(ctx context.Context, biz string) {
//		for {
//			select {
//			case <-ctx.Done():
//				return
//			default:
//				//data := redis.lrange
//				//if nomessage{
//				//	q.Enqueue(ctx,-1)
//				//	return
//				//}
//				data := func() (res []int) {
//					for i := 0; i < 40; i++ {
//						res = append(res, i)
//					}
//					res = append(res, -1)
//					return
//				}()
//				for i, d := range data {
//					err := q.Enqueue(ctx, d)
//					if err != nil {
//						WriteBackLeftMessage(data[i:])
//						return
//					}
//				}
//			}
//		}
//	}
//
//	func WriteBackLeftMessage(data []int) {
//		for {
//			if q.IsEmpty() {
//				break
//			}
//			re, err := q.Dequeue(context.Background())
//			if err != nil {
//				return
//			}
//			if re != -1 {
//				log.Println(re)
//			}
//		}
//		for _, d := range data {
//			if d != -1 {
//				log.Println(d)
//			}
//		}
//	}
//
//	func (c *Cmer) Cons(ctx, dequeueCtx context.Context, biz string, finished chan struct{}) error {
//		for {
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			case <-finished:
//				return nil
//			default:
//				msg, err := q.Dequeue(dequeueCtx)
//				if err != nil {
//					// 这里不知道是什么原因取消的，
//					// 如果是上级取消（服务关闭，暂停）会走入case <-ctx.Done():
//					// 如果是消费完了没有消息了，会走入case <-finished:
//					continue
//				}
//				if msg == -1 {
//					close(finished)
//					continue
//				}
//				// lg.Println(msg)
//			}
//		}
//	}
//
// var (
//
//	cmer   = &Cmer{}
//	puller = &Puller{}
//
// )
//
//	func TestConsumer_Push(t *testing.T) {
//		eg := errgroup.Group{}
//		ctx, cancel := context.WithCancel(context.Background())
//		defer cancel()
//		finish := make(chan struct{})
//		dequeueCtx, dCancel := context.WithCancel(ctx)
//		go func() {
//			<-finish
//			dCancel()
//		}()
//		//go func() {
//		//	time.Sleep(time.Millisecond * 5)
//		//	cancel()
//		//}()
//		go puller.Pull(ctx, "biz")
//		for i := 0; i < 10; i++ {
//			eg.Go(func() error {
//				return cmer.Cons(ctx, dequeueCtx, "biz", finish)
//			})
//		}
//		err := eg.Wait()
//		if err != nil {
//			return
//			// 由pull写回leftmsg
//		}
//	}
//
// //var lg = NewPushLogger()
// //
// //func NewPushLogger() *log.Logger {
// //	currentDir, err := os.Getwd()
// //	if err != nil {
// //		panic(err)
// //	}
// //	logFilePath := filepath.Join(currentDir, "push.log")
// //	// os.O_APPEND
// //	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0o666)
// //	if err != nil {
// //		panic(err)
// //	}
// //	return log.New(logFile, "pushlogger:", log.LstdFlags)
// //}
//
//	func dd(cmd redis.Cmdable, biz string, devices []domain.Device) {
//		for _, d := range devices {
//			taskKey := biz + ":" + d.Type
//			cmd.LPush(context.Background(), taskKey, d.ID)
//			cmd.SAdd(context.Background(), biz, taskKey)
//		}
//	}
func TestOneHash(t *testing.T) {
	// nodes := []string{"192.168.0.246:11212",
	//	"192.168.0.247:11212",
	//	"192.168.0.249:11212"}
	// ring := hashring.New(nodes)
	weights := make(map[string]int)
	weights["192.168.0.246:11212"] = 1
	weights["192.168.0.247:11212"] = 2
	weights["192.168.0.249:11212"] = 1

	ring := hashring.NewWithWeights(weights)
	keys := []string{
		"iHome1", "iHome1S", "iHome1X", "iHome2", "ihome2S",
		"xiaodu1", "xiaodu2", "xiaodu2S", "tiantian", "iHome3", "iHome3S", "iHome3X", "iHome4", "ihome4S",
		"xiaodu5", "xiaodu6", "xiaodu4S", "tiantian2",
	}
	mp := make(map[string][]string)
	for _, k := range keys {
		node, _ := ring.GetNode(k)
		if mp[node] == nil {
			slce := make([]string, 0)
			mp[node] = slce
		}
		mp[node] = append(mp[node], k)
	}
	log.Println(mp)

	ring = ring.AddWeightedNode("192.168.0.245:11212", 1)
	// ring = ring.AddWeightedNode("192.168.0.244:11212",1)
	mp = make(map[string][]string)
	for _, k := range keys {
		node, _ := ring.GetNode(k)
		if mp[node] == nil {
			slce := make([]string, 0)
			mp[node] = slce
		}
		mp[node] = append(mp[node], k)
	}
	log.Println(mp)
}

func BenchmarkBlockingQ(b *testing.B) {
	gNum := 10000
	msgNum := 102400
	q := queue.NewConcurrentBlockingQueue[domain.Message](gNum)
	// q := queue.NewConcurrentBlockingQueue[domain.Message](gNum)
	// q := queue.NewConcurrentBlockingQueue[domain.Message](gNum)
	prepare := func(wg *sync.WaitGroup, q *queue.ConcurrentBlockingQueue[domain.Message]) {
		ctx := context.Background()
		for i := 0; i < msgNum; i++ {
			err := q.Enqueue(ctx, domain.Message{
				Business: domain.Business{Name: "bench"},
				Device: domain.Device{
					Type: strconv.Itoa(i),
					ID:   strconv.Itoa(i),
				},
			})
			if err != nil {
				panic(err)
			}
		}
		for k := 0; k < gNum; k++ {
			err := q.Enqueue(context.Background(), domain.GetEOF("bench"))
			if err != nil {
				panic(err)
			}
		}
		wg.Done()
	}
	f := func() {
		worker := &Worker{}
		wg := &sync.WaitGroup{}
		wg.Add(10001)
		go prepare(wg, q)
		for k := 0; k < gNum; k++ {
			go worker.work(wg, q)
		}
		wg.Wait()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f()
	}
}

func TestBlockingQ(t *testing.T) {
	q := queue.NewConcurrentBlockingQueue[domain.Message](10000)
	prepare := func(wg *sync.WaitGroup) {
		ctx := context.Background()
		for i := 0; i < 102400; i++ {
			err := q.Enqueue(ctx, domain.Message{
				Business: domain.Business{Name: "bench"},
				Device: domain.Device{
					Type: strconv.Itoa(i),
					ID:   strconv.Itoa(i),
				},
			})
			if err != nil {
				panic(err)
			}
		}
		err := q.Enqueue(context.Background(), domain.GetEOF("bench"))
		if err != nil {
			panic(err)
		}
		wg.Done()
	}

	f := func() {
		worker := &Worker{}
		wg := &sync.WaitGroup{}
		wg.Add(10001)
		go prepare(wg)
		for k := 0; k < 10000; k++ {
			go worker.work(wg, q)
		}
		wg.Wait()
	}

	f()
}

type Worker struct{}

func (w *Worker) work(wg *sync.WaitGroup, q *queue.ConcurrentBlockingQueue[domain.Message]) {
	for {
		msg, err := q.Dequeue(context.Background())
		if err != nil {
			continue
		}
		if domain.IsEOF(msg) {
			wg.Done()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func BenchmarkGoroutinePool(b *testing.B) {
	//f := func() {
	//	wg := &sync.WaitGroup{}
	//	p, _ := ants.NewPoolWithFunc(10000, func(i any) {
	//		time.Sleep(20 * time.Millisecond)
	//		wg.Done()
	//	})
	//	defer p.Release()
	//	// Submit tasks one by one.
	//	for i := 0; i < 102400; i++ {
	//		wg.Add(1)
	//		_ = p.Invoke(int32(i))
	//	}
	//	wg.Wait()
	//}
	f := func() {
		wg := &sync.WaitGroup{}
		runTimes := 102400
		// Use the MultiPoolFunc and set the capacity of 10 goroutine pools to (runTimes/10).
		mpf, _ := ants.NewMultiPoolWithFunc(10, 1000, func(i interface{}) {
			time.Sleep(20 * time.Millisecond)
			wg.Done()
		}, ants.LeastTasks)
		defer func(mpf *ants.MultiPoolWithFunc, timeout time.Duration) {
			err := mpf.ReleaseTimeout(timeout)
			if err != nil {
				panic(err)
			}
		}(mpf, 5*time.Second)
		for i := 0; i < runTimes; i++ {
			wg.Add(1)
			_ = mpf.Invoke(int32(i))
		}
		wg.Wait()
		fmt.Printf("running goroutines: %d\n", mpf.Running())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f()
	}
}

//------------------------------------------------------------------------------------

func BenchmarkBlockingQManyTopics(b *testing.B) {
	gNum := 10000
	msgNum := 102400
	EOF := domain.GetEOF("bench")
	msgs := make([]domain.Message, 102411)
	q := queue.NewConcurrentBlockingQueue[domain.Message](gNum)
	q1 := queue.NewConcurrentBlockingQueue[domain.Message](gNum)
	q2 := queue.NewConcurrentBlockingQueue[domain.Message](gNum)
	prepare := func(wg *sync.WaitGroup, q *queue.ConcurrentBlockingQueue[domain.Message]) {
		ctx := context.Background()
		for i := 0; i < msgNum; i++ {
			err := q.Enqueue(ctx, msgs[i])
			if err != nil {
				panic(err)
			}
		}
		for k := 0; k < gNum; k++ {
			err := q.Enqueue(context.Background(), EOF)
			if err != nil {
				panic(err)
			}
		}
		wg.Done()
	}
	f := func() {
		worker := &Worker{}
		wg := &sync.WaitGroup{}
		wg.Add(gNum*3 + 3)
		go func() {
			go prepare(wg, q)
			for k := 0; k < gNum; k++ {
				go worker.work(wg, q)
			}
		}()
		go func() {
			go prepare(wg, q2)
			for k := 0; k < gNum; k++ {
				go worker.work(wg, q2)
			}
		}()
		go func() {
			go prepare(wg, q1)
			for k := 0; k < gNum; k++ {
				go worker.work(wg, q1)
			}
		}()
		wg.Wait()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f()
	}
}

func BenchmarkPoolManyTopics(b *testing.B) {
	msgs := make([]domain.Message, 102411)
	f := func() {
		wg := &sync.WaitGroup{}
		runTimes := 102400
		// Use the MultiPoolFunc and set the capacity of 10 goroutine pools to (runTimes/10).
		mpf, _ := ants.NewMultiPoolWithFunc(10, -1, func(a interface{}) {
			time.Sleep(20 * time.Millisecond)
			if _, ok := a.(domain.Message); !ok {
				panic("!ok")
			}
			wg.Done()
		}, ants.LeastTasks)
		defer func(mpf *ants.MultiPoolWithFunc, timeout time.Duration) {
			err := mpf.ReleaseTimeout(timeout)
			if err != nil {
				panic(err)
			}
		}(mpf, 5*time.Second)
		wg.Add(runTimes * 3)
		for k := 0; k < 3; k++ {
			go func() {
				for i := 0; i < runTimes; i++ {
					_ = mpf.Invoke(msgs[i])
				}
			}()
		}
		wg.Wait()
		fmt.Printf("running goroutines: %d\n", mpf.Running())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f()
	}
}

func TestProducer(t *testing.T) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	// 发送一次，不管服务端
	// cfg.Producer.RequiredAcks = sarama.NoResponse
	// 发送，并且需要写入主分区
	cfg.Producer.RequiredAcks = sarama.WaitForLocal
	// 发送，并且需要同步到所有的 ISR 上
	// cfg.Producer.RequiredAcks = sarama.WaitForAll

	cfg.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	// cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	// cfg.Producer.Partitioner = sarama.NewHashPartitioner
	// cfg.Producer.Partitioner = sarama.NewManualPartitioner
	// cfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
	// cfg.Producer.Partitioner = sarama.NewCustomPartitioner()
	// 这个是为了兼容 JAVA，不要用
	// cfg.Producer.Partitioner = sarama.NewReferenceHashPartitioner
	producer, err := sarama.NewSyncProducer([]string{"localhost:9094"}, cfg)
	assert.NoError(t, err)
	p, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic: "test_topic",
		Value: sarama.StringEncoder("hello,这是一条消x息"),
		// 会在 producer 和 consumer 之间传递
		//Headers: []sarama.RecordHeader{
		//	{Key: []byte("header1"), Value: []byte("header1_value")},
		//},
		//Metadata: map[string]any{"metadata1": "metadata_value1"},
	})
	assert.NoError(t, err)
	t.Log(p, offset)
}

func TestConsumer(t *testing.T) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.AutoCommit.Enable = false
	cfg.Consumer.Return.Errors = true

	cg, err := sarama.NewConsumerGroup([]string{"localhost:9094"},
		"test_group", cfg)
	assert.NoError(t, err)
	// 这里是测试，我们就控制消费三十秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	// 开始消费，会在这里阻塞住
	// err = cg.Work(ctx,
	//	[]string{"test_topic"},
	//	saramax.NewHandler[string](logx.NewZapLogger(ioc.Logger),
	//		func(msg *sarama.ConsumerMessage, x string) error {
	//			t.Log(msg.Topic, msg.Partition, msg.Offset, msg.Value, x)
	//			return nil
	//		}))
	err = cg.Consume(ctx,
		[]string{"test_topic"}, &ConsumerHandler{})
	assert.NoError(t, err)
}

type ConsumerHandler struct{}

func (c *ConsumerHandler) Setup(session sarama.ConsumerGroupSession) error {
	// 执行一些初始化的事情
	log.Println("Handler Setup")
	// 假设要重置到 0
	// var offset int64 = 0
	partitions := session.Claims()["test_topic"]
	for _, p := range partitions {
		// 这里之所以调用这个方法是因为当消费者没有消费过这个新分区的任何一条消息时
		// kafka内部__consumer_offsets这个主题下分区偏移量为-1
		// 当分区偏移量为-1时sess.ResetOffset方法是无效的，所以先将偏移量提交至
		// kafka集群并且0，这里不必担心偏移量出现问题，因为sess.MarkOffset方法
		// 不会提交真实分区偏移量小于第三个参数的情况
		session.MarkOffset("test_topic", p, 0, "")
		session.ResetOffset("test_topic", p, 0, "")
	}

	return nil
}

func (c *ConsumerHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	// 执行一些清理工作
	log.Println("Handler Cleanup")
	return nil
}

func (c *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ch := claim.Messages()
	for msg := range ch {
		log.Println(msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
		// 标记为消费成功
		session.MarkMessage(msg, "")
	}
	return nil
}
