package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	contract "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/contract"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	federatedContracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/repository/iam"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	authobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/auth"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	federatedService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FederatedHandler 暴露扫码 challenge/callback 路由。
type FederatedHandler struct {
	deps          *app.Deps
	loginService  *federatedService.LoginService
	authMode      *federatedService.AuthModeService
	contextSvc    *federatedService.ContextService
	wecomConfig   *federatedService.WeComConfigService
	dingtalkCfg   *federatedService.DingTalkConfigService
	larkCfg       *federatedService.LarkConfigService
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
		wecomConfig:   federatedService.NewWeComConfigService(defaultDB(deps)),
		dingtalkCfg:   federatedService.NewDingTalkConfigService(defaultDB(deps)),
		larkCfg:       federatedService.NewLarkConfigService(defaultDB(deps)),
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
	g.GET("/browser/callback", h.BrowserCallback)
	group.GET("/webhooks/wecom/tenant/:tenant_uuid/corp/:corp_id/app/:app_id", h.WeComServerCallback)
	group.POST("/webhooks/wecom/tenant/:tenant_uuid/corp/:corp_id/app/:app_id", h.WeComServerCallback)
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
		RedirectURI: h.resolveChallengeRedirectURI(c, provider.Key(), tenantUUID, challenge.State, challenge.Nonce, strings.TrimSpace(req.RedirectURI)),
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
	token, err := provider.ExchangeCode(c.Request.Context(), federatedContracts.ExchangeCodeRequest{
		Code:       req.Code,
		TenantUUID: challenge.TenantUUID,
	})
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
	authobs.RecordFederatedLoginSuccess(app.PluginID, challenge.TenantUUID)
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

func (h *FederatedHandler) BrowserCallback(c *gin.Context) {
	providerKey := strings.TrimSpace(c.Query("provider"))
	tenantUUID := strings.TrimSpace(c.Query("tenant_uuid"))
	state := strings.TrimSpace(c.Query("state"))
	nonce := strings.TrimSpace(c.Query("nonce"))
	code := strings.TrimSpace(c.Query("code"))
	frontRedirectRaw := strings.TrimSpace(c.Query("front_redirect_uri"))

	if providerKey == "" {
		providerKey = "wecom"
	}
	if tenantUUID == "" {
		tenantUUID = h.defaultTenant
	}
	if state == "" || nonce == "" || code == "" || tenantUUID == "" {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "扫码参数不完整，请重试")
		return
	}

	runtime := pluginbootstrap.Federated()
	if runtime == nil || runtime.Factory == nil || runtime.Challenge == nil || runtime.Risk == nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "联邦登录运行时未初始化")
		return
	}
	provider, err := runtime.Factory.MustGet(providerKey)
	if err != nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "渠道未配置")
		return
	}

	challenge, err := runtime.Challenge.ValidateAndConsume(c.Request.Context(), federatedContracts.ChallengeConsumeRequest{
		State:      state,
		Nonce:      nonce,
		TenantUUID: tenantUUID,
		Provider:   provider.Key(),
	})
	if err != nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "扫码会话已过期，请重新发起")
		return
	}

	decision := runtime.Risk.EvaluateCallback(c.Request.Context(), federatedContracts.RiskEvaluateRequest{
		Challenge:      challenge,
		State:          state,
		Nonce:          nonce,
		Code:           code,
		TenantUUID:     tenantUUID,
		SignatureValid: true,
	})
	if !decision.Allowed {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "登录风控校验失败")
		return
	}

	token, err := provider.ExchangeCode(c.Request.Context(), federatedContracts.ExchangeCodeRequest{
		Code:       code,
		TenantUUID: challenge.TenantUUID,
	})
	if err != nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "授权码兑换失败")
		return
	}
	identity, err := provider.ResolveIdentity(c.Request.Context(), federatedContracts.ResolveIdentityRequest{Token: token})
	if err != nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "身份解析失败")
		return
	}
	memberID, bindErr := h.resolveFederatedMemberID(c.Request.Context(), challenge.TenantUUID, provider.Key(), identity)
	if bindErr != nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "未找到可用成员绑定，请先同步组织并完成绑定")
		return
	}
	tokens, _, issueErr := h.issueLocalFederatedTokens(c.Request.Context(), challenge.TenantUUID, memberID, provider.Key(), identity.ExternalUserID)
	if issueErr != nil {
		h.redirectBrowserCallbackError(c, frontRedirectRaw, "本地会话签发失败")
		return
	}
	authobs.RecordFederatedLoginSuccess(app.PluginID, challenge.TenantUUID)

	redirectURL := resolveFrontRedirectURL(frontRedirectRaw)
	q := redirectURL.Query()
	q.Set("tenant_uuid", challenge.TenantUUID)
	q.Set("login_tab", "federated")
	q.Set("provider", provider.Key())
	q.Set("fed_ok", "1")
	q.Set("fed_access_token", tokens.AccessToken)
	q.Set("fed_refresh_token", tokens.RefreshToken)
	q.Set("fed_token_type", tokens.TokenType)
	q.Set("fed_expires_in", fmt.Sprintf("%d", tokens.ExpiresIn))
	q.Set("fed_scope", firstNonEmpty(strings.TrimSpace(tokens.Scope), provider.Key(), "access"))
	redirectURL.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, redirectURL.String())
}

