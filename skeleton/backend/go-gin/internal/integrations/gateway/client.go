package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	logger "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/google/uuid"
)

const (
	defaultRequestTimeout = 10 * time.Second
)

const (
	ErrCodeGatewayMissingBaseURL   = "GW_CFG_MISSING_BASE_URL"
	ErrCodeGatewayMissingSTSClient = "GW_CFG_MISSING_STS_CLIENT"
	ErrCodeGatewayMissingAPIKey    = "GW_CFG_MISSING_API_KEY"
	ErrCodeGatewayInvalidScheme    = "GW_CFG_INVALID_AUTH_SCHEME"
)

type GatewayConfigError struct {
	Code     string
	Message  string
	Required []string
	Present  []string
	IAMMode  string
}

func (e *GatewayConfigError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Code) == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// InvokeParams 描述一次能力调用的输入。
type InvokeParams struct {
	CapabilityID      string
	Action            string
	PreferredProtocol string
	Payload           any
	Headers           map[string]string
	RequestID         string
	TenantUUID        string
	AuthRequired      bool
	TenantScoped      bool
}

// InvokeResult 描述能力调用的输出。
type InvokeResult struct {
	TraceID    string
	Status     string
	Data       map[string]any
	Raw        json.RawMessage
	Mock       bool
	MockReason string
}

// ListPlatformCapabilitiesOptions controls remote catalog listing.
type ListPlatformCapabilitiesOptions struct {
	Source   string
	Channel  string
	PageSize int
}

// PlatformCapabilityRecord mirrors PowerX capability registry DTO.
type PlatformCapabilityRecord struct {
	CapabilityID     string                       `json:"capability_id"`
	PluginID         string                       `json:"plugin_id"`
	PluginVersion    string                       `json:"plugin_version"`
	Title            string                       `json:"title"`
	Description      string                       `json:"description"`
	Source           string                       `json:"source"`
	Categories       []string                     `json:"categories"`
	Intents          []string                     `json:"intents"`
	ToolScope        []string                     `json:"tool_scope"`
	CapabilitiesHash string                       `json:"capabilities_hash"`
	ProtocolHash     string                       `json:"protocol_hash"`
	Status           string                       `json:"status"`
	ExecutionMode    string                       `json:"execution_mode"`
	Protocols        []PlatformCapabilityProtocol `json:"protocols"`
}

// PlatformCapabilityProtocol describes channel metadata.
type PlatformCapabilityProtocol struct {
	Channel   string `json:"channel"`
	Endpoint  string `json:"endpoint"`
	SchemaRef string `json:"schema_ref"`
	Method    string `json:"method"`
	RPC       string `json:"rpc"`
	ToolRef   string `json:"tool_ref"`
}

type AgentRecord struct {
	ID          any            `json:"id,omitempty"`
	UUID        string         `json:"uuid,omitempty"`
	Key         string         `json:"key,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}

type PluginSkillSyncParams struct {
	PluginSkillID  string
	PowerXSkillID  string
	Version        string
	Title          string
	Description    string
	IntentExamples any
	InputSchema    any
	OutputSchema   any
	PromptSpec     any
	Executor       any
	Capability     string
	Checksum       string
	Provider       string
}

type PluginSkillSyncResult struct {
	PowerXSkillID string         `json:"powerx_skill_id"`
	Status        string         `json:"status"`
	TraceID       string         `json:"trace_id,omitempty"`
	Raw           map[string]any `json:"raw,omitempty"`
}

type PluginAgentSyncParams struct {
	PluginAgentID   string
	PowerXAgentUUID string
	AgentKey        string
	Name            string
	Description     string
	ModelProfileRef string
	Persona         string
	PromptSeed      string
	SkillIDs        []string
	Provider        string
	Meta            map[string]any
}

type PluginAgentSyncResult struct {
	PowerXAgentUUID string         `json:"powerx_agent_uuid"`
	PowerXAgentID   string         `json:"powerx_agent_id,omitempty"`
	Status          string         `json:"status"`
	TraceID         string         `json:"trace_id,omitempty"`
	Raw             map[string]any `json:"raw,omitempty"`
}

type AgentSessionParams struct {
	AgentID    string
	AgentUUID  string
	Title      string
	Env        string
	Meta       map[string]any
	TenantUUID string
}

type AgentSessionRecord struct {
	ID        any            `json:"id,omitempty"`
	UUID      string         `json:"uuid,omitempty"`
	AgentID   any            `json:"agentId,omitempty"`
	Title     string         `json:"title,omitempty"`
	Status    string         `json:"status,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
}

