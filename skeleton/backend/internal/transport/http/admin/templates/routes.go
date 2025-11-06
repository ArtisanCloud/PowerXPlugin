package templates

import (
	"github.com/gin-gonic/gin"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
	httpmw "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/middleware"
)

func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	h := NewTemplateHandler(deps)

	g := rg.Group("/templates", httpmw.EnsureTenant())
	{
		g.GET("", h.GetTemplates)
		g.GET("/:id", h.GetTemplate)
		g.POST("", h.CreateTemplate)
		g.PUT("/:id", h.UpdateTemplate)
		g.DELETE("/:id", h.DeleteTemplate)
	}
}
