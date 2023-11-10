package ioc

import (
	"github.com/xuhaidong1/go-generic-tools/pluginsx/logx"
	"github.com/xuhaidong1/offlinepush/pkg/zapx"
	"go.uber.org/zap"
)

var Logger, _ = logx.NewLogger(zap.InfoLevel).Build()
var Loggerx = logx.NewZapLogger(Logger)
var PushLogger = zapx.NewPushLogger()
