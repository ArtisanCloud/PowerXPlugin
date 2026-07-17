package skills

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HTTPAdapter struct {
	registry *Registry
}

func NewHTTPAdapter(registry *Registry) *HTTPAdapter {
	if registry == nil {
		registry = NewRegistry()
	}
	return &HTTPAdapter{registry: registry}
}

func (a *HTTPAdapter) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", a.List)
	group.GET("/", a.List)
	group.GET("/:skill_id/schema", a.Schema)
}

func (a *HTTPAdapter) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": a.registry.List()})
}

func (a *HTTPAdapter) Schema(c *gin.Context) {
	schema, ok := a.registry.Schema(c.Param("skill_id"), c.Query("version"))
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResult(PluginSkillInvocation{SkillID: c.Param("skill_id")}, NewError(ErrCodeNotFound, "skill is not registered")))
		return
	}
	c.JSON(http.StatusOK, schema)
}
