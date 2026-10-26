package ai_settings

import (
	"errors"
	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"net/http"

	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	localaisettings "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/ai_settings"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil || deps == nil {
		return
	}
	h := &Handler{mode: deps.ProviderMode, deps: deps}
	group := admin.Group("/ai-settings")
	registerMediaRoutes(group, h)
	group.GET("/mode", h.Mode)
	group.GET("/summary", h.Summary)
	group.GET("/provider-profiles", h.ProviderProfiles)
	group.GET("/model-profiles", h.ModelProfiles)
	group.GET("/routing", h.Routing)
	group.GET("/health", h.Health)
	group.GET("/local-setting", h.LocalSetting)
	group.GET("/catalog/providers", h.CatalogProviders)
	group.GET("/catalog/models", h.CatalogModels)
	group.PUT("/local-setting", h.SaveLocalSetting)
	group.POST("/test-connection", h.TestConnection)
	group.POST("/quick-call", h.QuickCall)
}

func (h *Handler) CatalogProviders(c *gin.Context) {
	modality := c.Query("modality")
	if modality == "" {
		mediaFailure(c, &powerxai.HTTPError{StatusCode: 400, ReasonCode: "AI_REQUEST_INVALID"})
		return
	}
	if source := c.Query("source"); source != "local" && source != "powerx" {
		mediaFailure(c, &powerxai.HTTPError{StatusCode: 400, ReasonCode: "AI_REQUEST_INVALID"})
		return
	}
	if c.Query("source") == "powerx" {
		if modality == "llm" {
			h.llmCatalog(c, true)
			return
		}
		capability, ok := mediaCapability(modality)
		if !ok {
			mediaFailure(c, &powerxai.HTTPError{StatusCode: 422, ReasonCode: "AI_MODEL_CAPABILITY_UNSUPPORTED"})
			return
		}
		if h.deps.PowerXAISettings == nil {
			mediaFailure(c, &powerxai.HTTPError{StatusCode: 503, ReasonCode: "AI_CLIENT_UNAVAILABLE"})
			return
		}
		out, err := h.deps.PowerXAISettings.CatalogProviders(c.Request.Context(), capability, requestID(c))
		if err != nil {
			mediaFailure(c, err)
			return
		}
		items := make([]map[string]string, 0, len(out.Providers))
		for _, p := range out.Providers {
			items = append(items, map[string]string{"id": p.Provider, "name": p.Label})
		}
		contracts.ResponseSuccess(c, gin.H{"items": items})
		return
	}
	svc, _, ok := h.localService(c)
	if !ok {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": svc.CatalogProviders(modality)})
}
func (h *Handler) CatalogModels(c *gin.Context) {
	modality, provider := c.Query("modality"), c.Query("provider")
	if modality == "" || provider == "" {
		mediaFailure(c, &powerxai.HTTPError{StatusCode: 400, ReasonCode: "AI_REQUEST_INVALID"})
		return
	}
	if source := c.Query("source"); source != "local" && source != "powerx" {
		mediaFailure(c, &powerxai.HTTPError{StatusCode: 400, ReasonCode: "AI_REQUEST_INVALID"})
		return
	}
	if c.Query("source") == "powerx" {
		if modality == "llm" {
			h.llmCatalog(c, false)
			return
		}
		capability, ok := mediaCapability(modality)
		if !ok {
			mediaFailure(c, &powerxai.HTTPError{StatusCode: 422, ReasonCode: "AI_MODEL_CAPABILITY_UNSUPPORTED"})
			return
		}
		if h.deps.PowerXAISettings == nil {
			mediaFailure(c, &powerxai.HTTPError{StatusCode: 503, ReasonCode: "AI_CLIENT_UNAVAILABLE"})
			return
		}
		out, err := h.deps.PowerXAISettings.CatalogModels(c.Request.Context(), capability, provider, c.Query("app"), requestID(c))
		if err != nil {
			mediaFailure(c, err)
			return
		}
		contracts.ResponseSuccess(c, out)
		return
	}
	svc, _, ok := h.localService(c)
	if !ok {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": svc.CatalogModels(modality, provider, c.Query("app"))})
}
func mediaCapability(modality string) (string, bool) {
	switch modality {
	case "vlm":
		return powerxai.VisionUnderstanding, true
	case "image":
		return powerxai.ImageGeneration, true
	case "video":
		return powerxai.VideoGeneration, true
	case "audio_tts":
		return powerxai.SpeechSynthesis, true
	}
	return "", false
}
func (h *Handler) llmCatalog(c *gin.Context, providers bool) {
	if h.deps.PowerXAISettings == nil {
		mediaFailure(c, &powerxai.HTTPError{StatusCode: 503, ReasonCode: "AI_CLIENT_UNAVAILABLE"})
		return
	}
	if providers {
		out, err := h.deps.PowerXAISettings.SettingsCatalogProviders(c.Request.Context(), requestID(c))
		if err != nil {
			mediaFailure(c, err)
			return
		}
		contracts.ResponseSuccess(c, out)
		return
	}
	out, err := h.deps.PowerXAISettings.SettingsCatalogModels(c.Request.Context(), c.Query("provider"), c.Query("app"), requestID(c))
	if err != nil {
		mediaFailure(c, err)
		return
	}
	contracts.ResponseSuccess(c, out)
}

type localSettingRequest struct {
	Source        string         `json:"source" binding:"required,oneof=local powerx"`
	Environment   string         `json:"environment" binding:"required,oneof=development production"`
	Modality      string         `json:"modality" binding:"required,oneof=llm vlm image embedding audio_tts audio_asr video model3d rerank"`
	Provider      string         `json:"provider" binding:"required"`
	App           string         `json:"app"`
	ModelKey      string         `json:"model_key" binding:"required"`
	Endpoint      string         `json:"endpoint"`
	CredentialRef string         `json:"credential_ref"`
	Parameters    map[string]any `json:"parameters"`
}

func (h *Handler) localService(c *gin.Context) (*localaisettings.LocalSettingsService, string, bool) {
	if h == nil || h.deps == nil || h.deps.LocalAISettings == nil {
		admincommon.ProviderUnavailable(c, "AI_SETTINGS_LOCAL_MODE_REQUIRED", "Local AI settings are unavailable in delegated mode", h.diagnostics())
		return nil, "", false
	}
	tenantUUID, ok := authx.TenantUUIDFromContext(c.Request.Context())
	if !ok || tenantUUID == "" {
		contracts.ResponseError(c, http.StatusUnauthorized, "UNAUTHORIZED", "UNAUTHORIZED")
		return nil, "", false
	}
	return h.deps.LocalAISettings, tenantUUID, true
}

func (h *Handler) LocalSetting(c *gin.Context) {
	svc, tenantUUID, ok := h.localService(c)
	if !ok {
		return
	}
	environment, modality, source := c.Query("environment"), c.Query("modality"), c.Query("source")
	if environment == "" || modality == "" || (source != "local" && source != "powerx") {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "environment, modality and source are required")
		return
	}
	item, err := svc.Get(c.Request.Context(), tenantUUID, environment, modality, source)
	respondDelegated(c, item, err)
}

func (h *Handler) SaveLocalSetting(c *gin.Context) {
	svc, tenantUUID, ok := h.localService(c)
	if !ok {
		return
	}
	var req localSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid local AI setting")
		return
	}
	item, err := svc.Save(c.Request.Context(), tenantUUID, req.Source, localaisettings.SaveInput{Environment: req.Environment, Modality: req.Modality, Provider: req.Provider, App: req.App, ModelKey: req.ModelKey, Endpoint: req.Endpoint, CredentialRef: req.CredentialRef, Parameters: req.Parameters})
	respondDelegated(c, item, err)
}

