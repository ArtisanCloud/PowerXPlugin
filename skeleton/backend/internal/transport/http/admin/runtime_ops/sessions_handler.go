package runtime_ops

import (
	"net/http"

	"github.com/gin-gonic/gin"
	runtimeops "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
	"gorm.io/gorm"
)

// SessionsHandler handles MCP session admin endpoints.
type SessionsHandler struct {
	svc *runtimeops.MCPSessionService
}

// NewSessionsHandler constructs handler with dependencies.
func NewSessionsHandler(deps *app.Deps) *SessionsHandler {
	var db *gorm.DB
	if deps != nil {
		db = deps.DB
	}
	return &SessionsHandler{svc: runtimeops.NewMCPSessionService(db)}
}

// Register handles MCP REGISTER requests (placeholder).
func (h *SessionsHandler) Register(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "MCP session register not implemented"})
}
