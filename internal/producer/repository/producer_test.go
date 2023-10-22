package repository

import (
	"testing"
)

func Test_producerRepository_WriteBackLeftTask(t *testing.T) {
	//simpleLru, _ := simplelru.NewLRU[string, any](100000, func(key string, value any) {})
	//cache := elru.NewCache(simpleLru)
	//msg := domain.Message{
	//	Business: domain.Business{
	//		Name:   "22",
	//		Domain: "3",
	//	},
	//	Device: domain.Device{
	//		Type: "3",
	//		ID:   "4",
	//	},
	//}
	//msgJson, err := json.Marshal(msg)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//_, err = cache.LPush(context.Background(), msg.Business.Name, string(msgJson))
	//_, err = cache.LPush(context.Background(), msg.Business.Name, string(msgJson))
	//val := cache.LPop(context.Background(), msg.Business.Name)
	//bytes, err := val.AsBytes()
	//if err != nil {
	//	log.Println(1)
	//	log.Fatalln(err)
	//}
	//val = cache.LPop(context.Background(), msg.Business.Name)
	//bytes, err = val.AsBytes()
	//if err != nil {
	//	log.Println(2)
	//	log.Fatalln(err)
	//}
	//var msgg domain.Message
	//err = json.Unmarshal(bytes, &msgg)
	//if err != nil {
	//	log.Fatalln(msgg)
	//}
}