func (h *FederatedHandler) resolveFederatedMemberID(ctx context.Context, tenantUUID, provider string, identity federatedContracts.ExternalIdentity) (uint64, error) {
	if h == nil || h.deps == nil || h.deps.DB == nil {
		return 0, fmt.Errorf("federated binding repository unavailable")
	}
	repo := iamrepo.NewFederatedBindingRepository(h.deps.DB)
	scopeCandidates := []string{strings.TrimSpace(identity.TenantScope), ""}
	seenScope := map[string]struct{}{}
	idCandidates := externalIdentityCandidates(identity, provider)
	for _, externalID := range idCandidates {
		for _, scope := range scopeCandidates {
			normScope := strings.TrimSpace(scope)
			key := normScope + "::" + externalID
			if _, ok := seenScope[key]; ok {
				continue
			}
			seenScope[key] = struct{}{}
			binding, err := repo.GetActiveByExternalScoped(ctx, tenantUUID, provider, normScope, externalID)
			if err != nil {
				return 0, err
			}
			if binding != nil && binding.MemberID > 0 {
				return binding.MemberID, nil
			}
		}
	}
	return 0, fmt.Errorf("active federated binding not found")
}

func externalIdentityCandidates(identity federatedContracts.ExternalIdentity, provider string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 6)
	appendCandidate := func(raw string) {
		v := strings.TrimSpace(raw)
		if v == "" {
			return
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
		prefix := strings.TrimSpace(provider) + ":"
		if strings.HasPrefix(v, prefix) {
			trimmed := strings.TrimSpace(strings.TrimPrefix(v, prefix))
			if trimmed != "" {
				if _, ok := seen[trimmed]; !ok {
					seen[trimmed] = struct{}{}
					out = append(out, trimmed)
				}
			}
		}
	}
	appendCandidate(identity.ExternalUserID)
	appendCandidate(identity.UnionID)
	appendCandidate(identity.OpenID)
	if identity.Raw != nil {
		appendCandidate(fmt.Sprintf("%v", identity.Raw["external_id"]))
		appendCandidate(fmt.Sprintf("%v", identity.Raw["user_id"]))
		appendCandidate(fmt.Sprintf("%v", identity.Raw["open_id"]))
	}
	return out
}

func (h *FederatedHandler) issueLocalFederatedTokens(ctx context.Context, tenantUUID string, memberID uint64, provider, externalUserID string) (*iamservice.AuthTokens, *iamservice.UserContext, error) {
	if h == nil || h.deps == nil || h.deps.IAMDirectory == nil {
		return nil, nil, fmt.Errorf("iam directory unavailable")
	}
	issuer, ok := h.deps.IAMDirectory.(interface {
		LoginByFederated(context.Context, iamservice.FederatedLoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error)
	})
	if !ok {
		return nil, nil, fmt.Errorf("iam directory does not support federated token issuing")
	}
	return issuer.LoginByFederated(ctx, iamservice.FederatedLoginRequest{
		TenantUUID:     strings.TrimSpace(tenantUUID),
		MemberID:       memberID,
		Provider:       strings.TrimSpace(provider),
		ExternalUserID: strings.TrimSpace(externalUserID),
	})
}

