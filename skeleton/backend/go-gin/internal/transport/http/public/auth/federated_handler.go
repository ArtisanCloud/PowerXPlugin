package auth

import (
	"net/http"
	"os"
	"strings"

	federatedContracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	authobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/auth"
	federatedService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// FederatedHandler 暴露扫码 challenge/callback 路由。
type FederatedHandler struct {
	deps          *app.Deps
	loginService  *federatedService.LoginService
	authMode      *federatedService.AuthModeService
	contextSvc    *federatedService.ContextService
	auditSvc      *authobs.FederatedAuditService
	defaultTenant string
}

func NewFederatedHandler(deps *app.Deps) *FederatedHandler {
	defaultTenant := ""
	if deps != nil && deps.Config != nil && deps.Config.GRPCUpstream != nil {
		defaultTenant = strings.TrimSpace(deps.Config.GRPCUpstream.TenantUUID)
	}
	return &FederatedHandler{
		deps:          deps,
		loginService:  federatedService.NewLoginService(),
		authMode:      federatedService.NewAuthModeService(os.Getenv("POWERX_FEDERATED_AUTH_MODE")),
		contextSvc:    federatedService.NewContextService(),
		auditSvc:      authobs.NewFederatedAuditService(app.PluginID),
		defaultTenant: defaultTenant,
	}
}

func RegisterRoutes(group *gin.RouterGroup, deps *app.Deps) {
	if group == nil {
		return
	}
	h := NewFederatedHandler(deps)
	g := group.Group("/auth/federated")
	g.Use(middleware.RequestTrace())
	g.POST("/challenge", h.Challenge)
	g.POST("/callback", h.Callback)
}

type challengeRequest struct {
	Provider    string `json:"provider"`
	TenantUUID  string `json:"tenant_uuid"`
	RedirectURI string `json:"redirect_uri"`
	Scope       string `json:"scope"`
}

type callbackRequest struct {
	Provider       string `json:"provider"`
	TenantUUID     string `json:"tenant_uuid"`
	State          string `json:"state"`
	Nonce          string `json:"nonce"`
	Code           string `json:"code"`
	SignatureValid *bool  `json:"signature_valid"`
}

func (h *FederatedHandler) Challenge(c *gin.Context) {
	if !h.authMode.FederatedEnabled() {
		contracts.ResponseServiceUnavailable(c, "当前模式未启用扫码登录", nil)
		return
	}
	runtime := pluginbootstrap.Federated()
	if runtime == nil || runtime.Factory == nil || runtime.Challenge == nil {
		contracts.ResponseServiceUnavailable(c, "联邦登录运行时未初始化", nil)
		return
	}
	var req challengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}
	provider, err := runtime.Factory.MustGet(req.Provider)
	if err != nil {
		contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, string(federatedContracts.ErrorCodeProviderNotFound), "渠道未配置", nil)
		return
	}

	tenantUUID := strings.TrimSpace(req.TenantUUID)
	if tenantUUID == "" {
		tenantUUID = h.defaultTenant
	}
	if tenantUUID == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid 必填")
		return
	}

	challenge, err := runtime.Challenge.Issue(c.Request.Context(), federatedContracts.ChallengeIssueRequest{
		TenantUUID: tenantUUID,
		Provider:   provider.Key(),
		TraceID:    c.GetString("request_id"),
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	authz, err := provider.BuildAuthorizeURL(c.Request.Context(), federatedContracts.AuthorizeRequest{
		TenantUUID:  tenantUUID,
		RedirectURI: strings.TrimSpace(req.RedirectURI),
		Scope:       strings.TrimSpace(req.Scope),
		State:       challenge.State,
		Nonce:       challenge.Nonce,
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"provider":      provider.Key(),
		"tenant_uuid":   tenantUUID,
		"state":         challenge.State,
		"nonce":         challenge.Nonce,
		"expires_at":    challenge.ExpiresAt,
		"authorize_url": authz.AuthorizeURL,
	})
}

func (h *FederatedHandler) Callback(c *gin.Context) {
	if !h.authMode.FederatedEnabled() {
		contracts.ResponseServiceUnavailable(c, "当前模式未启用扫码登录", nil)
		return
	}
	runtime := pluginbootstrap.Federated()
	if runtime == nil || runtime.Factory == nil || runtime.Challenge == nil || runtime.Risk == nil {
		contracts.ResponseServiceUnavailable(c, "联邦登录运行时未初始化", nil)
		return
	}
	var req callbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}
	provider, err := runtime.Factory.MustGet(req.Provider)
	if err != nil {
		contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, string(federatedContracts.ErrorCodeProviderNotFound), "渠道未配置", nil)
		return
	}
	tenantUUID := strings.TrimSpace(req.TenantUUID)
	if tenantUUID == "" {
		tenantUUID = h.defaultTenant
	}

	challenge, err := runtime.Challenge.ValidateAndConsume(c.Request.Context(), federatedContracts.ChallengeConsumeRequest{
		State:      strings.TrimSpace(req.State),
		Nonce:      strings.TrimSpace(req.Nonce),
		TenantUUID: tenantUUID,
		Provider:   provider.Key(),
	})
	if err != nil {
		h.auditSvc.Record(authobs.FederatedAuditEvent{
			Provider:       provider.Key(),
			TenantUUID:     tenantUUID,
			BindingOutcome: "challenge_rejected",
			RiskDecision:   "reject",
			ReasonCode:     string(federatedContracts.ErrorCodeInvalidChallenge),
			TraceID:        c.GetString("request_id"),
		})
		contracts.ResponseErrorWithDetails(c, http.StatusUnauthorized, string(federatedContracts.ErrorCodeInvalidChallenge), "登录失败，请稍后重试", gin.H{"error_code": err.Error()})
		return
	}
	signatureValid := true
	if req.SignatureValid != nil {
		signatureValid = *req.SignatureValid
	}
	decision := runtime.Risk.EvaluateCallback(c.Request.Context(), federatedContracts.RiskEvaluateRequest{
		Challenge:      challenge,
		State:          strings.TrimSpace(req.State),
		Nonce:          strings.TrimSpace(req.Nonce),
		Code:           strings.TrimSpace(req.Code),
		TenantUUID:     tenantUUID,
		SignatureValid: signatureValid,
	})
	if !decision.Allowed {
		h.auditSvc.Record(authobs.FederatedAuditEvent{
			Provider:       provider.Key(),
			TenantUUID:     challenge.TenantUUID,
			BindingOutcome: "risk_rejected",
			RiskDecision:   "reject",
			ReasonCode:     string(decision.Code),
			TraceID:        challenge.TraceID,
			Evidence:       decision.Evidence,
		})
		contracts.ResponseErrorWithDetails(c, http.StatusUnauthorized, string(decision.Code), "登录失败，请稍后重试", gin.H{"risk_code": decision.Code})
		return
	}
	token, err := provider.ExchangeCode(c.Request.Context(), federatedContracts.ExchangeCodeRequest{Code: req.Code})
	if err != nil {
		n := h.contextSvc.NormalizeUnavailableError(resolveIAMMode(h.deps), err)
		h.auditSvc.Record(authobs.FederatedAuditEvent{
			Provider:       provider.Key(),
			TenantUUID:     challenge.TenantUUID,
			BindingOutcome: "provider_exchange_failed",
			RiskDecision:   "reject",
			ReasonCode:     n.Code,
			TraceID:        challenge.TraceID,
		})
		contracts.ResponseErrorWithDetails(c, http.StatusUnauthorized, string(federatedContracts.ErrorCodeUnauthorized), "登录失败，请稍后重试", nil)
		return
	}
	identity, err := provider.ResolveIdentity(c.Request.Context(), federatedContracts.ResolveIdentityRequest{Token: token})
	if err != nil {
		h.auditSvc.Record(authobs.FederatedAuditEvent{
			Provider:       provider.Key(),
			TenantUUID:     challenge.TenantUUID,
			BindingOutcome: "identity_resolve_failed",
			RiskDecision:   "reject",
			ReasonCode:     string(federatedContracts.ErrorCodeUnauthorized),
			TraceID:        challenge.TraceID,
		})
		contracts.ResponseErrorWithDetails(c, http.StatusUnauthorized, string(federatedContracts.ErrorCodeUnauthorized), "登录失败，请稍后重试", nil)
		return
	}
	result := h.loginService.Build(identity, challenge.TenantUUID)
	result.Context = h.contextSvc.NormalizeContext(resolveIAMMode(h.deps), result.Context)
	h.auditSvc.Record(authobs.FederatedAuditEvent{
		Provider:         provider.Key(),
		TenantUUID:       challenge.TenantUUID,
		ExternalIdentity: identity.ExternalUserID,
		BindingOutcome:   "login_success",
		RiskDecision:     "allow",
		ReasonCode:       "ok",
		TraceID:          challenge.TraceID,
	})
	contracts.ResponseSuccess(c, gin.H{"tokens": result, "context": result.Context})
}

func resolveIAMMode(deps *app.Deps) string {
	if deps == nil {
		return "standalone"
	}
	mode := strings.TrimSpace(deps.IAMMode.String())
	if mode == "" {
		return "standalone"
	}
	return mode
}
