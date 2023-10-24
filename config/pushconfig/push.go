package pushconfig

import "github.com/xuhaidong1/offlinepush/internal/domain"

// PushConfig 推送配置
type PushConfig struct {
	Business       domain.Business
	DeviceTypeList []string // 描述推送哪些类型设备
	// 推送时间cron表达式
	Cron   string
	Weight uint32
	Qps    int
}

var PushMap = map[string]PushConfig{
	"reboot": {
		Business:       domain.Business{Name: "reboot"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X"},
		Cron:           "50 45 17 * * *",
		Weight:         400,
		Qps:            4000,
	},
	"weather": {
		Business:       domain.Business{Name: "weather"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X", "iHome2", "iHome2S", "iHome2C", "iHome2X"},
		Cron:           "@every 1h",
		Weight:         1300,
		Qps:            4000,
	},
	"card": {
		Business:       domain.Business{Name: "card", Domain: "www.card.iHome.com"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X", "iHome2", "iHome2S", "iHome2C", "iHome2X"},
		Cron:           "@every 3h",
		Weight:         1100,
		Qps:            4000,
	},
}
