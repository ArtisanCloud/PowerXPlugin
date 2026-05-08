package runtime_ops

import (
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires runtime ops endpoints behind the admin router.
func RegisterRoutes(router *gin.RouterGroup, deps *app.Deps) {
	runtimeService := runtimeops.NewService()
	bootstrap := NewBootstrapHandler(runtimeService)
	router.POST("/bootstrap", bootstrap.Bootstrap)

	sessions := NewSessionsHandler(deps)
	router.POST("/sessions/register", sessions.Register)
	router.POST("/sessions/:sessionID/ack", sessions.Ack)
	router.POST("/sessions/:sessionID/heartbeat", sessions.Heartbeat)
	router.POST("/sessions/:sessionID/close", sessions.Close)
	router.POST("/sessions/:sessionID/invoke", sessions.Invoke)

	quotaHandler := NewQuotaHandler(deps, deps.Config.RuntimeOps)
	quota := router.Group("/quota")
	quota.GET("/status", quotaHandler.GetStatus)
	quota.POST("/overrides", quotaHandler.SetOverride)

	router.GET("/metrics", MetricsHandler)

	router.POST("/event-bridge/emit", EventBridgeEmitHandler(deps))
	router.POST("/event-fabric/topics", EventFabricCreateTopicHandler(deps))
	router.POST("/ws-bus/publish", WSBusPublishHandler(deps))
	router.POST("/ws-bus/grant", WSBusGrantHandler(deps))
	router.POST("/ws-bus/test-flow", WSBusTestFlowHandler(deps))

	schedulerMode := NewSchedulerModeHandler(deps, runtimeService)
	schedulerRetry := NewSchedulerRetryHandler(deps, runtimeService)
	scheduler := router.Group("/scheduler")
	scheduler.POST("/mode/validate", schedulerMode.Validate)
	scheduler.POST("/dispatches/:dispatchId/retry", schedulerRetry.Retry)
	scheduler.POST("/dispatches/:dispatchId/pause", schedulerRetry.Pause)
	scheduler.POST("/tickets/:ticketId/resume", schedulerRetry.Resume)

	loggingPolicyHandler := NewLoggingPolicyHandler()
	logging := router.Group("/logging")
	logging.GET("/policy", loggingPolicyHandler.Get)
	logging.PUT("/policy", loggingPolicyHandler.Put)
	logging.POST("/probe", LoggingProbeHandler())
}
