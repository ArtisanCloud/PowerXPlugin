package runtime_ops

import (
	"github.com/gin-gonic/gin"
	service "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/admin/runtime_ops"
)

// MetricsHandler exposes runtime ops metrics endpoint.
func MetricsHandler(c *gin.Context) {
	handler := service.MetricsHTTPHandler()
	handler.ServeHTTP(c.Writer, c.Request)
}
