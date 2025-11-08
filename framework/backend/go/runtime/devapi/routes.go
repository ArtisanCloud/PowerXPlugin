package devapi

import (
	"log/slog"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/handlers"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/telemetry"
)

// RegisterRoutes registers all Dev API routes with mTLS middleware
func RegisterRoutes(router bootstrap.Router, config middleware.MTLSConfig) {
	// Core handlers
	handler := handlers.NewDevPluginHandler(nil)
	hostHandler := handlers.NewHostSimulatorHandler(nil)
	sandboxHandler := handlers.NewSandboxValidationHandler()
	debugHandler := handlers.NewDebugReportHandler(telemetry.NewRecorder(nil))

	// Group all Dev API routes with mTLS middleware
	devGroup := router.Group("/internal/dev/plugins")

	// Apply mTLS middleware to the entire Dev API group
	devGroup.Use(middleware.NewMTLSMiddleware(config))

	// Register endpoints
	devGroup.Handle("POST", "/register", handler.Register)
	devGroup.Handle("POST", "/reload", handler.Reload)
	devGroup.Handle("DELETE", "/register/:sessionId", handler.Delete)

	// Host simulator routes
	hostGroup := router.Group("/internal/dev/hosts")
	hostGroup.Use(middleware.NewMTLSMiddleware(config))
	hostGroup.Handle("POST", "/sessions", hostHandler.Start)
	hostGroup.Handle("GET", "/sessions/:sessionId", hostHandler.Status)
	hostGroup.Handle("DELETE", "/sessions/:sessionId", hostHandler.Stop)
	hostGroup.Handle("POST", "/sessions/:sessionId/attach", hostHandler.Attach)
	hostGroup.Handle("GET", "/sessions/:sessionId/logs", hostHandler.Logs)

	// Sandbox validation
	sandboxGroup := router.Group("/internal/dev/sandbox")
	sandboxGroup.Use(middleware.NewMTLSMiddleware(config))
	sandboxGroup.Handle("POST", "/deploy", sandboxHandler.Deploy)

	// Debug reports
	debugGroup := router.Group("/internal/dev/debug")
	debugGroup.Use(middleware.NewMTLSMiddleware(config))
	debugGroup.Handle("POST", "/report", debugHandler.Report)

	slog.Info("Dev API routes registered with mTLS middleware")
}
