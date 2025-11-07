package devapi

import (
	"log/slog"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/handlers"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/middleware"
)

// MTLSConfig holds mTLS configuration for Dev API
type MTLSConfig struct {
	// Client CA file path for verifying client certificates
	ClientCAFile string
	// Whether to require client certificates
	RequireClientCert bool
}

// RegisterRoutes registers all Dev API routes with mTLS middleware
func RegisterRoutes(router bootstrap.Router, config MTLSConfig) {
	// Create Dev API handler
	handler := handlers.NewDevPluginHandler(nil)

	// Group all Dev API routes with mTLS middleware
	devGroup := router.Group("/internal/dev/plugins")

	// Apply mTLS middleware to the entire Dev API group
	devGroup.Use(middleware.NewMTLSMiddleware(config))

	// Register endpoints
	devGroup.Handle("POST", "/register", handler.Register)
	devGroup.Handle("POST", "/reload", handler.Reload)
	devGroup.Handle("DELETE", "/register/:sessionId", handler.Delete)

	slog.Info("Dev API routes registered with mTLS middleware")
}
