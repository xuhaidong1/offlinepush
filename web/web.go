package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xuhaidong1/offlinepush/internal/service"
)

type PushHandler struct {
	service *service.PushService
}

func NewPushHandler(pushService *service.PushService) *PushHandler {
	return &PushHandler{service: pushService}
}

func (h *PushHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/offlinepush")
	g.GET("/config/:name", h.GetPushConfig)
	g.GET("/business/status", h.GetBusinessStatus)
	g.GET("/pods", h.PodsList)
	g.POST("/task/add", h.AddTask)
	g.POST("/business/status", h.SetBusinessStatus)
	g.POST("/shutdown", h.Shutdown)
}

func (h *PushHandler) GetPushConfig(ctx *gin.Context) {
	biz := ctx.Param("name")
	config, err := h.service.GetPushConfig(biz)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, config)
}

func (h *PushHandler) AddTask(ctx *gin.Context) {
	type Req struct {
		Business string `json:"business"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	err := h.service.AddTask(req.Business)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.String(http.StatusOK, "success")
}

func (h *PushHandler) SetBusinessStatus(ctx *gin.Context) {
	type Req struct {
		Business string `json:"business"`
		Stop     bool   `json:"stop"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	if req.Stop {
		err := h.service.Pause(req.Business)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.String(http.StatusOK, "success")
		return
	} else {
		err := h.service.Resume(ctx, req.Business)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.String(http.StatusOK, "success")
		return
	}
}

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

func (h *PushHandler) Shutdown(ctx *gin.Context) {
	h.service.Shutdown()
	ctx.JSON(http.StatusOK, "server closed")
}