func firstNonEmpty(values ...string) string {
	for _, raw := range values {
		if strings.TrimSpace(raw) != "" {
			return strings.TrimSpace(raw)
		}
	}
	return ""
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

// WeComServerCallback 处理企业微信服务器回调（URL 验证 + 消息解密接收）。
func (h *FederatedHandler) WeComServerCallback(c *gin.Context) {
	tenantUUID := strings.TrimSpace(c.Param("tenant_uuid"))
	corpID := strings.TrimSpace(c.Param("corp_id"))
	appID := strings.TrimSpace(c.Param("app_id"))
	if tenantUUID == "" || corpID == "" || appID == "" {
		pxlog.WarnCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
			"component":   "iam.wecom_callback",
			"request_id":  c.GetString("request_id"),
			"tenant_uuid": tenantUUID,
			"corp_id":     corpID,
			"app_id":      appID,
			"biz_scene":   "wecom_callback_validate",
			"biz_domain":  "iam",
		}), "wecom callback path params missing")
		contracts.ResponseBadRequest(c, "wecom callback params missing: tenant_uuid/corp_id/app_id")
		return
	}
	wecomCfg, err := h.wecomConfig.ResolveCallbackConfig(c.Request.Context(), tenantUUID, corpID, appID)
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
			"component":   "iam.wecom_callback",
			"request_id":  c.GetString("request_id"),
			"tenant_uuid": tenantUUID,
			"corp_id":     corpID,
			"app_id":      appID,
			"error":       err.Error(),
			"biz_scene":   "wecom_callback_config_resolve",
			"biz_domain":  "iam",
		}), "wecom callback config resolve failed")
		contracts.ResponseBadRequest(c, "load wecom config failed: "+err.Error())
		return
	}

	callbackURL := resolveWeComCallbackURL(c, wecomCfg.CallbackHost)
	pxlog.InfoCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
		"component":         "iam.wecom_callback",
		"request_id":        c.GetString("request_id"),
		"tenant_uuid":       tenantUUID,
		"corp_id":           corpID,
		"app_id":            appID,
		"configured_host":   wecomCfg.CallbackHost,
		"resolved_callback": callbackURL,
		"request_host":      c.Request.Host,
		"x_forwarded_host":  strings.TrimSpace(c.GetHeader("X-Forwarded-Host")),
		"x_forwarded_proto": strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")),
		"x_forwarded_for":   strings.TrimSpace(c.GetHeader("X-Forwarded-For")),
		"user_agent":        strings.TrimSpace(c.GetHeader("User-Agent")),
		"biz_scene":         "wecom_callback_url_resolve",
		"biz_domain":        "iam",
	}), "wecom callback url resolved")
	appSDK, err := work.NewWork(&work.UserConfig{
		CorpID:      wecomCfg.CorpID,
		AgentID:     wecomCfg.AgentID,
		Secret:      wecomCfg.Secret,
		Token:       wecomCfg.Token,
		AESKey:      wecomCfg.AESKey,
		CallbackURL: callbackURL,
		OAuth: work.OAuth{
			Callback: callbackURL,
			Scopes:   []string{"snsapi_privateinfo"},
		},
		HttpDebug: wecomCfg.HttpDebug,
	})
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
			"component":         "iam.wecom_callback",
			"request_id":        c.GetString("request_id"),
			"tenant_uuid":       tenantUUID,
			"corp_id":           corpID,
			"app_id":            appID,
			"configured_host":   wecomCfg.CallbackHost,
			"resolved_callback": callbackURL,
			"request_host":      c.Request.Host,
			"x_forwarded_host":  strings.TrimSpace(c.GetHeader("X-Forwarded-Host")),
			"x_forwarded_proto": strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")),
			"error":             err.Error(),
			"biz_scene":         "wecom_callback_sdk_init",
			"biz_domain":        "iam",
		}), "wecom callback sdk init failed")
		contracts.ResponseBadRequest(c, "init wecom sdk failed: "+err.Error())
		return
	}

	if c.Request.Method == http.MethodGet {
		resp, err := appSDK.Server.VerifyURL(c.Request)
		if err != nil {
			pxlog.WarnCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
				"component":      "iam.wecom_callback",
				"request_id":     c.GetString("request_id"),
				"tenant_uuid":    tenantUUID,
				"corp_id":        corpID,
				"app_id":         appID,
				"msg_signature":  strings.TrimSpace(c.Query("msg_signature")),
				"timestamp":      strings.TrimSpace(c.Query("timestamp")),
				"nonce":          strings.TrimSpace(c.Query("nonce")),
				"error":          err.Error(),
				"callback_route": c.Request.URL.Path,
				"biz_scene":      "wecom_callback_verify",
				"biz_domain":     "iam",
			}), "wecom callback verify url failed")
			contracts.ResponseBadRequest(c, "verify callback url failed: "+err.Error())
			return
		}
		if resp == nil {
			c.String(http.StatusOK, kernel.SUCCESS_EMPTY_RESPONSE)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.String(resp.StatusCode, string(body))
		return
	}

	resp, err := appSDK.Server.Notify(c.Request, func(event contract.EventInterface) interface{} {
		h.auditSvc.Record(authobs.FederatedAuditEvent{
			Provider:       "wecom",
			TenantUUID:     tenantUUID,
			BindingOutcome: "wecom_server_event_received",
			RiskDecision:   "allow",
			ReasonCode:     fmt.Sprintf("%T", event),
			TraceID:        c.GetString("request_id"),
		})
		return kernel.SUCCESS_EMPTY_RESPONSE
	})
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
			"component":      "iam.wecom_callback",
			"request_id":     c.GetString("request_id"),
			"tenant_uuid":    tenantUUID,
			"corp_id":        corpID,
			"app_id":         appID,
			"msg_signature":  strings.TrimSpace(c.Query("msg_signature")),
			"timestamp":      strings.TrimSpace(c.Query("timestamp")),
			"nonce":          strings.TrimSpace(c.Query("nonce")),
			"error":          err.Error(),
			"callback_route": c.Request.URL.Path,
			"biz_scene":      "wecom_callback_notify",
			"biz_domain":     "iam",
		}), "wecom callback notify failed")
		contracts.ResponseBadRequest(c, "notify callback failed: "+err.Error())
		return
	}
	if resp == nil {
		c.String(http.StatusOK, kernel.SUCCESS_EMPTY_RESPONSE)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.String(resp.StatusCode, string(body))
}

