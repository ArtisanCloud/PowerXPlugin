package security

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/privacy"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	secobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/security"
	agentsec "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/agent/security"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type PrivacyHandler struct {
	guard *agentsec.PrivacyGuard
	audit *secobs.AuditWriter
	deps  *app.Deps
}

func NewPrivacyHandler(deps *app.Deps, guard *agentsec.PrivacyGuard, audit *secobs.AuditWriter) *PrivacyHandler {
	return &PrivacyHandler{guard: guard, audit: audit, deps: deps}
}

func (h *PrivacyHandler) GetActiveConsent(c *gin.Context) {
	tenantUint, ok := middleware.TenantIDFromContext(c.Request.Context())
	if !ok || tenantUint == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context missing"})
		return
	}
	tenantID := fmt.Sprintf("%d", tenantUint)
	tokens, err := h.guard.ActiveConsentTokens(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scope := make(map[string]struct{})
	for _, token := range tokens {
		values, _ := token.ScopeValues()
		for _, asset := range values {
			scope[asset] = struct{}{}
		}
	}
	assets := make([]string, 0, len(scope))
	for asset := range scope {
		assets = append(assets, asset)
	}
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "assets": assets})
}

func (h *PrivacyHandler) AcknowledgeLifecycleEvent(c *gin.Context) {
	var payload struct {
		EventType string                 `json:"event_type" binding:"required"`
		AssetKey  string                 `json:"asset_key" binding:"required"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	tenantUint, ok := middleware.TenantIDFromContext(c.Request.Context())
	if !ok || tenantUint == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context missing"})
		return
	}
	tenantID := fmt.Sprintf("%d", tenantUint)
	event := &privacy.LifecycleEvent{
		TenantID:   tenantID,
		EventType:  payload.EventType,
		AssetKey:   payload.AssetKey,
		RecordedBy: "agent",
		Status:     privacy.LifecycleStatusSucceeded,
	}
	if payload.Metadata != nil {
		filtered := h.guard.FilterAIData(payload.Metadata)
		blob, _ := json.Marshal(filtered)
		event.Payload = datatypes.JSON(blob)
	}
	if _, err := h.guard.RecordLifecycleEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.audit != nil {
		if err := h.audit.EmitLifecycleSuccess(tenantID, payload.EventType, "agent", payload.Metadata); err != nil {
			if h.deps != nil {
				if logger := h.deps.RuntimeLogger(c.Request.Context(), "agent_privacy_handler", nil); logger != nil {
					logger.WithError(err).Warn("failed to emit lifecycle success audit event")
				}
			}
		}
	}
	c.Status(http.StatusAccepted)
}
