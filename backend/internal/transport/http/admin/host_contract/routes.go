package host_contract

import (
	"github.com/example.com/framework-ci-fixture/backend/internal/shared/app"
	httpmw "github.com/example.com/framework-ci-fixture/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil {
		return
	}
	handler := NewHandler(deps)
	// Probe requires the authenticated tenant context; do not populate it from headers.
	group := admin.Group("/host-contract", httpmw.RequireRoot())
	group.POST("/probe", handler.Probe)
}
