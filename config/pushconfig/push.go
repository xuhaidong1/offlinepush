package pushconfig

import "github.com/xuhaidong1/offlinepush/internal/domain"

// PushConfig 推送配置
type PushConfig struct {
	Topic          domain.Topic `json:"topic"`
	DeviceTypeList []string     `json:"device_type_list"` // 描述推送哪些类型设备
	// 推送时间cron表达式
	Cron string `json:"cron"`
	Qps  int    `json:"qps"`
	Num  int    `json:"num"`
}

var PushMap = map[string]PushConfig{
	"reboot": {
		Topic:          domain.Topic{Name: "reboot"},
		DeviceTypeList: []string{"DeviceQ", "DeviceR", "DeviceS", "DeviceT", "DeviceX"},
		Cron:           "0 15 * * *",
		Qps:            4000,
	},
	"weather": {
		Topic: domain.Topic{Name: "weather"},
		DeviceTypeList: []string{
			"DeviceA", "DeviceB", "DeviceC", "DeviceD", "DeviceE", "DeviceF", "DeviceG", "DeviceH", "DeviceI", "DeviceJ",
			"DeviceK", "DeviceL", "DeviceM", "DeviceN", "DeviceP", "DeviceQ", "DeviceR", "DeviceS", "DeviceT", "DeviceX",
		},
		Cron: "@hourly",
		Qps:  4000,
	},
	"card": {
		Topic:          domain.Topic{Name: "card", Domain: "www.card.iHome.com"},
		DeviceTypeList: []string{"DeviceA", "DeviceB", "DeviceC", "DeviceD", "DeviceE", "DeviceF", "DeviceG", "DeviceH"},
		Cron:           "30 * * * *",
		Qps:            4000,
	},
}