// Both actions execute the selected capability. Media generation returns a task,
// never a synthetic connectivity result. Credentials are selected at startup.
func (h *Handler) TestConnection(c *gin.Context) { h.invokeSettingTest(c) }
func (h *Handler) QuickCall(c *gin.Context)      { h.invokeSettingTest(c) }

type Handler struct {
	mode fwprovider.Mode
	deps *app.Deps
}

func (h *Handler) Mode(c *gin.Context) {
	contracts.ResponseSuccess(c, h.diagnostics())
}

func (h *Handler) Summary(c *gin.Context) {
	if h.deps != nil && h.deps.AISettings != nil {
		out, err := h.deps.AISettings.Summary(c.Request.Context(), requestID(c))
		respondDelegated(c, out, err)
		return
	}
	admincommon.ProviderUnavailable(c, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", "AI settings provider is not configured for this plugin", h.diagnostics())
}

func (h *Handler) ProviderProfiles(c *gin.Context) {
	if h.deps != nil && h.deps.AISettings != nil {
		items, err := h.deps.AISettings.ProviderProfiles(c.Request.Context(), requestID(c))
		respondDelegated(c, gin.H{"items": items}, err)
		return
	}
	admincommon.ProviderUnavailable(c, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", "AI settings provider is not configured for this plugin", h.diagnostics())
}

func (h *Handler) ModelProfiles(c *gin.Context) {
	if h.deps != nil && h.deps.AISettings != nil {
		items, err := h.deps.AISettings.ModelProfiles(c.Request.Context(), requestID(c))
		respondDelegated(c, gin.H{"items": items}, err)
		return
	}
	admincommon.ProviderUnavailable(c, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", "AI settings provider is not configured for this plugin", h.diagnostics())
}

func (h *Handler) Routing(c *gin.Context) {
	if h.deps != nil && h.deps.AISettings != nil {
		out, err := h.deps.AISettings.Routing(c.Request.Context(), requestID(c))
		respondDelegated(c, out, err)
		return
	}
	admincommon.ProviderUnavailable(c, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", "AI settings provider is not configured for this plugin", h.diagnostics())
}

func (h *Handler) Health(c *gin.Context) {
	if h.deps != nil && h.deps.AISettings != nil {
		out, err := h.deps.AISettings.Health(c.Request.Context(), requestID(c))
		respondDelegated(c, out, err)
		return
	}
	admincommon.ProviderUnavailable(c, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", "AI settings provider is not configured for this plugin", h.diagnostics())
}

func (h *Handler) diagnostics() admincommon.ProviderDiagnostics {
	mode := string(fwprovider.ModeLocal)
	if h != nil && h.mode.String() != "" {
		mode = h.mode.String()
	}
	available := h != nil && h.deps != nil && h.deps.AISettings != nil
	out := admincommon.NewProviderDiagnostics(mode, h.isDelegated() && available, !h.isDelegated() && available)
	out.ReadOnly = true
	return out
}

func (h *Handler) isDelegated() bool {
	return h != nil && h.mode == fwprovider.ModeDelegated
}

func respondDelegated(c *gin.Context, data any, err error) {
	if err != nil {
		var aiErr *powerxai.HTTPError
		if errors.As(err, &aiErr) {
			mediaFailure(c, err)
			return
		}
		var contractErr *hostapi.HTTPError
		if errors.As(err, &contractErr) {
			contracts.ResponseError(c, contractErr.StatusCode, contractErr.ReasonCode, contractErr.ReasonCode)
			return
		}
		var gatewayErr *frameworkgateway.InvocationError
		if errors.As(err, &gatewayErr) && gatewayErr.StatusCode >= http.StatusBadRequest && gatewayErr.StatusCode < http.StatusInternalServerError {
			contracts.ResponseError(c, gatewayErr.StatusCode, "AI_SETTINGS_CATALOG_ACCESS_DENIED", "AI_SETTINGS_CATALOG_ACCESS_DENIED")
			return
		}
		contracts.ResponseError(c, http.StatusBadGateway, "AI_SETTINGS_GATEWAY_FAILED", err.Error())
		return
	}
	contracts.ResponseSuccess(c, data)
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v := c.GetHeader("X-Request-ID"); v != "" {
		return v
	}
	return c.GetString("request_id")
}
