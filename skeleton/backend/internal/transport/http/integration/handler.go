package integration

import (
	"net/http"

	"github.com/gin-gonic/gin"
	integrationService "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/integration"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
	"github.com/sirupsen/logrus"
)

// Handler 提供 integration HTTP API 的入口。
type Handler struct {
	deps     *app.Deps
	dispatch *integrationService.DispatchService
	logger   *logrus.Entry
}

// NewHandler 构造新的 Handler。
func NewHandler(deps *app.Deps) *Handler {
	var logger *logrus.Entry
	if deps != nil {
		logger = deps.RuntimeLogger(deps.Ctx, "integration_http", nil)
	}
	h := &Handler{
		deps:   deps,
		logger: logger,
	}
	h.dispatch = h.buildDispatchService()
	return h
}

func (h *Handler) buildDispatchService() *integrationService.DispatchService {
	if h.deps == nil {
		return nil
	}

	logger := h.logger
	if logger == nil {
		logger = logrus.WithField("component", "integration_http")
	}
	service := integrationService.BuildDispatchService(h.deps, logger)
	if service == nil {
		return nil
	}
	return service
}

// ListGrantMatrix 返回当前 GrantMatrix 视图。
func (h *Handler) ListGrantMatrix(c *gin.Context) {
	respondPlaceholder(c, http.StatusOK, "grant matrix listing not implemented")
}

// SubmitGrantMatrix 接收数据库覆盖项。
func (h *Handler) SubmitGrantMatrix(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "grant matrix override submission pending approval workflow")
}

// CreateSubscription 注册 webhook 订阅。
func (h *Handler) CreateSubscription(c *gin.Context) {
	respondPlaceholder(c, http.StatusCreated, "webhook subscription endpoint not implemented")
}

// ListSubscriptions 查询 webhook 订阅。
func (h *Handler) ListSubscriptions(c *gin.Context) {
	respondPlaceholder(c, http.StatusOK, "webhook subscription list not implemented")
}

// ReplayDLQ 触发 DLQ 补发。
func (h *Handler) ReplayDLQ(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "webhook DLQ replay not implemented")
}

// CreateSecret 注册外部凭证。
func (h *Handler) CreateSecret(c *gin.Context) {
	respondPlaceholder(c, http.StatusCreated, "secret lifecycle endpoint not implemented")
}

// RotateSecret 触发凭证轮换。
func (h *Handler) RotateSecret(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "secret rotation workflow not implemented")
}

func respondPlaceholder(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"status":  "pending",
		"message": message,
	})
}
