package capability

import (
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires capability registration routes under /admin.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	handler := NewRegisterHandler(deps)
	if handler == nil {
		return
	}
	group := rg.Group("/capabilities/register")
	{
		group.GET("/template", handler.GetTemplate)
		group.POST("/validate", handler.ValidateDraft)
		group.POST("", handler.Submit)
	}
}
