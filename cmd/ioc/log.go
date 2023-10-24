package ioc

import (
	"github.com/xuhaidong1/offlinepush/pkg/zapx"
	"go.uber.org/zap"
)

var Logger, _ = zapx.NewLogger(zap.InfoLevel).Build()
var PushLogger = zapx.NewPushLogger()
