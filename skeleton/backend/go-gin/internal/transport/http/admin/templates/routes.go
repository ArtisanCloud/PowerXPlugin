package templates

import (
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	h := NewTemplateHandler(deps)

	g := rg.Group("/templates", httpmw.EnsureTenant())
	{
		g.GET("", h.GetTemplates)
		g.GET("/:id", h.GetTemplate)
	}

	manage := rg.Group("/templates", httpmw.EnsureTenant(), httpmw.RequireRoot())
	{
		manage.POST("", h.CreateTemplate)
		manage.PUT("/:id", h.UpdateTemplate)
		manage.DELETE("/:id", h.DeleteTemplate)
		manage.POST("/batch-clone", h.BatchCloneTemplates)
		manage.POST("/:id/validate", h.ValidateTemplateCapability)
	}

	adminGroup := rg.Group("/admin/templates", httpmw.EnsureTenant(), httpmw.RequireRoot())
	{
		adminGroup.POST("/batch-clone", h.BatchCloneTemplates)
		adminGroup.POST("/:id/validate", h.ValidateTemplateCapability)
	}
}