type AgentSessionMessageRecord struct {
	ID        any            `json:"id,omitempty"`
	UUID      string         `json:"uuid,omitempty"`
	SessionID any            `json:"sessionId,omitempty"`
	AgentID   any            `json:"agentId,omitempty"`
	Role      string         `json:"role,omitempty"`
	Content   string         `json:"content,omitempty"`
	Format    string         `json:"format,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	CreatedAt string         `json:"createdAt,omitempty"`
}

type AgentStreamParams struct {
	AgentID    string
	SessionID  string
	TraceID    string
	Query      string
	Intent     string
	Source     string
	Env        string
	TenantUUID string
}

type AgentSessionListOptions struct {
	AgentID    string
	SessionID  string
	Env        string
	Status     string
	Limit      int
	TenantUUID string
}

type AgentSessionMessageListOptions struct {
	SessionID  string
	Env        string
	AfterID    string
	Limit      int
	TenantUUID string
}

type AgentSessionMutationOptions struct {
	SessionID  string
	Env        string
	TenantUUID string
}

type AgentStream struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

type gatewayInvoker interface {
	Invoke(ctx context.Context, req frameworkgateway.InvokeRequest) (*frameworkgateway.Response, error)
	Close() error
}

// Client 封装 Skeleton 环境的 Gateway 调用能力，支持 PX_USE_MOCK 与离线提示。
type Client struct {
	transport     gatewayInvoker
	logger        *logger.Entry
	useMock       map[string]struct{}
	offlineReason string
	cfg           *config.Config
	refreshMu     sync.Mutex
	tokenProvider frameworkgateway.TokenProvider
	tenantMu      sync.Mutex
	tenantUUID    string
}

// NewClient 构造 Gateway Client；若凭证缺失，则进入离线模式。
func NewClient(cfg *config.Config, log *logger.Entry, opts ...ClientOption) *Client {
	c := &Client{
		logger:  ensureLogger(log),
		useMock: make(map[string]struct{}),
		cfg:     cfg,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	c.logger = logger.WithComponent("skeleton.gateway.client")

	if cfg == nil || cfg.Gateway == nil {
		c.offlineReason = "未找到 gateway 配置，请执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 或在 .env.local 写入 PX_GATEWAY_*"
		return c
	}

	for _, module := range cfg.Gateway.UseMock {
		if normalized := normalizeModule(module); normalized != "" {
			c.useMock[normalized] = struct{}{}
		}
	}

	gcfg := cfg.Gateway
	baseURL := effectiveGatewayBaseURL(gcfg)
	authScheme := effectiveGatewayAuthScheme(gcfg)
	credential := gatewayCredential(gcfg, authScheme)
	iamMode := gatewayIAMMode(cfg)
	hostDelegated := isHostDelegatedMode(cfg)

	if baseURL == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayMissingBaseURL, "PX_GATEWAY_BASE_URL is required", gcfg, iamMode, requiredGatewayEnv(cfg, gcfg, authScheme))
		c.offlineReason = cfgErr.Error()
		return c
	}
	if authScheme == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayInvalidScheme, "gateway auth_scheme must be bearer or apikey", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_GATEWAY_AUTH_SCHEME"})
		c.offlineReason = cfgErr.Error()
		return c
	}
	if authScheme == "bearer" && credential == "" {
		if hostDelegated {
			if cfgErr := validateHostDelegatedConfig(cfg, gcfg, iamMode); cfgErr != nil {
				c.offlineReason = cfgErr.Error()
				return c
			}
		} else {
			cfgErr := newGatewayConfigError(ErrCodeGatewayMissingSTSClient, "bearer gateway requires STS token provider", gcfg, iamMode, requiredGatewayEnv(cfg, gcfg, authScheme))
			c.offlineReason = cfgErr.Error()
			return c
		}
	}
	if authScheme == "apikey" && credential == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayMissingAPIKey, "PX_GATEWAY_API_KEY is required", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_GATEWAY_API_KEY", "PX_GATEWAY_AUTH_SCHEME=apikey"})
		c.offlineReason = cfgErr.Error()
		return c
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	if err := c.reconnectTransport(); err != nil {
		c.offlineReason = fmt.Sprintf("初始化 Gateway Client 失败: %v", err)
		logger.WarnCtx(logger.WithLogFields(context.Background(), map[string]interface{}{
			"component":  "skeleton.gateway.client",
			"biz_scene":  "gateway_init",
			"biz_domain": "integration",
			"error":      err.Error(),
		}), "无法初始化 Gateway Client，Skeleton 将保持离线状态")
		return c
	}

	logger.InfoCtx(logger.WithLogFields(context.Background(), map[string]interface{}{
		"component":   "skeleton.gateway.client",
		"mockModules": c.mockModules(),
		"baseURL":     baseURL,
		"biz_scene":   "gateway_init",
		"biz_domain":  "integration",
	}), "Gateway Client 初始化完成")
	return c
}

// ClientOption customizes the Gateway client.
type ClientOption func(*Client)

// WithTokenProvider injects an STS-backed bearer token provider.
func WithTokenProvider(provider frameworkgateway.TokenProvider) ClientOption {
	return func(c *Client) {
		if c != nil {
			c.tokenProvider = provider
		}
	}
}

// Enabled 返回 Gateway Client 是否可用。
func (c *Client) Enabled() bool {
	return c.transport != nil
}

// Invoke 触发能力调用；若命中 PX_USE_MOCK，则直接返回 Mock 响应。
func (c *Client) Invoke(ctx context.Context, params InvokeParams) (*InvokeResult, error) {
	module, mocked := c.shouldMock(params.CapabilityID)
	if mocked {
		logger.InfoWith(c.logger, ctx, "PX_USE_MOCK 生效，返回 Mock 数据", map[string]interface{}{
			"capability": params.CapabilityID,
			"action":     params.Action,
			"module":     module,
			"biz_scene":  "gateway_invoke",
			"biz_domain": "integration",
			"component":  "skeleton.gateway.client",
		})
		return c.mockResult(module, params, "PX_USE_MOCK"), nil
	}

	if c.transport == nil {
		return nil, c.unavailableError(params.CapabilityID)
	}

	requestAuthHeader := headerValue(params.Headers, "Authorization")
	tokenSource := resolveTokenSource(c.cfg, requestAuthHeader)
	tokenClaims := parseAuthIdentityClaims(requestAuthHeader)
	tokenTID := ""
	if params.AuthRequired && strings.TrimSpace(requestAuthHeader) == "" {
		return nil, &PolicyError{
			Code:    "GW_POLICY_AUTH_REQUIRED",
			Message: "auth_required=true 时必须提供请求态 Authorization（Bearer STS token）",
		}
	}
	if params.TenantScoped {
		tid, ok := tenantUUIDFromAuthHeader(requestAuthHeader)
		if !ok || strings.TrimSpace(tid) == "" {
			return nil, &PolicyError{
				Code:    "GW_POLICY_TENANT_TOKEN_REQUIRED",
				Message: "tenant_scoped=true 时 Authorization 必须为包含 tid claim 的 Bearer token",
			}
		}
		if isZeroTenantUUID(tid) {
			return nil, &PolicyError{
				Code:    "GW_POLICY_ZERO_TENANT_FORBIDDEN",
				Message: "tenant_scoped=true 时不允许使用零租户 token（tid=00000000-...）",
			}
		}
		if wanted := strings.TrimSpace(params.TenantUUID); wanted != "" && !strings.EqualFold(wanted, tid) {
			return nil, &PolicyError{
				Code:    "GW_POLICY_TENANT_MISMATCH",
				Message: fmt.Sprintf("tenant token tid(%s) 与请求 tenant_uuid(%s) 不一致", tid, wanted),
			}
		}
		params.TenantUUID = tid
		tokenTID = tid
	}
	if tokenTID == "" {
		tokenTID = tokenClaims.TenantUUID
	}

	req := frameworkgateway.InvokeRequest{
		CapabilityID:      params.CapabilityID,
		Action:            params.Action,
		PreferredProtocol: params.PreferredProtocol,
		Payload:           params.Payload,
		RequestID:         params.RequestID,
		Headers:           copyHeaders(params.Headers),
		TenantUUID:        params.TenantUUID,
		DisableAuth:       !params.AuthRequired,
	}
	if c.logger != nil {
		baseURL := ""
		apiPrefix := ""
		authScheme := ""
		effectiveBase := ""
		if c.cfg != nil && c.cfg.Gateway != nil {
			baseURL = strings.TrimSpace(c.cfg.Gateway.BaseURL)
			apiPrefix = strings.TrimSpace(c.cfg.Gateway.APIPrefix)
			authScheme = effectiveGatewayAuthScheme(c.cfg.Gateway)
			effectiveBase = effectiveGatewayBaseURL(c.cfg.Gateway)
		}
		permissionAudit := strings.TrimSpace(params.CapabilityID + ":" + params.Action)
		tenantAudit := strings.TrimSpace(params.TenantUUID)
		if tenantAudit == "" {
			tenantAudit = tokenTID
		}
		logger.InfoCtx(logger.WithLogFields(ctx, map[string]interface{}{
			"component":              "skeleton.gateway.client",
			"capability":             params.CapabilityID,
			"action":                 params.Action,
			"preferred_protocol":     params.PreferredProtocol,
			"request_id":             params.RequestID,
			"auth_required":          params.AuthRequired,
			"tenant_scoped":          params.TenantScoped,
			"token_source":           tokenSource,
			"token_tid":              maskTenantUUID(tokenTID),
			"payload_method":         strings.TrimSpace(strings.ToUpper(fmt.Sprint(extractMapValue(params.Payload, "method")))),
			"payload_endpoint":       strings.TrimSpace(fmt.Sprint(extractMapValue(params.Payload, "endpoint"))),
			"gateway_base_url":       baseURL,
			"gateway_api_prefix":     apiPrefix,
			"gateway_effective_base": effectiveBase,
			"gateway_auth_scheme":    authScheme,
			"mode":                   gatewayIAMMode(c.cfg),
			"tenant_uuid":            maskTenantUUID(tenantAudit),
			"user_id":                tokenClaims.UserID,
			"permission":             permissionAudit,
			"trace_id":               strings.TrimSpace(params.RequestID),
			"token_roles":            tokenClaims.Roles,
			"token_permissions":      tokenClaims.Permissions,
			"biz_scene":              "gateway_invoke",
			"biz_domain":             "integration",
		}), "gateway invoke dispatch")
	}
	resp, err := c.transport.Invoke(ctx, req)
	if err != nil {
		if retryResp, retryErr := c.handleInvokeError(ctx, req, params.CapabilityID, params.Action, err); retryErr == nil && retryResp != nil {
			resp = retryResp
		} else if retryErr != nil {
			return nil, fmt.Errorf("gateway invoke %s/%s 失败: %w", params.CapabilityID, params.Action, retryErr)
		} else {
			return nil, fmt.Errorf("gateway invoke %s/%s 失败: %w", params.CapabilityID, params.Action, err)
		}
	}
	return &InvokeResult{
		TraceID: resp.TraceID,
		Status:  resp.Status,
		Data:    resp.Data,
		Raw:     resp.RawData,
	}, nil
}

func extractMapValue(payload any, key string) any {
	valueMap, ok := payload.(map[string]any)
	if !ok || valueMap == nil {
		return nil
	}
	return valueMap[key]
}

// ListPlatformCapabilities retrieves platform capability metadata via tenant API.
func (c *Client) ListPlatformCapabilities(ctx context.Context, opts ListPlatformCapabilitiesOptions) ([]PlatformCapabilityRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/tenant/capabilities")
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenant(gcfg)

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	query := url.Values{}
	if source := strings.TrimSpace(opts.Source); source != "" {
		query.Set("source", source)
	} else {
		query.Set("source", "corex")
	}
	if channel := strings.TrimSpace(opts.Channel); channel != "" {
		query.Set("channel", channel)
	}
	pageSize := opts.PageSize
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 200
	}
	query.Set("page_size", strconv.Itoa(pageSize))

	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload platformCapabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode capability list: %w", err)
	}
	if resp.StatusCode >= 400 || payload.Code >= 400 {
		return nil, fmt.Errorf("platform capability list failed: status=%d code=%d message=%s", resp.StatusCode, payload.Code, payload.Message)
	}
	return payload.Data.Items, nil
}

func (c *Client) ResolveGatewayTenantUUID(ctx context.Context) (string, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return "", fmt.Errorf("gateway config missing")
	}
	c.tenantMu.Lock()
	cached := strings.TrimSpace(c.tenantUUID)
	c.tenantMu.Unlock()
	if cached != "" {
		return cached, nil
	}

	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/admin/user/auth/me/context")
	if endpoint == "" {
		return "", fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	req.Header.Set("X-Request-ID", uuid.NewString())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			CurrentTenantUUID string `json:"current_tenant_uuid"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode gateway tenant context: %w", err)
	}
	if resp.StatusCode >= 400 || payload.Code >= 400 {
		return "", fmt.Errorf("gateway tenant context failed: status=%d code=%d message=%s", resp.StatusCode, payload.Code, payload.Message)
	}
	tenantUUID := strings.TrimSpace(payload.Data.CurrentTenantUUID)
	if tenantUUID == "" {
		return "", fmt.Errorf("gateway tenant context missing current_tenant_uuid")
	}
	c.tenantMu.Lock()
	c.tenantUUID = tenantUUID
	c.tenantMu.Unlock()
	return tenantUUID, nil
}

