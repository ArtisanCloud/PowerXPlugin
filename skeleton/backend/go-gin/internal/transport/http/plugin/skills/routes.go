package skills

import (
	"errors"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	samples "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/skills"
	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if deps == nil || deps.DB == nil {
		panic(errors.New("plugin skill routes require database"))
	}
	group := rg.Group("/plugin/skills")
	samples.NewTemplateRegistryHTTPAdapter(deps.DB).RegisterRoutes(group)
}
