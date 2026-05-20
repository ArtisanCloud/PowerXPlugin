package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	logger "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/google/uuid"
)

const (
	defaultRequestTimeout = 10 * time.Second
)

const (
	ErrCodeGatewayMissingBaseURL   = "GW_CFG_MISSING_BASE_URL"
	ErrCodeGatewayMissingToolToken = "GW_CFG_MISSING_TOOL_TOKEN"
	ErrCodeGatewayMissingAPIKey    = "GW_CFG_MISSING_API_KEY"
	ErrCodeGatewayInvalidScheme    = "GW_CFG_INVALID_AUTH_SCHEME"
	ErrCodeGatewayTokenInvalidTID  = "GW_TOKEN_INVALID_TID"
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
}

// NewClient 构造 Gateway Client；若凭证缺失，则进入离线模式。
func NewClient(cfg *config.Config, log *logger.Entry) *Client {
	c := &Client{
		logger:  ensureLogger(log),
		useMock: make(map[string]struct{}),
		cfg:     cfg,
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

	if baseURL == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayMissingBaseURL, "PX_GATEWAY_BASE_URL is required", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_PLUGIN_TOOL_TOKEN", "PX_GATEWAY_AUTH_SCHEME=bearer"})
		c.offlineReason = cfgErr.Error()
		return c
	}
	if authScheme == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayInvalidScheme, "gateway auth_scheme must be bearer or apikey", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_GATEWAY_AUTH_SCHEME"})
		c.offlineReason = cfgErr.Error()
		return c
	}
	if authScheme == "bearer" && credential == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayMissingToolToken, "PX_PLUGIN_TOOL_TOKEN is required", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_PLUGIN_TOOL_TOKEN", "PX_GATEWAY_AUTH_SCHEME=bearer"})
		c.offlineReason = cfgErr.Error()
		return c
	}
	if authScheme == "apikey" && credential == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayMissingAPIKey, "PX_GATEWAY_API_KEY is required", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_GATEWAY_API_KEY", "PX_GATEWAY_AUTH_SCHEME=apikey"})
		c.offlineReason = cfgErr.Error()
		return c
	}
	if authScheme == "bearer" && tenantUUIDFromJWT(strings.TrimSpace(gcfg.ToolToken)) == "" {
		cfgErr := newGatewayConfigError(ErrCodeGatewayTokenInvalidTID, "PX_PLUGIN_TOOL_TOKEN missing tid claim", gcfg, iamMode, []string{"PX_GATEWAY_BASE_URL", "PX_PLUGIN_TOOL_TOKEN", "PX_GATEWAY_AUTH_SCHEME=bearer"})
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
	baseURL := strings.TrimRight(effectiveGatewayBaseURL(gcfg), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	authScheme := effectiveGatewayAuthScheme(gcfg)
	credential := gatewayCredential(gcfg, authScheme)
	if credential == "" {
		return nil, fmt.Errorf("Gateway 凭证未配置（auth_scheme=%s）", authScheme)
	}
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

	endpoint := baseURL + "/tenant/capabilities"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
		}), "PX_PLUGIN_TOOL_TOKEN 自动刷新失败")
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
	if strings.TrimSpace(cfg.Gateway.ToolToken) != "" {
		return "gateway_tool"
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
	required := []string{"PX_GATEWAY_BASE_URL", "PX_PLUGIN_TOOL_TOKEN", "PX_GATEWAY_AUTH_SCHEME=bearer"}

	if effectiveGatewayBaseURL(gcfg) == "" {
		return newGatewayConfigError(ErrCodeGatewayMissingBaseURL, "PX_GATEWAY_BASE_URL is required", gcfg, iamMode, required)
	}
	if effectiveGatewayAuthScheme(gcfg) != "bearer" {
		return newGatewayConfigError(ErrCodeGatewayInvalidScheme, "gateway auth_scheme must be bearer", gcfg, iamMode, required)
	}
	token := gatewayCredential(gcfg, "bearer")
	if token == "" {
		return newGatewayConfigError(ErrCodeGatewayMissingToolToken, "PX_PLUGIN_TOOL_TOKEN is required", gcfg, iamMode, required)
	}
	if tenantUUIDFromJWT(token) == "" {
		return newGatewayConfigError(ErrCodeGatewayTokenInvalidTID, "PX_PLUGIN_TOOL_TOKEN missing tid claim", gcfg, iamMode, required)
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
	if base == "" || authScheme == "" || credential == "" {
		return errors.New("gateway config requires base_url + credential matching auth_scheme")
	}
	return nil
}

func (c *Client) refreshCredentials(ctx context.Context) (bool, error) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	if c.cfg == nil || c.cfg.Gateway == nil {
		return false, fmt.Errorf("gateway config missing")
	}
	if effectiveGatewayAuthScheme(c.cfg.Gateway) != "bearer" {
		return false, fmt.Errorf("gateway auth_scheme=%s 不支持自动刷新", effectiveGatewayAuthScheme(c.cfg.Gateway))
	}
	if strings.TrimSpace(c.cfg.Gateway.RefreshToken) == "" {
		return false, fmt.Errorf("PX_TOOL_REFRESH_TOKEN 未配置")
	}
	logger.InfoCtx(logger.WithLogFields(ctx, map[string]interface{}{
		"component":  "skeleton.gateway.client",
		"biz_scene":  "gateway_token_refresh",
		"biz_domain": "integration",
	}), "检测到 Gateway 凭证失败，尝试自动刷新 PX_PLUGIN_TOOL_TOKEN")
	if _, _, err := RefreshToolToken(ctx, c.cfg); err != nil {
		return false, err
	}
	if err := c.reconnectTransport(); err != nil {
		return false, err
	}
	logger.InfoCtx(logger.WithLogFields(ctx, map[string]interface{}{
		"component":  "skeleton.gateway.client",
		"biz_scene":  "gateway_token_refresh",
		"biz_domain": "integration",
	}), "PX_PLUGIN_TOOL_TOKEN 已刷新，准备重试 Gateway 调用")
	return true, nil
}

