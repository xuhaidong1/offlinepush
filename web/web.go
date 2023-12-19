package web

import (
	"errors"
	"net/http"

	"github.com/xuhaidong1/offlinepush/config/pushconfig"

	"github.com/gin-gonic/gin"
	"github.com/xuhaidong1/offlinepush/internal/service"
)

var ErrNoBiz = errors.New("没有这个业务")

type PushHandler struct {
	service *service.PushService
}

func NewPushHandler(pushService *service.PushService) *PushHandler {
	return &PushHandler{service: pushService}
}

func (h *PushHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/offlinepush")
	g.GET("/config/:name", h.GetPushConfig)
	// g.GET("/business/status", h.GetBusinessStatus)
	g.GET("/pods", h.PodsList)
	g.GET("/count", h.GetCount)
	g.POST("/task/add", h.AddTask)
	g.POST("/count", h.ResetCount)
	// g.POST("/business/status", h.SetBusinessStatus)
	// g.POST("/shutdown", h.Shutdown)
}

func (h *PushHandler) GetPushConfig(ctx *gin.Context) {
	biz := ctx.Param("name")
	cfg, ok := pushconfig.PushMap[biz]
	if !ok {
		ctx.String(http.StatusBadRequest, ErrNoBiz.Error())
		return
	}
	ctx.JSON(http.StatusOK, cfg)
}

func (h *PushHandler) AddTask(ctx *gin.Context) {
	type Req struct {
		Business string `json:"business"`
		Num      int    `json:"num"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	_, ok := pushconfig.PushMap[req.Business]
	if !ok {
		ctx.String(http.StatusBadRequest, ErrNoBiz.Error())
		return
	}
	err := h.service.AddTask(ctx, req.Business, req.Num)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.String(http.StatusOK, "success")
}

func (h *PushHandler) ResetCount(ctx *gin.Context) {
	err := h.service.ResetCounter(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.String(http.StatusOK, "success")
}

func (h *PushHandler) GetCount(ctx *gin.Context) {
	mp, err := h.service.GetCount(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, mp)
}

//func (h *PushHandler) SetBusinessStatus(ctx *gin.Context) {
//	type Req struct {
//		Business string `json:"business"`
//		Stop     bool   `json:"stop"`
//	}
//	var req Req
//	if err := ctx.Bind(&req); err != nil {
//		ctx.String(http.StatusInternalServerError, err.Error())
//		return
//	}
//	if req.Stop {
//		err := h.service.Pause(ctx, req.Business)
//		if err != nil {
//			ctx.String(http.StatusBadRequest, err.Error())
//			return
//		}
//		ctx.String(http.StatusOK, "success")
//		return
//	} else {
//		err := h.service.Resume(ctx, req.Business)
//		if err != nil {
//			ctx.String(http.StatusBadRequest, err.Error())
//			return
//		}
//		ctx.String(http.StatusOK, "success")
//		return
//	}
//}

func (h *PushHandler) GetBusinessStatus(ctx *gin.Context) {
	res := h.service.GetBizStatus()
	ctx.JSON(http.StatusOK, res)
}

func (h *PushHandler) PodsList(ctx *gin.Context) {
	list, err := h.service.PodList(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, list)
}

//func (h *PushHandler) Shutdown(ctx *gin.Context) {
//	h.service.Shutdown()
//	ctx.JSON(http.StatusOK, "server closed")
//}
