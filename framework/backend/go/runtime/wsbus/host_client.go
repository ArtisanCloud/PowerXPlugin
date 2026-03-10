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

	"github.com/google/uuid"
)

const (
	defaultAPIPrefix = "/api/v1"
	hostPublishPath  = "/internal/ws-bus/publish"
	hostRegisterPath = "/internal/ws-bus/grant"
)

type HostClientConfig struct {
	BaseURL    string
	APIPrefix  string
	AuthScheme string
	Token      string
	APIKey     string
	TenantUUID string
	UserAgent  string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type HostClient struct {
	baseURL    string
	apiPrefix  string
	authScheme string
	credential string
	tenantUUID string
	userAgent  string
	timeout    time.Duration
	httpClient *http.Client
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
	authScheme, credential, err := resolveAuth(cfg.AuthScheme, cfg.Token, cfg.APIKey)
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
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiPrefix:  normalizeAPIPrefix(cfg.APIPrefix),
		authScheme: authScheme,
		credential: credential,
		tenantUUID: strings.TrimSpace(cfg.TenantUUID),
		userAgent:  strings.TrimSpace(cfg.UserAgent),
		timeout:    timeout,
		httpClient: client,
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

	return c.publishToEndpoint(ctx, c.buildEndpoint(hostPublishPath), bodyBytes, tenantUUID, requestID, opts, "publish request")
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
	return c.registerTopicsToEndpoint(ctx, c.buildEndpoint(hostRegisterPath), bodyBytes, tenantUUID, requestID, opts, "grant request")
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FailureResult(ErrorCodeRegisterUpstreamFailed, err.Error())
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return FailureResult(ErrorCodeRegisterResponseInvalid, fmt.Sprintf("failed to read %s response", label))
	}
	if resp.StatusCode >= http.StatusBadRequest {
		message := fmt.Sprintf("grant rejected with status %d", resp.StatusCode)
		if detail := extractHostErrorMessage(payloadBytes); detail != "" {
			message = detail
		}
		return FailureResult(ErrorCodeRegisterUpstreamFailed, message)
	}
	if len(payloadBytes) == 0 {
		return SuccessResult()
	}
	var envelope hostPublishEnvelope
	if err := json.Unmarshal(payloadBytes, &envelope); err != nil {
		return FailureResult(ErrorCodeRegisterResponseInvalid, fmt.Sprintf("failed to decode %s response", label))
	}
	if isLegacyHostSuccess(payloadBytes) {
		return SuccessResult()
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
		return FailureResult(code, message)
	}
	if envelope.Data.OK {
		return SuccessResult()
	}
	return FailureResult(ErrorCodeRegisterUpstreamFailed, "grant rejected by host")
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FailureResult(ErrorCodeHostPublishFailed, err.Error())
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return FailureResult(ErrorCodePublishResponseInvalid, fmt.Sprintf("failed to read %s response", label))
	}

	if resp.StatusCode >= http.StatusBadRequest {
		message := fmt.Sprintf("publish rejected with status %d", resp.StatusCode)
		if detail := extractHostErrorMessage(payloadBytes); detail != "" {
			message = detail
		}
		return FailureResult(ErrorCodePublishUpstreamRejected, message)
	}

	if len(payloadBytes) == 0 {
		return SuccessResult()
	}
	var envelope hostPublishEnvelope
	if err := json.Unmarshal(payloadBytes, &envelope); err != nil {
		return FailureResult(ErrorCodePublishResponseInvalid, fmt.Sprintf("failed to decode %s response", label))
	}
	if isLegacyHostSuccess(payloadBytes) {
		return SuccessResult()
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
		return FailureResult(code, message)
	}
	if envelope.Data.OK {
		return SuccessResult()
	}
	return FailureResult(ErrorCodePublishUpstreamRejected, "publish rejected by host")
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
	if token := strings.TrimSpace(opts.BearerToken); token != "" {
		return fmt.Sprintf("Bearer %s", token)
	}
	if c.authScheme == "apikey" {
		return fmt.Sprintf("ApiKey %s", c.credential)
	}
	return fmt.Sprintf("Bearer %s", c.credential)
}

func resolveAuth(rawScheme, bearerToken, apiKey string) (scheme string, credential string, err error) {
	scheme = normalizeAuthScheme(rawScheme)
	bearer := strings.TrimSpace(bearerToken)
	key := strings.TrimSpace(apiKey)
	switch scheme {
	case "apikey":
		if key == "" {
			return "", "", errors.New("wsbus host: api key is required when auth_scheme=apikey")
		}
		return scheme, key, nil
	case "bearer":
		if bearer == "" {
			return "", "", errors.New("wsbus host: token is required when auth_scheme=bearer")
		}
		return scheme, bearer, nil
	default:
		if key != "" {
			return "apikey", key, nil
		}
		if bearer != "" {
			return "bearer", bearer, nil
		}
		return "", "", errors.New("wsbus host: missing credential (token/api key)")
	}
}

func normalizeAuthScheme(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "apikey", "api_key", "api-key":
		return "apikey"
	case "bearer":
		return "bearer"
	default:
		return ""
	}
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
