package ioc

import (
	"github.com/xuhaidong1/offlinepush/internal/id"
	dao2 "github.com/xuhaidong1/offlinepush/internal/repository/dao"
	"testing"
	"time"
)

// 初始化200w条数据使用
func Test_init_Insert(t *testing.T) {
	gter, _ := id.NewGenerator(1)
	dao := dao2.NewGormDeviceDAO(InitDB())
	typ := []string{"DeviceA", "DeviceB", "DeviceC", "DeviceD", "DeviceE", "DeviceF", "DeviceG", "DeviceH", "DeviceI", "DeviceJ",
		"DeviceK", "DeviceL", "DeviceM", "DeviceN", "DeviceP", "DeviceQ", "DeviceR", "DeviceS", "DeviceT", "DeviceX"}

	for _, tp := range typ {
		for i := 0; i < 10; i++ {
			ds := make([]dao2.Device, 0, 10000)
			now := time.Now().UnixMilli()
			for j := 0; j < 10000; j++ {
				ds = append(ds, dao2.Device{
					Type:  tp,
					CID:   gter.Generate(),
					Ctime: now,
					Utime: now,
				})
			}
			err := dao.BatchInsert(ds)
			if err != nil {
				panic(err)
			}
		}
	}
}

func TestInitDB(t *testing.T) {
	InitDB()
}
