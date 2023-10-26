package ioc

import (
	"github.com/gin-gonic/gin"
	"github.com/xuhaidong1/offlinepush/web"
)

func InitWebServer(hdl *web.PushHandler) *gin.Engine {
	server := gin.Default()
	hdl.RegisterRoutes(server)
	return server
}
