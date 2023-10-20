package pushconfig

import "github.com/xuhaidong1/offlinepush/internal/domain"

// PushConfig 推送配置
type PushConfig struct {
	Business       domain.Business
	DeviceTypeList []string // 描述推送哪些类型设备
	// 推送时间cron表达式
	Cron string
}

var PushMap = map[string]PushConfig{
	"reboot": {
		Business:       domain.Business{Name: "reboot"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X"},
		Cron:           "0 0 3 * * *",
	},
	"weather": {
		Business:       domain.Business{Name: "weather"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X", "iHome2", "iHome2S", "iHome2C", "iHome2X"},
		Cron:           "@every 1h",
	},
	"card": {
		Business:       domain.Business{Name: "card", Domain: "www.card.iHome.com"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X", "iHome2", "iHome2S", "iHome2C", "iHome2X"},
		Cron:           "@every 3h",
	},
}
