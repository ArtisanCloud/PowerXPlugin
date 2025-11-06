package integration

import (
	"github.com/gin-gonic/gin"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
)

// RegisterAPIRoutes 挂载 Integration 运行时 HTTP API。
func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil {
		return
	}

	handler := NewHandler(deps)
	group := rg.Group("/integration")
	{
		group.POST("/dispatch", handler.Dispatch)

		group.GET("/grant-matrix", handler.ListGrantMatrix)
		group.POST("/grant-matrix", handler.SubmitGrantMatrix)

		group.POST("/webhooks/subscriptions", handler.CreateSubscription)
		group.GET("/webhooks/subscriptions", handler.ListSubscriptions)
		group.POST("/webhooks/dlq/:attemptId/replay", handler.ReplayDLQ)

		group.POST("/secrets", handler.CreateSecret)
		group.POST("/secrets/:secretId/rotate", handler.RotateSecret)
	}
}
