package marketplace

import (
	"github.com/gin-gonic/gin"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
	httpmw "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/middleware"
)

// RegisterRoutes wires tenant-facing marketplace license endpoints.
func RegisterRoutes(group *gin.RouterGroup, deps *app.Deps) {
	if group == nil || deps == nil {
		return
	}

	handler := NewLicenseHandler(deps)
	if handler == nil {
		return
	}

	licenses := group.Group("/licenses", httpmw.EnsureTenant())
	{
		licenses.POST("", handler.Create)
		licenses.GET("/:id", handler.Get)
		licenses.POST("/:id", handler.Renew)
		licenses.POST("/:id/offline-extend", handler.ExtendOffline)
	}
}
