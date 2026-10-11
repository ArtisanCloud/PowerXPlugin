package runtime_ops

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	fwagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

type AgentLifecycleHandler struct{ deps *app.Deps }

func NewAgentLifecycleHandler(deps *app.Deps) *AgentLifecycleHandler {
	return &AgentLifecycleHandler{deps: deps}
}

func (h *AgentLifecycleHandler) Health(c *gin.Context) {
	lifecycle, agentUUID, ok := h.resolveLifecycle(c)
	if !ok {
		return
	}
	summary, err := lifecycle.GetHealthSummary(c.Request.Context(), agentUUID)
	if !h.respondLifecycleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, summary)
}

func (h *AgentLifecycleHandler) HealthHistory(c *gin.Context) {
	lifecycle, agentUUID, ok := h.resolveLifecycle(c)
	if !ok {
		return
	}
	rangeHours, ok := positiveQuery(c, "range_hours", 24, 24*365)
	if !ok {
		contracts.ResponseBadRequest(c, "invalid range_hours")
		return
	}
	limit, ok := positiveQuery(c, "limit", 100, 1000)
	if !ok {
		contracts.ResponseBadRequest(c, "invalid limit")
		return
	}
	history, err := lifecycle.ListHealthHistory(c.Request.Context(), agentUUID, rangeHours, limit)
	if !h.respondLifecycleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, history)
}

func (h *AgentLifecycleHandler) BridgeState(c *gin.Context) {
	lifecycle, agentUUID, ok := h.resolveLifecycle(c)
	if !ok {
		return
	}
	limit, ok := positiveQuery(c, "limit", 100, 1000)
	if !ok {
		contracts.ResponseBadRequest(c, "invalid limit")
		return
	}
	state, err := lifecycle.GetBridgeState(c.Request.Context(), agentUUID, limit)
	if !h.respondLifecycleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, state)
}

func (h *AgentLifecycleHandler) resolveLifecycle(c *gin.Context) (fwagent.LifecycleService, string, bool) {
	if h == nil || h.deps == nil || h.deps.AgentLifecycle == nil {
		contracts.ResponseServiceUnavailable(c, "agent lifecycle runtime is not configured", nil)
		return nil, "", false
	}
	if tenant := admincommon.ResolveTenantUUID(c); strings.TrimSpace(tenant) == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return nil, "", false
	}
	agentUUID := strings.TrimSpace(c.Param("agentUUID"))
	if agentUUID == "" {
		contracts.ResponseBadRequest(c, "agent_uuid is required")
		return nil, "", false
	}
	lifecycle, err := h.deps.AgentLifecycle.Lifecycle()
	if err != nil {
		var moduleErr *module.Error
		if errors.As(err, &moduleErr) {
			contracts.ResponseServiceUnavailable(c, moduleErr.Message, nil)
			return nil, "", false
		}
		contracts.ResponseServiceUnavailable(c, "agent lifecycle runtime is not configured", nil)
		return nil, "", false
	}
	return lifecycle, agentUUID, true
}

func (h *AgentLifecycleHandler) respondLifecycleError(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	if typed, ok := err.(*powerxagent.Error); ok && typed.Code == powerxagent.ErrCodeForbidden {
		contracts.ResponseError(c, http.StatusForbidden, typed.Code, typed.Message)
		return false
	}
	contracts.ResponseError(c, http.StatusBadGateway, "AGENT_UPSTREAM_DEPENDENCY", err.Error())
	return false
}

func positiveQuery(c *gin.Context, key string, fallback, maximum int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil && value > 0 && value <= maximum
}

var _ fwagent.LifecycleService = (*powerxagent.Client)(nil)
