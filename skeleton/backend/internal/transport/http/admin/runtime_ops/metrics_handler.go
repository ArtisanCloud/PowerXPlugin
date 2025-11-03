package runtime_ops

import (
	service "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/gin-gonic/gin"
)

// MetricsHandler exposes runtime ops metrics endpoint.
func MetricsHandler(c *gin.Context) {
	handler := service.MetricsHTTPHandler()
	handler.ServeHTTP(c.Writer, c.Request)
}
