package wsbus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/google/uuid"
)

const (
	defaultAPIPrefix    = "/api/v1"
	defaultPublishPath  = "/admin/runtime/ws-bus/publish"
	defaultRegisterPath = "/admin/runtime/ws-bus/grant"
)

type HostClientConfig struct {
	BaseURL      string
	APIPrefix    string
	AuthScheme   string
	Token        string
	APIKey       string
	TenantUUID   string
	UserAgent    string
	PublishPath  string
	RegisterPath string
	Timeout      time.Duration
	HTTPClient   *http.Client
}

type HostClient struct {
	baseURL      string
	apiPrefix    string
	authScheme   string
	credential   string
	tenantUUID   string
	userAgent    string
	publishPath  string
	registerPath string
	timeout      time.Duration
	httpClient   *http.Client
}

type hostPublishEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		OK bool `json:"ok"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

func NewHostClient(cfg HostClientConfig) (*HostClient, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("wsbus host: base URL is required")
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	authScheme, credential, err := resolveCredential(cfg.AuthScheme, cfg.Token, cfg.APIKey)
	if err != nil {
		return nil, err
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &HostClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiPrefix:    normalizeAPIPrefix(cfg.APIPrefix),
		authScheme:   authScheme,
		credential:   credential,
		tenantUUID:   strings.TrimSpace(cfg.TenantUUID),
		userAgent:    strings.TrimSpace(cfg.UserAgent),
		publishPath:  normalizeWSBusPath(cfg.PublishPath, defaultPublishPath),
		registerPath: normalizeWSBusPath(cfg.RegisterPath, defaultRegisterPath),
		timeout:      timeout,
		httpClient:   client,
	}, nil
}

func (c *HostClient) Publish(ctx context.Context, topic string, payload any, opts PublishOptions) PublishResult {
	if c == nil {
		return FailureResult(ErrorCodePublisherNotConfigured, "host client is not configured")
	}
	if strings.TrimSpace(topic) == "" {
		return FailureResult(ErrorCodeTopicRequired, "topic is required")
	}
	if payload == nil {
		return FailureResult(ErrorCodePayloadRequired, "payload is required")
	}
	tenantUUID := strings.TrimSpace(opts.TenantUUID)
	if tenantUUID == "" {
		tenantUUID = c.tenantUUID
	}

	body := map[string]any{
		"topic":   topic,
		"payload": payload,
	}
	if tenantUUID != "" {
		body["tenant_uuid"] = tenantUUID
	}
	if strings.TrimSpace(opts.TraceID) != "" {
		body["trace_id"] = strings.TrimSpace(opts.TraceID)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return FailureResult(ErrorCodePublishRequestInvalid, "failed to encode publish payload")
	}

	requestID := strings.TrimSpace(opts.TraceID)
	if requestID == "" {
		requestID = uuid.NewString()
	}

	return c.publishToEndpoint(ctx, c.buildEndpoint(c.publishPath), bodyBytes, tenantUUID, requestID, opts, "publish request")
}

func (c *HostClient) RegisterTopics(ctx context.Context, topics []string, opts PublishOptions) PublishResult {
	if c == nil {
		return FailureResult(ErrorCodePublisherNotConfigured, "host client is not configured")
	}
	expanded, result := ExpandTopicsForRegister(topics)
	if !result.OK {
		return result
	}

	tenantUUID := strings.TrimSpace(opts.TenantUUID)
	if tenantUUID == "" {
		tenantUUID = c.tenantUUID
	}

	body := map[string]any{
		"topics": expanded,
	}
	if tenantUUID != "" {
		body["tenant_uuid"] = tenantUUID
	}
	if strings.TrimSpace(opts.TraceID) != "" {
		body["trace_id"] = strings.TrimSpace(opts.TraceID)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return FailureResult(ErrorCodeRegisterRequestInvalid, "failed to encode register payload")
	}

	requestID := strings.TrimSpace(opts.TraceID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return c.registerTopicsToEndpoint(ctx, c.buildEndpoint(c.registerPath), bodyBytes, tenantUUID, requestID, opts, "grant request")
}

func (c *HostClient) registerTopicsToEndpoint(ctx context.Context, endpoint string, bodyBytes []byte, tenantUUID, requestID string, opts PublishOptions, label string) PublishResult {
	if ctx == nil {
		ctx = context.Background()
	}
	ctxReq, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxReq, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return FailureResult(ErrorCodeRegisterRequestInvalid, fmt.Sprintf("failed to build %s", label))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.resolveAuthHeader(opts))
	req.Header.Set("X-Request-ID", requestID)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	logWSBusHost(ctxReq, "info", "wsbus host register start", map[string]any{
		"endpoint":         endpoint,
		"label":            label,
		"request_id":       requestID,
		"tenant_uuid":      tenantUUID,
		"auth_header_kind": authHeaderKind(req.Header.Get("Authorization")),
		"payload_size":     len(bodyBytes),
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logWSBusHost(ctxReq, "warn", "wsbus host register http error", map[string]any{
			"endpoint":    endpoint,
			"label":       label,
			"request_id":  requestID,
			"tenant_uuid": tenantUUID,
			"error":       err.Error(),
		})
		result := FailureResult(ErrorCodeRegisterUpstreamFailed, err.Error())
		result.OutboundURL = endpoint
		return result
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logWSBusHost(ctxReq, "warn", "wsbus host register read error", map[string]any{
			"endpoint":    endpoint,
			"label":       label,
			"request_id":  requestID,
			"tenant_uuid": tenantUUID,
			"http_status": resp.StatusCode,
			"error":       err.Error(),
		})
		result := FailureResult(ErrorCodeRegisterResponseInvalid, fmt.Sprintf("failed to read %s response", label))
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		return result
	}
	responseBody := strings.TrimSpace(string(payloadBytes))
	logWSBusHost(ctxReq, "info", "wsbus host register response", map[string]any{
		"endpoint":      endpoint,
		"label":         label,
		"request_id":    requestID,
		"tenant_uuid":   tenantUUID,
		"http_status":   resp.StatusCode,
		"response_size": len(responseBody),
	})
	if resp.StatusCode >= http.StatusBadRequest {
		message := fmt.Sprintf("grant rejected with status %d", resp.StatusCode)
		if detail := extractHostErrorMessage(payloadBytes); detail != "" {
			message = detail
		}
		result := FailureResult(ErrorCodeRegisterUpstreamFailed, message)
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if len(payloadBytes) == 0 {
		result := SuccessResult()
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		return result
	}
	var envelope hostPublishEnvelope
	if err := json.Unmarshal(payloadBytes, &envelope); err != nil {
		result := FailureResult(ErrorCodeRegisterResponseInvalid, fmt.Sprintf("failed to decode %s response", label))
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if isLegacyHostSuccess(payloadBytes) {
		result := SuccessResult()
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if !envelope.Success {
		code := ErrorCodeRegisterUpstreamFailed
		message := envelope.Message
		if envelope.Error != nil {
			if strings.TrimSpace(envelope.Error.Code) != "" {
				code = envelope.Error.Code
			}
			if strings.TrimSpace(envelope.Error.Message) != "" {
				message = envelope.Error.Message
			}
		}
		if strings.TrimSpace(message) == "" {
			message = "grant rejected by host"
		}
		result := FailureResult(code, message)
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if envelope.Data.OK {
		result := SuccessResult()
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	result := FailureResult(ErrorCodeRegisterUpstreamFailed, "grant rejected by host")
	result.OutboundURL = endpoint
	result.HTTPStatus = resp.StatusCode
	result.ResponseBody = responseBody
	return result
}

func (c *HostClient) publishToEndpoint(ctx context.Context, endpoint string, bodyBytes []byte, tenantUUID, requestID string, opts PublishOptions, label string) PublishResult {
	if ctx == nil {
		ctx = context.Background()
	}
	ctxReq, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxReq, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return FailureResult(ErrorCodePublishRequestInvalid, fmt.Sprintf("failed to build %s", label))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.resolveAuthHeader(opts))
	req.Header.Set("X-Request-ID", requestID)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	logWSBusHost(ctxReq, "info", "wsbus host publish start", map[string]any{
		"endpoint":         endpoint,
		"label":            label,
		"request_id":       requestID,
		"tenant_uuid":      tenantUUID,
		"trace_id":         strings.TrimSpace(opts.TraceID),
		"auth_header_kind": authHeaderKind(req.Header.Get("Authorization")),
		"payload_size":     len(bodyBytes),
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logWSBusHost(ctxReq, "warn", "wsbus host publish http error", map[string]any{
			"endpoint":    endpoint,
			"label":       label,
			"request_id":  requestID,
			"tenant_uuid": tenantUUID,
			"trace_id":    strings.TrimSpace(opts.TraceID),
			"error":       err.Error(),
		})
		result := FailureResult(ErrorCodeHostPublishFailed, err.Error())
		result.OutboundURL = endpoint
		return result
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logWSBusHost(ctxReq, "warn", "wsbus host publish read error", map[string]any{
			"endpoint":    endpoint,
			"label":       label,
			"request_id":  requestID,
			"tenant_uuid": tenantUUID,
			"trace_id":    strings.TrimSpace(opts.TraceID),
			"http_status": resp.StatusCode,
			"error":       err.Error(),
		})
		result := FailureResult(ErrorCodePublishResponseInvalid, fmt.Sprintf("failed to read %s response", label))
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		return result
	}
	responseBody := strings.TrimSpace(string(payloadBytes))
	logWSBusHost(ctxReq, "info", "wsbus host publish response", map[string]any{
		"endpoint":      endpoint,
		"label":         label,
		"request_id":    requestID,
		"tenant_uuid":   tenantUUID,
		"trace_id":      strings.TrimSpace(opts.TraceID),
		"http_status":   resp.StatusCode,
		"response_size": len(responseBody),
	})

	if resp.StatusCode >= http.StatusBadRequest {
		message := fmt.Sprintf("publish rejected with status %d", resp.StatusCode)
		if detail := extractHostErrorMessage(payloadBytes); detail != "" {
			message = detail
		}
		result := FailureResult(ErrorCodePublishUpstreamRejected, message)
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}

	if len(payloadBytes) == 0 {
		result := SuccessResult()
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		return result
	}
	var envelope hostPublishEnvelope
	if err := json.Unmarshal(payloadBytes, &envelope); err != nil {
		result := FailureResult(ErrorCodePublishResponseInvalid, fmt.Sprintf("failed to decode %s response", label))
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if isLegacyHostSuccess(payloadBytes) {
		result := SuccessResult()
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if !envelope.Success {
		code := ErrorCodePublishUpstreamRejected
		message := envelope.Message
		if envelope.Error != nil {
			if strings.TrimSpace(envelope.Error.Code) != "" {
				code = envelope.Error.Code
			}
			if strings.TrimSpace(envelope.Error.Message) != "" {
				message = envelope.Error.Message
			}
		}
		if strings.TrimSpace(message) == "" {
			message = "publish rejected by host"
		}
		result := FailureResult(code, message)
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	if envelope.Data.OK {
		result := SuccessResult()
		result.OutboundURL = endpoint
		result.HTTPStatus = resp.StatusCode
		result.ResponseBody = responseBody
		return result
	}
	result := FailureResult(ErrorCodePublishUpstreamRejected, "publish rejected by host")
	result.OutboundURL = endpoint
	result.HTTPStatus = resp.StatusCode
	result.ResponseBody = responseBody
	return result
}

func isLegacyHostSuccess(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	var legacy struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(payload, &legacy); err != nil {
		return false
	}
	if legacy.Code == 200 {
		msg := strings.TrimSpace(strings.ToLower(legacy.Message))
		return msg == "" || msg == "success" || msg == "ok"
	}
	return false
}

func extractHostErrorMessage(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	var envelope hostPublishEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return ""
	}
	if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
		return envelope.Error.Message
	}
	if strings.TrimSpace(envelope.Message) != "" {
		return envelope.Message
	}
	return ""
}

func (c *HostClient) resolveAuthHeader(opts PublishOptions) string {
	if strings.EqualFold(strings.TrimSpace(c.authScheme), "apikey") {
		return fmt.Sprintf("ApiKey %s", c.credential)
	}
	return fmt.Sprintf("Bearer %s", c.credential)
}

func authHeaderKind(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "empty"
	}
	switch {
	case strings.HasPrefix(strings.ToLower(trimmed), "bearer "):
		return "bearer"
	case strings.HasPrefix(strings.ToLower(trimmed), "apikey "):
		return "apikey"
	default:
		return "unknown"
	}
}

func resolveCredential(authScheme, token, apiKey string) (string, string, error) {
	scheme := strings.ToLower(strings.TrimSpace(authScheme))
	if scheme == "" {
		if strings.TrimSpace(apiKey) != "" {
			scheme = "apikey"
		} else {
			scheme = "bearer"
		}
	}
	switch scheme {
	case "apikey", "api_key", "api-key":
		key := strings.TrimSpace(apiKey)
		if key == "" {
			return "", "", errors.New("wsbus host: api key is required (PX_GATEWAY_API_KEY)")
		}
		return "apikey", key, nil
	case "bearer":
		bearer := strings.TrimSpace(token)
		if bearer == "" {
			return "", "", errors.New("wsbus host: token is required (PX_PLUGIN_TOOL_TOKEN)")
		}
		return "bearer", bearer, nil
	default:
		return "", "", fmt.Errorf("wsbus host: unsupported auth scheme: %s", scheme)
	}
}

func logWSBusHost(ctx context.Context, level, message string, fields map[string]any) {
	facade := runtimelogging.NewFacade(ctx, nil).With(runtimelogging.Fields{
		runtimelogging.FieldComponent:  "wsbus.host_client",
		runtimelogging.FieldSubscriber: "wsbus.host_client",
		runtimelogging.FieldTopic:      "runtime_ops.ws_bus.host_client",
	})
	entry := runtimelogging.Entry{Fields: runtimelogging.Fields(fields)}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "warn", "warning":
		facade.Warn(message, entry)
	case "error":
		facade.Error(message, entry)
	default:
		facade.Info(message, entry)
	}
}

func normalizeWSBusPath(path string, fallback string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = fallback
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultAPIPrefix
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return ""
	}
	return value
}

func (c *HostClient) buildEndpoint(routePath string) string {
	base := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	route := "/" + strings.TrimLeft(strings.TrimSpace(routePath), "/")
	if c.apiPrefix == "" {
		return base + route
	}
	if strings.HasSuffix(base, c.apiPrefix) {
		return base + route
	}
	return base + c.apiPrefix + route
}
