package runtime_ops

import (
	runtimeops "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires runtime ops endpoints behind the admin router.
func RegisterRoutes(router *gin.RouterGroup, deps *app.Deps) {
	bootstrap := NewBootstrapHandler(runtimeops.NewService())
	router.POST("/bootstrap", bootstrap.Bootstrap)

	sessions := NewSessionsHandler(deps)
	router.POST("/sessions/register", sessions.Register)

	quotaHandler := NewQuotaHandler(deps, deps.Config.RuntimeOps)
	quota := router.Group("/quota")
	quota.GET("/status", quotaHandler.GetStatus)
	quota.POST("/overrides", quotaHandler.SetOverride)

	router.GET("/metrics", MetricsHandler)
}
