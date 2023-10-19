package pushconfig

import "github.com/xuhaidong1/offlinepush/internal/domain"

// PushConfig 推送配置
type PushConfig struct {
	Business       domain.Business
	DeviceTypeList []string // 描述推送哪些类型设备
	// 推送时间cron表达式
	Cron string
}
