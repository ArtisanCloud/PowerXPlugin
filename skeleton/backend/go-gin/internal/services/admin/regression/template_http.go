package regression

import (
	"net/http"

	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	TemplatesHTTP "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/templates"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Use the production routes, binding and handlers with the runner's trusted
// actor. This in-process boundary verifies HTTP validation, not session or Core
// authorization. It deliberately has no outbound publisher or network listener.
func templateHTTP(db *gorm.DB, actor Regression.Actor) http.Handler {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		Middleware.SetTenantContext(c, Middleware.TenantContext{
			TenantUUID: actor.TenantUUID, UserUUID: actor.UUID, MemberUUID: actor.MemberUUID,
		})
		c.Next()
	})
	TemplatesHTTP.RegisterAPIRoutes(engine.Group("/api/v1"), &App.Deps{DB: db})
	return engine
}