func defaultDB(deps *app.Deps) *gorm.DB {
	if deps == nil {
		return nil
	}
	return deps.DB
}

func (h *FederatedHandler) resolveChallengeRedirectURI(c *gin.Context, providerKey, tenantUUID, state, nonce, rawRedirectURI string) string {
	redirectURI := strings.TrimSpace(rawRedirectURI)
	if redirectURI == "" {
		return redirectURI
	}
	callbackHost, err := h.resolveProviderCallbackHost(c, strings.TrimSpace(providerKey), tenantUUID)
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(c.Request.Context(), map[string]interface{}{
			"component":   "iam.federated_challenge",
			"request_id":  c.GetString("request_id"),
			"tenant_uuid": tenantUUID,
			"provider":    strings.TrimSpace(providerKey),
			"error":       err.Error(),
			"biz_scene":   "federated_challenge_resolve_redirect",
			"biz_domain":  "iam",
		}), "federated challenge resolve config failed, fallback to original redirect_uri")
		return redirectURI
	}
	rewritten, ok := buildBrowserCallbackRedirectURI(redirectURI, callbackHost, providerKey, tenantUUID, state, nonce)
	if !ok {
		return redirectURI
	}
	return rewritten
}

func (h *FederatedHandler) resolveProviderCallbackHost(c *gin.Context, providerKey, tenantUUID string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(providerKey)) {
	case "wecom":
		cfg, err := h.wecomConfig.GetByTenant(c.Request.Context(), tenantUUID)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(cfg.CallbackHost), nil
	case "dingtalk":
		cfg, err := h.dingtalkCfg.GetByTenant(c.Request.Context(), tenantUUID)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(cfg.CallbackHost), nil
	case "lark":
		cfg, err := h.larkCfg.GetByTenant(c.Request.Context(), tenantUUID)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(cfg.CallbackHost), nil
	default:
		return "", fmt.Errorf("provider not supported")
	}
}

func buildBrowserCallbackRedirectURI(rawRedirect, callbackHost, providerKey, tenantUUID, state, nonce string) (string, bool) {
	redirect := strings.TrimSpace(rawRedirect)
	host := strings.TrimSpace(callbackHost)
	if redirect == "" || host == "" {
		return redirect, false
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	base, err := url.Parse(host)
	if err != nil || strings.TrimSpace(base.Host) == "" {
		return redirect, false
	}
	target := &url.URL{
		Scheme: base.Scheme,
		Host:   base.Host,
		Path:   "/api/v1/auth/federated/browser/callback",
	}
	values := target.Query()
	values.Set("provider", strings.TrimSpace(providerKey))
	values.Set("tenant_uuid", strings.TrimSpace(tenantUUID))
	values.Set("state", strings.TrimSpace(state))
	values.Set("nonce", strings.TrimSpace(nonce))
	values.Set("front_redirect_uri", redirect)
	target.RawQuery = values.Encode()
	return target.String(), true
}

func resolveFrontRedirectURL(raw string) *url.URL {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return &url.URL{Scheme: "http", Host: "127.0.0.1:3131", Path: "/users/login"}
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return &url.URL{Scheme: "http", Host: "127.0.0.1:3131", Path: "/users/login"}
	}
	return parsed
}

func (h *FederatedHandler) redirectBrowserCallbackError(c *gin.Context, frontRedirectRaw, message string) {
	redirectURL := resolveFrontRedirectURL(frontRedirectRaw)
	q := redirectURL.Query()
	q.Set("error", message)
	q.Set("login_tab", "federated")
	redirectURL.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, redirectURL.String())
}

func resolveWeComCallbackURL(c *gin.Context, configuredHost string) string {
	path := c.Request.URL.Path
	host := strings.TrimSpace(configuredHost)
	if host != "" {
		host = strings.TrimRight(host, "/")
		if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
			return host + path
		}
		return "https://" + host + path
	}

	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = "https"
	}
	reqHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if reqHost == "" {
		reqHost = strings.TrimSpace(c.Request.Host)
	}
	if reqHost == "" {
		return "https://plugin.local" + path
	}
	return scheme + "://" + reqHost + path
}
