package agent_registry

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	runtimeskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResponsePowerXSyncErrorMapsAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/sync", func(c *gin.Context) {
		responsePowerXSyncError(c, "POWERX_SKILL_SYNC_FAILED", errors.New("forbidden: admin only"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)

	var payload struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.False(t, payload.Success)
	require.Equal(t, "POWERX_ADMIN_REQUIRED", payload.Error.Code)
	require.Contains(t, payload.Error.Message, "Gateway 凭证")
	require.Equal(t, "forbidden: admin only", payload.Error.Details["cause"])
}

func TestResponsePowerXSyncErrorKeepsFallbackForOtherErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/sync", func(c *gin.Context) {
		responsePowerXSyncError(c, "POWERX_AGENT_SYNC_FAILED", errors.New("upstream timeout"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "POWERX_AGENT_SYNC_FAILED")
	require.Contains(t, w.Body.String(), "upstream timeout")
}

func TestLocalRegistrationRewritesSkillCapabilityRefs(t *testing.T) {
	identity := registrationIdentity{PluginID: pluginID + ".local", Suffix: "local"}
	manifest := identity.rewriteSkillManifest(runtimeskills.PluginSkillManifest{
		SkillID:  "powerxplugin.template.basic",
		Provider: pluginID,
		Executor: runtimeskills.PluginSkillExecutor{
			Type:              "capability",
			Capability:        "powerxplugin.template",
			PrepareCapability: "com.powerx.plugins.base.template.prepare",
			ActionMap: map[string]string{
				"create": "com.powerx.plugins.base.template.create",
			},
		},
	})

	require.Equal(t, "powerxplugin.template.basic.local", manifest.SkillID)
	require.Equal(t, pluginID+".local", manifest.Provider)
	require.Equal(t, "powerxplugin.template", manifest.Executor.Capability)
	require.Equal(t, "com.powerx.plugins.base.local.template.prepare", manifest.Executor.PrepareCapability)
	require.Equal(t, "com.powerx.plugins.base.local.template.create", manifest.Executor.ActionMap["create"])
}

func TestLocalRegistrationRewritesRawManifestCapabilityRefs(t *testing.T) {
	identity := registrationIdentity{PluginID: pluginID + ".local", Suffix: "local"}
	raw := map[string]any{
		"skill_id":   "powerxplugin.template.basic",
		"provider":   pluginID,
		"capability": "powerxplugin.template",
		"action_capabilities": map[string]any{
			"create": "com.powerx.plugins.base.template.create",
		},
		"executor": map[string]any{
			"type":               "capability",
			"capability":         "powerxplugin.template",
			"prepare_capability": "com.powerx.plugins.base.template.prepare",
			"action_map": map[string]any{
				"create": "com.powerx.plugins.base.template.create",
			},
		},
	}

	out, ok := identity.rewriteManifest(raw).(map[string]any)
	require.True(t, ok)
	require.Equal(t, "powerxplugin.template.basic.local", out["skill_id"])
	require.Equal(t, pluginID+".local", out["provider"])
	require.Equal(t, "powerxplugin.template", out["capability"])
	executor := out["executor"].(map[string]any)
	require.Equal(t, "powerxplugin.template", executor["capability"])
	require.Equal(t, "com.powerx.plugins.base.local.template.prepare", executor["prepare_capability"])
	require.Equal(t, "com.powerx.plugins.base.local.template.create", executor["action_map"].(map[string]any)["create"])
	require.Equal(t, "com.powerx.plugins.base.local.template.create", out["action_capabilities"].(map[string]any)["create"])
}

func TestValidateTemplateSkillManifestRequiresPrepareCapability(t *testing.T) {
	err := validateTemplateSkillManifest(runtimeskills.PluginSkillManifest{
		Executor: runtimeskills.PluginSkillExecutor{
			Type:       "capability",
			Capability: "powerxplugin.template",
			ActionMap: map[string]string{
				"create": "com.powerx.plugins.base.template.create",
			},
		},
	})

	require.ErrorContains(t, err, "executor.prepare_capability is required")
}
