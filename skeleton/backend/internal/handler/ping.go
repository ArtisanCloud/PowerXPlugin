package handler

import (
	"net/http"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/service"
)

// PingHandler 演示如何组合 service 与框架 Handler。
type PingHandler struct {
	svc *service.PingService
}

// NewPingHandler 创建 PingHandler。
func NewPingHandler(svc *service.PingService) *PingHandler {
	return &PingHandler{svc: svc}
}

// Handle 返回框架识别的 Handler。
func (h *PingHandler) Handle() bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		ctx.JSON(http.StatusOK, h.svc.Ping())
	}
}