func (c *Client) ListAgents(ctx context.Context, env string) ([]AgentRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/admin/agents")
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenant(gcfg)

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	query := url.Values{}
	if strings.TrimSpace(env) != "" {
		query.Set("env", strings.TrimSpace(env))
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload platformAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode agent list: %w", err)
	}
	if resp.StatusCode >= 400 || payload.Code >= 400 {
		return nil, fmt.Errorf("platform agent list failed: status=%d code=%d message=%s", resp.StatusCode, payload.Code, payload.Message)
	}
	return payload.Data.Items, nil
}

func (c *Client) GetAgent(ctx context.Context, agentUUID string) (*AgentRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	agentUUID = strings.TrimSpace(agentUUID)
	if agentUUID == "" {
		return nil, fmt.Errorf("agent uuid is required")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/admin/agents/"+url.PathEscape(agentUUID))
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenant(gcfg)
	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload struct {
		Success bool        `json:"success"`
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    AgentRecord `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode agent detail: %w", err)
	}
	if resp.StatusCode >= 400 || payload.Code >= 400 {
		return nil, fmt.Errorf("platform agent detail failed: status=%d code=%d message=%s", resp.StatusCode, payload.Code, payload.Message)
	}
	return &payload.Data, nil
}

func (c *Client) SyncPluginSkill(ctx context.Context, params PluginSkillSyncParams) (*PluginSkillSyncResult, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	if strings.TrimSpace(params.PluginSkillID) == "" {
		return nil, fmt.Errorf("plugin skill id is required")
	}
	if strings.TrimSpace(params.Title) == "" {
		return nil, fmt.Errorf("skill title is required")
	}
	if strings.TrimSpace(params.Capability) == "" {
		return nil, fmt.Errorf("skill capability is required")
	}
	body := map[string]any{
		"skill_id":        firstNonEmptyString(params.PowerXSkillID, params.PluginSkillID),
		"plugin_skill_id": params.PluginSkillID,
		"provider":        strings.TrimSpace(params.Provider),
		"version":         firstNonEmptyString(params.Version, "1.0.0"),
		"title":           strings.TrimSpace(params.Title),
		"description":     strings.TrimSpace(params.Description),
		"intent_examples": params.IntentExamples,
		"input_schema":    params.InputSchema,
		"output_schema":   params.OutputSchema,
		"prompt_spec":     params.PromptSpec,
		"executor":        params.Executor,
		"capability":      strings.TrimSpace(params.Capability),
		"checksum":        normalizeSHA256Checksum(params.Checksum),
		"source":          "plugin",
		"sync_source":     "plugin_registry",
	}
	endpoint := "/admin/skills/plugin-registry/sync"
	if strings.TrimSpace(params.PowerXSkillID) != "" {
		endpoint = "/admin/skills/plugin-registry/" + url.PathEscape(strings.TrimSpace(params.PowerXSkillID)) + "/sync"
	}
	raw, traceID, err := c.doPlatformJSON(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	return &PluginSkillSyncResult{
		PowerXSkillID: firstNonEmptyString(stringFromMap(raw, "powerx_skill_id"), stringFromMap(raw, "skill_id"), strings.TrimSpace(params.PowerXSkillID), strings.TrimSpace(params.PluginSkillID)),
		Status:        firstNonEmptyString(stringFromMap(raw, "status"), "synced"),
		TraceID:       traceID,
		Raw:           raw,
	}, nil
}

func (c *Client) SyncPluginAgent(ctx context.Context, params PluginAgentSyncParams) (*PluginAgentSyncResult, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	if strings.TrimSpace(params.PluginAgentID) == "" {
		return nil, fmt.Errorf("plugin agent id is required")
	}
	if strings.TrimSpace(params.Name) == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	body := map[string]any{
		"env":               "dev",
		"key":               strings.TrimSpace(params.AgentKey),
		"name":              strings.TrimSpace(params.Name),
		"description":       strings.TrimSpace(params.Description),
		"model_profile_ref": strings.TrimSpace(params.ModelProfileRef),
		"persona":           strings.TrimSpace(params.Persona),
		"promptSeed":        strings.TrimSpace(params.PromptSeed),
		"skillIds":          params.SkillIDs,
		"status":            "active",
		"visibility":        "tenant",
		"scope":             "tenant",
		"ownerPluginId":     strings.TrimSpace(params.Provider),
		"managedByPlugin":   true,
		"source":            "plugin:" + strings.TrimSpace(params.Provider),
		"meta": mergeStringAnyMaps(params.Meta, map[string]any{
			"source":          "plugin_registry",
			"provider":        strings.TrimSpace(params.Provider),
			"plugin_agent_id": strings.TrimSpace(params.PluginAgentID),
			"skillIds":        params.SkillIDs,
		}),
	}
	powerxAgentUUID := strings.TrimSpace(params.PowerXAgentUUID)
	if powerxAgentUUID == "" && strings.TrimSpace(params.AgentKey) != "" {
		if existing, err := c.findAgentByKey(ctx, params.AgentKey, "dev"); err == nil && existing != nil {
			powerxAgentUUID = strings.TrimSpace(existing.UUID)
		}
	}

	method := http.MethodPost
	endpoint := "/admin/agents"
	if powerxAgentUUID != "" {
		method = http.MethodPatch
		endpoint = "/admin/agents/" + url.PathEscape(powerxAgentUUID) + "?env=dev"
	}
	raw, traceID, err := c.doPlatformJSON(ctx, method, endpoint, body)
	if err != nil {
		if method == http.MethodPost && strings.Contains(strings.ToLower(err.Error()), "duplicate key") && strings.TrimSpace(params.AgentKey) != "" {
			if existing, findErr := c.findAgentByKey(ctx, params.AgentKey, "dev"); findErr == nil && existing != nil && strings.TrimSpace(existing.UUID) != "" {
				return &PluginAgentSyncResult{
					PowerXAgentUUID: strings.TrimSpace(existing.UUID),
					PowerXAgentID:   stringFromAny(existing.ID),
					Status:          firstNonEmptyString(existing.Status, "active"),
					TraceID:         traceID,
					Raw: map[string]any{
						"uuid":   strings.TrimSpace(existing.UUID),
						"id":     existing.ID,
						"key":    strings.TrimSpace(existing.Key),
						"name":   strings.TrimSpace(existing.Name),
						"status": firstNonEmptyString(existing.Status, "active"),
					},
				}, nil
			}
		}
		return nil, err
	}
	return &PluginAgentSyncResult{
		PowerXAgentUUID: firstNonEmptyString(stringFromMap(raw, "powerx_agent_uuid"), stringFromMap(raw, "uuid"), powerxAgentUUID),
		PowerXAgentID:   firstNonEmptyString(stringFromMap(raw, "id"), stringFromMap(raw, "agent_id")),
		Status:          firstNonEmptyString(stringFromMap(raw, "status"), "active"),
		TraceID:         traceID,
		Raw:             raw,
	}, nil
}

func (c *Client) RegisterCatalog(ctx context.Context, catalog *capabilities.CatalogSnapshot, assets []capabilities.ProtocolAsset) error {
	if catalog == nil {
		return fmt.Errorf("capability catalog is required")
	}
	if strings.TrimSpace(catalog.PluginID) == "" {
		return fmt.Errorf("capability catalog plugin_id is required")
	}
	if len(catalog.Entries) == 0 {
		return fmt.Errorf("capability catalog entries are required")
	}
	payloadAssets := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		path := strings.TrimSpace(asset.Path)
		if path == "" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read capability asset %s: %w", path, err)
		}
		payloadAssets = append(payloadAssets, map[string]any{
			"type":    strings.TrimSpace(asset.Type),
			"path":    filepath.ToSlash(path),
			"size":    len(content),
			"content": base64.StdEncoding.EncodeToString(content),
		})
	}
	_, _, err := c.doPlatformJSON(ctx, http.MethodPost, "/internal/plugins/capabilities/catalog", map[string]any{
		"catalog": catalog,
		"assets":  payloadAssets,
	})
	if err != nil {
		return fmt.Errorf("sync capability catalog: %w", err)
	}
	return nil
}

func (c *Client) findAgentByKey(ctx context.Context, agentKey string, env string) (*AgentRecord, error) {
	agentKey = strings.TrimSpace(agentKey)
	if agentKey == "" {
		return nil, fmt.Errorf("agent key is required")
	}
	items, err := c.ListAgents(ctx, env)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		if strings.EqualFold(strings.TrimSpace(items[idx].Key), agentKey) {
			return &items[idx], nil
		}
	}
	return nil, nil
}

func (c *Client) CreateAgentSession(ctx context.Context, params AgentSessionParams) (*AgentSessionRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	agentUUID := strings.TrimSpace(params.AgentUUID)
	agentID := strings.TrimSpace(params.AgentID)
	if agentUUID == "" && agentID == "" {
		return nil, fmt.Errorf("agent_uuid or agent_id is required")
	}

	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/agents/sessions")
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenantForRequest(gcfg, params.TenantUUID)

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	body := map[string]any{
		"title": strings.TrimSpace(params.Title),
		"meta":  params.Meta,
	}
	if env := strings.TrimSpace(params.Env); env != "" {
		body["env"] = env
	}
	if agentUUID != "" {
		body["agentUuid"] = agentUUID
	} else {
		body["agentId"] = agentID
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctxReq, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response platformAgentSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode agent session: %w", err)
	}
	if resp.StatusCode >= 400 || response.Code >= 400 {
		return nil, fmt.Errorf("platform agent session create failed: status=%d code=%d message=%s", resp.StatusCode, response.Code, response.Message)
	}
	session := response.Data
	if strings.TrimSpace(session.SessionID) == "" {
		session.SessionID = firstSessionIdentifier(session)
	}
	if strings.TrimSpace(session.SessionID) == "" {
		return nil, fmt.Errorf("platform agent session response missing id")
	}
	return &session, nil
}

func (c *Client) ListAgentSessions(ctx context.Context, opts AgentSessionListOptions) ([]AgentSessionRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	agentID := strings.TrimSpace(opts.AgentID)
	if agentID == "" {
		return nil, fmt.Errorf("agent_id or agent_uuid is required")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/agents/sessions")
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	query := url.Values{}
	if env := strings.TrimSpace(opts.Env); env != "" {
		query.Set("env", env)
	}
	if looksLikeUUID(agentID) {
		query.Set("agent_uuid", agentID)
	} else {
		query.Set("agent_id", agentID)
	}
	if status := strings.TrimSpace(opts.Status); status != "" {
		query.Set("status", status)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query.Set("limit", strconv.Itoa(limit))
	endpoint += "?" + query.Encode()

	var response platformAgentSessionListResponse
	if err := c.doGatewayJSONWithTenant(ctx, http.MethodGet, endpoint, nil, &response, opts.TenantUUID); err != nil {
		return nil, err
	}
	if response.Code >= 400 {
		return nil, fmt.Errorf("platform agent sessions list failed: code=%d message=%s", response.Code, response.Message)
	}
	out := response.Data.Items
	for i := range out {
		if strings.TrimSpace(out[i].SessionID) == "" {
			out[i].SessionID = firstSessionIdentifier(out[i])
		}
	}
	return out, nil
}

func (c *Client) ResolveAgentSessionID(ctx context.Context, opts AgentSessionListOptions) (string, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" || !looksLikeUUID(sessionID) {
		return sessionID, nil
	}
	if c.cfg == nil || c.cfg.Gateway == nil {
		return sessionID, fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/agents/sessions")
	if endpoint == "" {
		return sessionID, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenantForRequest(gcfg, opts.TenantUUID)
	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	query := url.Values{}
	if env := strings.TrimSpace(opts.Env); env != "" {
		query.Set("env", env)
	}
	agentID := strings.TrimSpace(opts.AgentID)
	if looksLikeUUID(agentID) {
		query.Set("agent_uuid", agentID)
	} else if agentID != "" {
		query.Set("agent_id", agentID)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query.Set("limit", strconv.Itoa(limit))
	endpoint += "?" + query.Encode()

	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return sessionID, err
	}
	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, endpoint, nil)
	if err != nil {
		return sessionID, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return sessionID, err
	}
	defer resp.Body.Close()
	var response platformAgentSessionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return sessionID, fmt.Errorf("decode agent sessions: %w", err)
	}
	if resp.StatusCode >= 400 || response.Code >= 400 {
		return sessionID, fmt.Errorf("platform agent sessions list failed: status=%d code=%d message=%s", resp.StatusCode, response.Code, response.Message)
	}
	for _, item := range response.Data.Items {
		if strings.EqualFold(strings.TrimSpace(item.UUID), sessionID) {
			if id := strings.TrimSpace(firstSessionIdentifier(AgentSessionRecord{ID: item.ID})); id != "" {
				return id, nil
			}
			return sessionID, nil
		}
	}
	return sessionID, nil
}

func (c *Client) ListAgentSessionMessages(ctx context.Context, opts AgentSessionMessageListOptions) ([]AgentSessionMessageRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/agents/sessions/"+url.PathEscape(sessionID)+"/messages")
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	query := url.Values{}
	if env := strings.TrimSpace(opts.Env); env != "" {
		query.Set("env", env)
	}
	if afterID := strings.TrimSpace(opts.AfterID); afterID != "" {
		query.Set("after_id", afterID)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 200
	}
	query.Set("limit", strconv.Itoa(limit))
	endpoint += "?" + query.Encode()

	var response platformAgentSessionMessageListResponse
	if err := c.doGatewayJSONWithTenant(ctx, http.MethodGet, endpoint, nil, &response, opts.TenantUUID); err != nil {
		return nil, err
	}
	if response.Code >= 400 {
		return nil, fmt.Errorf("platform agent session messages failed: code=%d message=%s", response.Code, response.Message)
	}
	return response.Data.Items, nil
}

func (c *Client) DeleteAgentSession(ctx context.Context, opts AgentSessionMutationOptions) error {
	return c.mutateAgentSession(ctx, http.MethodDelete, opts, "")
}

func (c *Client) ArchiveAgentSession(ctx context.Context, opts AgentSessionMutationOptions) error {
	return c.mutateAgentSession(ctx, http.MethodPost, opts, "/archive")
}

func (c *Client) mutateAgentSession(ctx context.Context, method string, opts AgentSessionMutationOptions, suffix string) error {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return fmt.Errorf("gateway config missing")
	}
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/agents/sessions/"+url.PathEscape(sessionID)+suffix)
	if endpoint == "" {
		return fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	if env := strings.TrimSpace(opts.Env); env != "" {
		endpoint += "?env=" + url.QueryEscape(env)
	}
	var response platformSimpleResponse
	if err := c.doGatewayJSONWithTenant(ctx, method, endpoint, nil, &response, opts.TenantUUID); err != nil {
		return err
	}
	if response.Code >= 400 {
		return fmt.Errorf("platform agent session mutation failed: code=%d message=%s", response.Code, response.Message)
	}
	return nil
}

func (c *Client) StreamAgentSSE(ctx context.Context, params AgentStreamParams) (*AgentStream, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	if strings.TrimSpace(params.AgentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if strings.TrimSpace(params.TraceID) == "" {
		return nil, fmt.Errorf("trace_id is required")
	}
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("q is required")
	}

	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, "/agents/stream/sse")
	if endpoint == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenantForRequest(gcfg, params.TenantUUID)

	query := url.Values{}
	agentID := strings.TrimSpace(params.AgentID)
	sessionID := strings.TrimSpace(params.SessionID)
	resolvedSessionID, err := c.ResolveAgentSessionID(ctx, AgentSessionListOptions{
		AgentID:    agentID,
		SessionID:  sessionID,
		Env:        params.Env,
		TenantUUID: params.TenantUUID,
	})
	if err != nil {
		return nil, err
	}
	sessionUUID := sessionID
	sessionID = strings.TrimSpace(resolvedSessionID)
	if looksLikeUUID(agentID) {
		query.Set("agent_uuid", agentID)
	} else {
		query.Set("agent_id", agentID)
	}
	if looksLikeUUID(sessionUUID) {
		query.Set("session_uuid", sessionUUID)
	}
	query.Set("session_id", sessionID)
	query.Set("trace_id", strings.TrimSpace(params.TraceID))
	query.Set("q", strings.TrimSpace(params.Query))
	if intent := strings.TrimSpace(params.Intent); intent != "" {
		query.Set("intent", intent)
	}
	if source := strings.TrimSpace(params.Source); source != "" {
		query.Set("source", source)
	}
	if env := strings.TrimSpace(params.Env); env != "" {
		query.Set("env", env)
	}
	endpoint += "?" + query.Encode()

	credential, err := c.gatewayCredential(ctx, authScheme)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", strings.TrimSpace(params.TraceID))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("powerx agent stream failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return &AgentStream{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       resp.Body,
	}, nil
}

func (c *Client) doPlatformJSON(ctx context.Context, method, path string, body map[string]any) (map[string]any, string, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, "", fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	endpoint := gatewayEndpoint(gcfg, path)
	if endpoint == "" {
		return nil, "", fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenant(gcfg)
	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return nil, "", err
	}
	requestID := uuid.NewString()
	req, err := http.NewRequestWithContext(ctxReq, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", requestID)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	rawBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, requestID, fmt.Errorf("read platform response: %w", readErr)
	}
	var envelope struct {
		Success   bool            `json:"success"`
		Code      int             `json:"code"`
		Message   string          `json:"message"`
		Data      map[string]any  `json:"data"`
		Error     json.RawMessage `json:"error"`
		RequestID string          `json:"request_id"`
	}
	if len(strings.TrimSpace(string(rawBody))) == 0 {
		return nil, requestID, fmt.Errorf("platform request failed: status=%d empty response body", resp.StatusCode)
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		bodyText := strings.TrimSpace(string(rawBody))
		if len(bodyText) > 500 {
			bodyText = bodyText[:500] + "..."
		}
		return nil, requestID, fmt.Errorf("decode platform response: status=%d body=%s: %w", resp.StatusCode, bodyText, err)
	}
	errorCode, errorMessage := parsePlatformError(envelope.Error)
	if resp.StatusCode >= 400 || envelope.Code >= 400 || (!envelope.Success && (errorCode != "" || errorMessage != "")) {
		msg := firstNonEmptyString(errorMessage, envelope.Message)
		if msg == "" {
			msg = fmt.Sprintf("platform request failed: status=%d code=%d", resp.StatusCode, envelope.Code)
		}
		if errorCode != "" {
			msg = errorCode + ": " + msg
		}
		return nil, firstNonEmptyString(envelope.RequestID, requestID), fmt.Errorf("%s", msg)
	}
	return envelope.Data, firstNonEmptyString(envelope.RequestID, requestID), nil
}

func (c *Client) doGatewayJSON(ctx context.Context, method, endpoint string, body any, out any) error {
	return c.doGatewayJSONWithTenant(ctx, method, endpoint, body, out, "")
}

func (c *Client) doGatewayJSONWithTenant(ctx context.Context, method, endpoint string, body any, out any, tenantOverride string) error {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	tenant := effectiveGatewayTenantForRequest(gcfg, tenantOverride)
	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	credential, err := c.gatewayCredential(ctxReq, authScheme)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctxReq, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", buildGatewayAuthHeader(authScheme, credential))
	if tenant != "" {
		req.Header.Set("tenant_uuid", tenant)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode platform response: %w", err)
	}
	if resp.StatusCode >= 400 {
		message := ""
		switch payload := out.(type) {
		case *platformSimpleResponse:
			message = strings.TrimSpace(payload.Message)
		case interface{ GetMessage() string }:
			message = strings.TrimSpace(payload.GetMessage())
		}
		if message != "" {
			return fmt.Errorf("platform request failed: status=%d message=%s", resp.StatusCode, message)
		}
		return fmt.Errorf("platform request failed: status=%d", resp.StatusCode)
	}
	return nil
}

func parsePlatformError(raw json.RawMessage) (string, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return "", strings.TrimSpace(text)
	}
	var object struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		return strings.TrimSpace(object.Code), strings.TrimSpace(object.Message)
	}
	return "", trimmed
}

func normalizeSHA256Checksum(checksum string) string {
	value := strings.TrimSpace(checksum)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "sha256:") || strings.HasPrefix(lower, "sha256-") {
		return value
	}
	if len(value) == 64 {
		for _, ch := range value {
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
				return value
			}
		}
		return "sha256:" + value
	}
	return value
}

func normalizeGatewayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func looksLikeUUID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}
	for i, ch := range value {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

func gatewayEndpoint(gcfg *config.GatewayConfig, path string) string {
	baseURL := strings.TrimRight(effectiveGatewayBaseURL(gcfg), "/")
	if baseURL == "" {
		return ""
	}
	return baseURL + normalizeGatewayPath(path)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func mergeStringAnyMaps(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		if strings.TrimSpace(key) != "" {
			out[key] = value
		}
	}
	return out
}

func (c *Client) handleInvokeError(ctx context.Context, req frameworkgateway.InvokeRequest, capabilityID, action string, invokeErr error) (*frameworkgateway.Response, error) {
	var gwErr *frameworkgateway.InvocationError
	if !errors.As(invokeErr, &gwErr) {
		return nil, invokeErr
	}
	if gwErr.StatusCode != http.StatusUnauthorized && gwErr.StatusCode != http.StatusForbidden {
		return nil, invokeErr
	}
	refreshed, err := c.refreshCredentials(ctx)
	if err != nil {
		logger.WarnCtx(logger.WithLogFields(ctx, map[string]interface{}{
			"component":  "skeleton.gateway.client",
			"biz_scene":  "gateway_token_refresh",
			"biz_domain": "integration",
			"error":      err.Error(),
		}), "Gateway bearer token refresh failed")
		return nil, invokeErr
	}
	if !refreshed {
		return nil, invokeErr
	}
	resp, err := c.transport.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gateway invoke %s/%s 失败: %w", capabilityID, action, err)
	}
	return resp, nil
}

// Close 释放底层连接。
func (c *Client) Close() error {
	if c.transport == nil {
		return nil
	}
	return c.transport.Close()
}

// UnavailableError 用于提示 Gateway 离线的原因。
type UnavailableError struct {
	Reason string
}

func (e *UnavailableError) Error() string {
	if e == nil {
		return ""
	}
	return "gateway 不可用: " + e.Reason
}

// PolicyError 表示调用策略不满足（例如缺少请求态鉴权信息）。
type PolicyError struct {
	Code    string
	Message string
}

func (e *PolicyError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Code) == "" {
		return strings.TrimSpace(e.Message)
	}
	return strings.TrimSpace(e.Code) + ": " + strings.TrimSpace(e.Message)
}

func (c *Client) unavailableError(capabilityID string) error {
	reason := c.offlineReason
	if reason == "" {
		reason = "Gateway 未初始化"
	}
	if capabilityID != "" {
		reason += " (capability: " + capabilityID + ")"
	}
	return &UnavailableError{Reason: reason}
}

func (c *Client) mockResult(module string, params InvokeParams, reason string) *InvokeResult {
	traceID := fmt.Sprintf("mock-%s-%d", module, time.Now().UnixNano())
	payloadEcho := params.Payload
	data := map[string]any{
		"mock":       true,
		"module":     module,
		"action":     params.Action,
		"capability": params.CapabilityID,
		"message":    fmt.Sprintf("Mock 模式生效：%s", reason),
	}
	if payloadEcho != nil {
		data["echoPayload"] = payloadEcho
	}
	return &InvokeResult{
		TraceID:    traceID,
		Status:     "mock",
		Data:       data,
		Mock:       true,
		MockReason: reason,
	}
}

func (c *Client) shouldMock(capabilityID string) (string, bool) {
	if len(c.useMock) == 0 {
		return "", false
	}
	module := moduleFromCapability(capabilityID)
	if module == "" {
		return "", false
	}
	_, ok := c.useMock[module]
	return module, ok
}

func (c *Client) mockModules() []string {
	if len(c.useMock) == 0 {
		return nil
	}
	result := make([]string, 0, len(c.useMock))
	for module := range c.useMock {
		result = append(result, module)
	}
	return result
}

func moduleFromCapability(capabilityID string) string {
	parts := strings.Split(strings.TrimSpace(capabilityID), ".")
	if len(parts) >= 3 {
		return normalizeModule(parts[2])
	}
	if len(parts) >= 2 {
		return normalizeModule(parts[1])
	}
	if len(parts) == 1 {
		return normalizeModule(parts[0])
	}
	return ""
}

func normalizeModule(module string) string {
	return strings.ToLower(strings.TrimSpace(module))
}

func copyHeaders(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dest := make(map[string]string, len(src))
	for k, v := range src {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		dest[k] = v
	}
	return dest
}

func headerValue(headers map[string]string, key string) string {
	if len(headers) == 0 || strings.TrimSpace(key) == "" {
		return ""
	}
	for k, v := range headers {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type authIdentityClaims struct {
	TenantUUID  string
	UserID      string
	Roles       []string
	Permissions []string
}

func resolveTokenSource(cfg *config.Config, requestAuthHeader string) string {
	if strings.TrimSpace(requestAuthHeader) != "" {
		return "request"
	}
	if cfg == nil || cfg.Gateway == nil {
		return "none"
	}
	authScheme := effectiveGatewayAuthScheme(cfg.Gateway)
	if authScheme == "apikey" && strings.TrimSpace(cfg.Gateway.APIKey) != "" {
		return "gateway_apikey"
	}
	if isHostDelegatedMode(cfg) {
		return "gateway_sts"
	}
	return "none"
}

func parseAuthIdentityClaims(header string) authIdentityClaims {
	claims := authIdentityClaims{}
	token, ok := bearerTokenFromHeader(header)
	if !ok {
		return claims
	}
	decoded := decodeJWTClaims(token)
	if decoded == nil {
		return claims
	}
	claims.TenantUUID = strings.TrimSpace(firstClaimString(decoded, "tid", "tenant_uuid"))
	claims.UserID = strings.TrimSpace(firstClaimString(decoded, "sub", "uid", "user_id"))
	claims.Roles = parseSliceClaim(decoded["roles"])
	claims.Permissions = parseSliceClaim(decoded["permissions"])
	return claims
}

func bearerTokenFromHeader(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return "", false
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	if token == "" {
		return "", false
	}
	return token, true
}

func decodeJWTClaims(token string) map[string]any {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}

func firstClaimString(claims map[string]any, keys ...string) string {
	if claims == nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := claims[key]
		if !ok || raw == nil {
			continue
		}
		if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func parseSliceClaim(raw any) []string {
	result := make([]string, 0)
	seen := map[string]struct{}{}
	push := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	switch value := raw.(type) {
	case []string:
		for _, item := range value {
			push(item)
		}
	case []any:
		for _, item := range value {
			push(fmt.Sprint(item))
		}
	case string:
		if strings.Contains(value, ",") {
			for _, item := range strings.Split(value, ",") {
				push(item)
			}
		} else {
			push(value)
		}
	}
	return result
}

func tenantUUIDFromAuthHeader(header string) (string, bool) {
	token, ok := bearerTokenFromHeader(header)
	if !ok {
		return "", false
	}
	tid := strings.TrimSpace(tenantUUIDFromJWT(token))
	if tid == "" {
		return "", false
	}
	return tid, true
}

func isZeroTenantUUID(tid string) bool {
	normalized := strings.ToLower(strings.TrimSpace(tid))
	return normalized == "00000000-0000-0000-0000-000000000000"
}

func maskTenantUUID(tid string) string {
	tid = strings.TrimSpace(tid)
	if tid == "" {
		return ""
	}
	if len(tid) <= 8 {
		return tid
	}
	return tid[:8] + "***"
}

func ensureLogger(entry *logger.Entry) *logger.Entry {
	if entry != nil {
		return entry
	}
	return logger.WithComponent("skeleton.gateway.client")
}

// ValidateDelegatedConfig validates delegated gateway contract v1.
func ValidateDelegatedConfig(cfg *config.Config) *GatewayConfigError {
	iamMode := gatewayIAMMode(cfg)
	var gcfg *config.GatewayConfig
	if cfg != nil {
		gcfg = cfg.Gateway
	}
	if isHostDelegatedMode(cfg) {
		return validateHostDelegatedConfig(cfg, gcfg, iamMode)
	}
	required := requiredGatewayEnv(cfg, gcfg, effectiveGatewayAuthScheme(gcfg))

	if effectiveGatewayBaseURL(gcfg) == "" {
		return newGatewayConfigError(ErrCodeGatewayMissingBaseURL, "PX_GATEWAY_BASE_URL is required", gcfg, iamMode, required)
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	if authScheme == "" {
		return newGatewayConfigError(ErrCodeGatewayInvalidScheme, "gateway auth_scheme must be bearer or apikey", gcfg, iamMode, required)
	}
	if gatewayCredential(gcfg, authScheme) == "" {
		switch authScheme {
		case "apikey":
			return newGatewayConfigError(ErrCodeGatewayMissingAPIKey, "PX_GATEWAY_API_KEY is required", gcfg, iamMode, required)
		default:
			return newGatewayConfigError(ErrCodeGatewayMissingSTSClient, "bearer gateway requires STS token provider", gcfg, iamMode, required)
		}
	}
	return nil
}

func validateHostDelegatedConfig(cfg *config.Config, gcfg *config.GatewayConfig, iamMode string) *GatewayConfigError {
	required := []string{
		"PX_GATEWAY_BASE_URL",
		"POWERX_STS_CLIENT_ID",
		"POWERX_GRPC_UPSTREAM_ADDRESS",
		"POWERX_GRPC_UPSTREAM_TENANT_UUID",
		"PX_GATEWAY_AUTH_SCHEME=bearer",
	}
	if effectiveGatewayBaseURL(gcfg) == "" {
		return newGatewayConfigError(ErrCodeGatewayMissingBaseURL, "PX_GATEWAY_BASE_URL is required", gcfg, iamMode, required)
	}
	if effectiveGatewayAuthScheme(gcfg) != "bearer" {
		return newGatewayConfigError(ErrCodeGatewayInvalidScheme, "gateway auth_scheme must be bearer", gcfg, iamMode, required)
	}
	if cfg == nil || cfg.GRPCUpstream == nil ||
		strings.TrimSpace(cfg.GRPCUpstream.STSClientID) == "" ||
		strings.TrimSpace(cfg.GRPCUpstream.STSClientSecret) == "" ||
		strings.TrimSpace(cfg.GRPCUpstream.Address) == "" ||
		strings.TrimSpace(cfg.GRPCUpstream.TenantUUID) == "" {
		return newGatewayConfigError(ErrCodeGatewayMissingSTSClient, "delegated host mode requires STS client env", gcfg, iamMode, required)
	}
	return nil
}

// ValidateConfig 快速检查 Gateway 配置是否就绪（供启动前自检使用）。
func ValidateConfig(cfg *config.Config) error {
	if cfg == nil || cfg.Gateway == nil {
		return errors.New("gateway config missing")
	}
	base := effectiveGatewayBaseURL(cfg.Gateway)
	authScheme := effectiveGatewayAuthScheme(cfg.Gateway)
	credential := gatewayCredential(cfg.Gateway, authScheme)
	if base == "" || authScheme == "" {
		return errors.New("gateway config requires base_url + credential matching auth_scheme")
	}
	if authScheme == "apikey" && credential == "" {
		return errors.New("gateway config requires base_url + api key")
	}
	if authScheme == "bearer" && credential == "" && !isHostDelegatedMode(cfg) {
		return errors.New("gateway config requires bearer STS token provider")
	}
	return nil
}

func (c *Client) refreshCredentials(ctx context.Context) (bool, error) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	if c.cfg == nil || c.cfg.Gateway == nil {
		return false, fmt.Errorf("gateway config missing")
	}
	if isHostDelegatedMode(c.cfg) {
		if c.tokenProvider == nil {
			return false, fmt.Errorf("gateway STS token provider missing")
		}
		if token, err := c.tokenProvider(ctx); err != nil {
			return false, err
		} else if strings.TrimSpace(token) == "" {
			return false, fmt.Errorf("gateway STS token is empty")
		}
		if err := c.reconnectTransport(); err != nil {
			return false, err
		}
		return true, nil
	}
	if effectiveGatewayAuthScheme(c.cfg.Gateway) != "bearer" {
		return false, fmt.Errorf("gateway auth_scheme=%s 不支持自动刷新", effectiveGatewayAuthScheme(c.cfg.Gateway))
	}
	return false, fmt.Errorf("gateway bearer token refresh is deprecated; configure STS instead")
}

func (c *Client) reconnectTransport() error {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	baseURL := strings.TrimRight(effectiveGatewayBaseURL(gcfg), "/")
	authScheme := effectiveGatewayAuthScheme(gcfg)
	credential := gatewayCredential(gcfg, authScheme)

	if baseURL == "" || (authScheme == "apikey" && credential == "") || (authScheme == "bearer" && credential == "" && c.tokenProvider == nil) {
		return fmt.Errorf("PX_GATEWAY_BASE_URL 与 Gateway 凭证未配置（auth_scheme=%s）", authScheme)
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	client, err := frameworkgateway.NewClient(frameworkgateway.Config{
		BaseURL:        baseURL,
		AuthScheme:     authScheme,
		BearerToken:    credential,
		TokenProvider:  c.tokenProvider,
		APIKey:         strings.TrimSpace(gcfg.APIKey),
		RequestTimeout: timeout,
		UserAgent:      strings.TrimSpace(gcfg.UserAgent),
	})
	if err != nil {
		return err
	}
	if c.transport != nil {
		_ = c.transport.Close()
	}
	c.transport = client
	return nil
}

func (c *Client) gatewayCredential(ctx context.Context, authScheme string) (string, error) {
	if c == nil || c.cfg == nil || c.cfg.Gateway == nil {
		return "", fmt.Errorf("gateway config missing")
	}
	switch authScheme {
	case "apikey":
		credential := strings.TrimSpace(c.cfg.Gateway.APIKey)
		if credential == "" {
			return "", fmt.Errorf("PX_GATEWAY_API_KEY is required")
		}
		return credential, nil
	case "bearer":
		if c.tokenProvider != nil {
			token, err := c.tokenProvider(ctx)
			if err != nil {
				return "", fmt.Errorf("gateway STS token exchange failed: %w", err)
			}
			token = strings.TrimSpace(token)
			if token == "" {
				return "", fmt.Errorf("gateway STS token is empty")
			}
			return token, nil
		}
		return "", fmt.Errorf("gateway STS token provider is required")
	default:
		return "", fmt.Errorf("unsupported gateway auth scheme: %s", authScheme)
	}
}

func effectiveGatewayBaseURL(gcfg *config.GatewayConfig) string {
	if gcfg == nil {
		return ""
	}
	baseURL := strings.TrimRight(strings.TrimSpace(gcfg.BaseURL), "/")
	if baseURL == "" {
		return ""
	}
	apiPrefix := normalizeAPIPrefix(gcfg.APIPrefix)
	if apiPrefix == "" {
		return baseURL
	}
	if strings.HasSuffix(baseURL, apiPrefix) {
		return baseURL
	}
	return baseURL + apiPrefix
}

func normalizeAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return "/api/v1"
	}
	return value
}

type platformCapabilityResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []PlatformCapabilityRecord `json:"items"`
	} `json:"data"`
}

type platformAgentResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []AgentRecord `json:"items"`
	} `json:"data"`
}

type platformAgentSessionResponse struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    AgentSessionRecord `json:"data"`
}

type platformAgentSessionListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []AgentSessionRecord `json:"items"`
	} `json:"data"`
}

type platformAgentSessionMessageListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []AgentSessionMessageRecord `json:"items"`
	} `json:"data"`
}

type platformSimpleResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func firstSessionIdentifier(session AgentSessionRecord) string {
	if strings.TrimSpace(session.UUID) != "" {
		return strings.TrimSpace(session.UUID)
	}
	switch value := session.ID.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strconv.FormatUint(uint64(value), 10)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case json.Number:
		return strings.TrimSpace(value.String())
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func effectiveGatewayTenant(gcfg *config.GatewayConfig) string {
	return ""
}

func effectiveGatewayTenantForRequest(gcfg *config.GatewayConfig, tenantOverride string) string {
	if effectiveGatewayAuthScheme(gcfg) == "apikey" {
		return ""
	}
	if tenant := strings.TrimSpace(tenantOverride); tenant != "" {
		return tenant
	}
	return effectiveGatewayTenant(gcfg)
}

func effectiveGatewayAuthScheme(gcfg *config.GatewayConfig) string {
	if gcfg == nil {
		return "bearer"
	}
	explicit := strings.ToLower(strings.TrimSpace(gcfg.AuthScheme))
	switch explicit {
	case "bearer":
		return "bearer"
	case "apikey", "api_key", "api-key":
		return "apikey"
	}
	if strings.TrimSpace(gcfg.APIKey) != "" {
		return "apikey"
	}
	return "bearer"
}

func gatewayCredential(gcfg *config.GatewayConfig, authScheme string) string {
	if gcfg == nil {
		return ""
	}
	switch authScheme {
	case "bearer":
		return ""
	case "apikey":
		return strings.TrimSpace(gcfg.APIKey)
	default:
		return ""
	}
}

func buildGatewayAuthHeader(authScheme, credential string) string {
	switch authScheme {
	case "apikey":
		return "ApiKey " + strings.TrimSpace(credential)
	default:
		return "Bearer " + strings.TrimSpace(credential)
	}
}

func gatewayIAMMode(cfg *config.Config) string {
	if cfg == nil || cfg.Context == nil {
		return "unknown"
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Context.IAMMode))
	if mode == "" {
		return "unknown"
	}
	return mode
}

func isHostDelegatedMode(cfg *config.Config) bool {
	return cfg != nil &&
		strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" &&
		gatewayIAMMode(cfg) == "delegated"
}

func requiredGatewayEnv(cfg *config.Config, gcfg *config.GatewayConfig, authScheme string) []string {
	if isHostDelegatedMode(cfg) {
		return []string{
			"PX_GATEWAY_BASE_URL",
			"POWERX_STS_CLIENT_ID",
			"POWERX_STS_CLIENT_SECRET",
			"POWERX_GRPC_UPSTREAM_ADDRESS",
			"POWERX_GRPC_UPSTREAM_TENANT_UUID",
			"PX_GATEWAY_AUTH_SCHEME=bearer",
		}
	}
	if strings.EqualFold(authScheme, "apikey") || strings.TrimSpace(gcfgValueAPIKey(gcfg)) != "" {
		return []string{"PX_GATEWAY_BASE_URL", "PX_GATEWAY_API_KEY", "PX_GATEWAY_AUTH_SCHEME=apikey"}
	}
	return []string{"PX_GATEWAY_BASE_URL", "POWERX_STS_CLIENT_ID", "POWERX_STS_CLIENT_SECRET", "PX_GATEWAY_AUTH_SCHEME=bearer"}
}

func gcfgValueAPIKey(gcfg *config.GatewayConfig) string {
	if gcfg == nil {
		return ""
	}
	return gcfg.APIKey
}

func newGatewayConfigError(code, msg string, gcfg *config.GatewayConfig, iamMode string, required []string) *GatewayConfigError {
	present := make([]string, 0, 3)
	if gcfg != nil {
		if strings.TrimSpace(gcfg.BaseURL) != "" {
			present = append(present, "PX_GATEWAY_BASE_URL")
		}
		if strings.EqualFold(strings.TrimSpace(gcfg.AuthScheme), "bearer") {
			present = append(present, "PX_GATEWAY_AUTH_SCHEME=bearer")
		}
		if strings.EqualFold(strings.TrimSpace(gcfg.AuthScheme), "apikey") ||
			strings.EqualFold(strings.TrimSpace(gcfg.AuthScheme), "api_key") ||
			strings.EqualFold(strings.TrimSpace(gcfg.AuthScheme), "api-key") {
			present = append(present, "PX_GATEWAY_AUTH_SCHEME=apikey")
		}
		if strings.TrimSpace(gcfg.APIKey) != "" {
			present = append(present, "PX_GATEWAY_API_KEY")
		}
	}
	if strings.TrimSpace(os.Getenv("POWERX_STS_CLIENT_ID")) != "" {
		present = append(present, "POWERX_STS_CLIENT_ID")
	}
	if strings.TrimSpace(os.Getenv("POWERX_STS_CLIENT_SECRET")) != "" {
		present = append(present, "POWERX_STS_CLIENT_SECRET")
	}
	return &GatewayConfigError{
		Code:     code,
		Message:  msg,
		Required: required,
		Present:  present,
		IAMMode:  iamMode,
	}
}

func tenantUUIDFromJWT(token string) string {
	claims := decodeJWTClaims(token)
	if claims == nil {
		return ""
	}
	return strings.TrimSpace(firstClaimString(claims, "tid", "tenant_uuid"))
}
