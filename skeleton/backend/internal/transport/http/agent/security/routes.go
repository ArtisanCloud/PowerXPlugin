package security

import (
	"github.com/gin-gonic/gin"
	secobs "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/observability/security"
	agentsec "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/agent/security"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
)

// RegisterRoutes wires the agent security namespace (privacy consent endpoints).
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	logger := deps.RuntimeLogger(deps.Ctx, "agent_security_privacy", nil)
	guard := agentsec.NewPrivacyGuard(deps.DB, deps.Config, logger)
	auditPath := deps.Config.SecurityBaselineConfig().ConsentDefaults.AuditChannel
	auditWriter, _ := secobs.NewFileAuditWriter(auditPath)
	h := NewPrivacyHandler(deps, guard, auditWriter)
	toolgrant := NewToolGrantHandler(deps)
	sec := rg.Group("/security")
	{
		sec.GET("/privacy/consent", h.GetActiveConsent)
		sec.POST("/privacy/lifecycle", h.AcknowledgeLifecycleEvent)
		sec.POST("/toolgrants/verify", toolgrant.Verify)
	}
}
