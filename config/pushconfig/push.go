package pushconfig

import "github.com/xuhaidong1/offlinepush/internal/domain"

// PushConfig 推送配置
type PushConfig struct {
	Business       domain.Business `json:"business"`
	DeviceTypeList []string        `json:"device_type_list"` // 描述推送哪些类型设备
	// 推送时间cron表达式
	Cron   string `json:"cron"`
	Weight uint32 `json:"weight"`
	Qps    int    `json:"qps"`
}

var PushMap = map[string]PushConfig{
	"reboot": {
		Business:       domain.Business{Name: "reboot"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X"},
		Cron:           "0 0 3 * *",
		Weight:         400,
		Qps:            10000,
	},
	"weather": {
		Business:       domain.Business{Name: "weather"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X", "iHome2", "iHome2S", "iHome2C", "iHome2X"},
		Cron:           "@every 30m",
		Weight:         1300,
		Qps:            4000,
	},
	"card": {
		Business:       domain.Business{Name: "card", Domain: "www.card.iHome.com"},
		DeviceTypeList: []string{"iHome1", "iHome1S", "iHome1C", "iHome1X", "iHome2", "iHome2S", "iHome2C", "iHome2X"},
		Cron:           "@every 1h",
		Weight:         1100,
		Qps:            4000,
	},
}