func (c *Client) reconnectTransport() error {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	baseURL := strings.TrimRight(effectiveGatewayBaseURL(gcfg), "/")
	authScheme := effectiveGatewayAuthScheme(gcfg)
	toolToken := strings.TrimSpace(gcfg.ToolToken)
	tenantUUID := effectiveGatewayTenant(gcfg)
	credential := gatewayCredential(gcfg, authScheme)

	if baseURL == "" || credential == "" {
		return fmt.Errorf("PX_GATEWAY_BASE_URL 与 Gateway 凭证未配置（auth_scheme=%s）", authScheme)
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	client, err := frameworkgateway.NewClient(frameworkgateway.Config{
		BaseURL:        baseURL,
		AuthScheme:     authScheme,
		BearerToken:    toolToken,
		APIKey:         strings.TrimSpace(gcfg.APIKey),
		TenantUUID:     tenantUUID,
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

func effectiveGatewayTenant(gcfg *config.GatewayConfig) string {
	if gcfg == nil {
		return ""
	}
	if tokenTenant := tenantUUIDFromJWT(strings.TrimSpace(gcfg.ToolToken)); tokenTenant != "" {
		return tokenTenant
	}
	return strings.TrimSpace(gcfg.TenantUUID)
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
	if strings.TrimSpace(gcfg.ToolToken) != "" {
		return "bearer"
	}
	if strings.TrimSpace(gcfg.APIKey) != "" {
		return "apikey"
	}
	return ""
}

func gatewayCredential(gcfg *config.GatewayConfig, authScheme string) string {
	if gcfg == nil {
		return ""
	}
	switch authScheme {
	case "bearer":
		return strings.TrimSpace(gcfg.ToolToken)
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
		if strings.TrimSpace(gcfg.ToolToken) != "" {
			present = append(present, "PX_PLUGIN_TOOL_TOKEN")
		}
		if strings.TrimSpace(gcfg.APIKey) != "" {
			present = append(present, "PX_GATEWAY_API_KEY")
		}
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
