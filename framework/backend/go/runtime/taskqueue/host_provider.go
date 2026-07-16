package taskqueue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type tokenInvalidator interface {
	InvalidateToken()
}

type legacyTokenInvalidator interface {
	Invalidate()
}

type AuthenticationError struct {
	StatusCode int
	Body       string
}

func (e *AuthenticationError) Error() string {
	if e == nil {
		return ""
	}
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("taskqueue host authentication failed: status=%d", e.StatusCode)
	}
	return fmt.Sprintf("taskqueue host authentication failed: status=%d body=%s", e.StatusCode, body)
}

type HostProviderConfig struct {
	BaseURL       string
	APIPrefix     string
	AuthScheme    string
	APIKey        string
	Token         string
	TokenProvider TokenProvider
	TenantUUID    string
	UserAgent     string
	Timeout       time.Duration
	HTTPClient    *http.Client
}

type HostProvider struct {
	baseURL       string
	apiPrefix     string
	authScheme    string
	apiKey        string
	token         string
	tokenProvider TokenProvider
	tenantUUID    string
	userAgent     string
	httpClient    *http.Client
}

func NewHostProvider(cfg HostProviderConfig) (*HostProvider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("taskqueue host provider: base_url is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ua := strings.TrimSpace(cfg.UserAgent)
	if ua == "" {
		ua = "powerx-plugin-taskqueue/0.1"
	}
	authScheme := normalizeAuthScheme(cfg.AuthScheme, cfg.APIKey)
	if authScheme == "apikey" && strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("taskqueue host provider: api key is required for apikey auth")
	}
	if authScheme == "bearer" && strings.TrimSpace(cfg.Token) == "" && cfg.TokenProvider == nil {
		return nil, errors.New("taskqueue host provider: token provider is required for bearer auth")
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	return &HostProvider{
		baseURL:       baseURL,
		apiPrefix:     normalizeAPIPrefix(cfg.APIPrefix),
		authScheme:    authScheme,
		apiKey:        strings.TrimSpace(cfg.APIKey),
		token:         strings.TrimSpace(cfg.Token),
		tokenProvider: cfg.TokenProvider,
		tenantUUID:    strings.TrimSpace(cfg.TenantUUID),
		userAgent:     ua,
		httpClient:    httpClient,
	}, nil
}

func (p *HostProvider) Enqueue(ctx context.Context, req EnqueueRequest) error {
	_, err := p.do(ctx, http.MethodPost, "/admin/runtime/task-queue/enqueue", encodeMessageRequest(req), nil)
	return err
}

func (p *HostProvider) Dequeue(ctx context.Context, req DequeueRequest) ([]Message, error) {
	var out struct {
		Messages []hostMessage `json:"messages"`
	}
	_, err := p.do(ctx, http.MethodPost, "/admin/runtime/task-queue/dequeue", map[string]any{
		"tenant_key":      strings.TrimSpace(req.TenantKey),
		"subscriber_id":   strings.TrimSpace(req.SubscriberID),
		"max_items":       req.MaxItems,
		"wait_timeout_ms": int64(req.WaitTimeout / time.Millisecond),
	}, &out)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, 0, len(out.Messages))
	for _, item := range out.Messages {
		messages = append(messages, item.toMessage())
	}
	return messages, nil
}

func (p *HostProvider) Ack(ctx context.Context, req AckRequest) error {
	_, err := p.do(ctx, http.MethodPost, "/admin/runtime/task-queue/ack", req, nil)
	return err
}

func (p *HostProvider) Nack(ctx context.Context, req NackRequest) error {
	body := map[string]any{
		"tenant_key":    strings.TrimSpace(req.TenantKey),
		"subscriber_id": strings.TrimSpace(req.SubscriberID),
		"message_id":    strings.TrimSpace(req.MessageID),
		"reason":        strings.TrimSpace(req.Reason),
		"metadata":      req.Metadata,
	}
	if !req.RetryAt.IsZero() {
		body["retry_at"] = req.RetryAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := p.do(ctx, http.MethodPost, "/admin/runtime/task-queue/nack", body, nil)
	return err
}

func (p *HostProvider) Retry(ctx context.Context, req RetryRequest) error {
	body := encodeMessageRequest(EnqueueRequest{Message: req.Message})
	if !req.RetryAt.IsZero() {
		body["retry_at"] = req.RetryAt.UTC().Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(req.Reason) != "" {
		body["reason"] = strings.TrimSpace(req.Reason)
	}
	_, err := p.do(ctx, http.MethodPost, "/admin/runtime/task-queue/retry", body, nil)
	return err
}

func (p *HostProvider) do(ctx context.Context, method, path string, body any, out any) ([]byte, error) {
	if p == nil || p.httpClient == nil {
		return nil, errors.New("taskqueue host provider is not configured")
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, raw, err := p.doAttempt(ctx, method, path, bodyBytes)
	if err != nil {
		return raw, err
	}
	if isAuthStatus(resp.StatusCode) && p.authScheme == "bearer" && p.tokenProvider != nil {
		p.invalidateToken()
		resp, raw, err = p.doAttempt(ctx, method, path, bodyBytes)
		if err != nil {
			return raw, err
		}
	}
	if isAuthStatus(resp.StatusCode) {
		return raw, &AuthenticationError{StatusCode: resp.StatusCode, Body: string(raw)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, fmt.Errorf("taskqueue host request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	if out == nil {
		return raw, nil
	}
	var envelope struct {
		Success bool            `json:"success"`
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
		Error   any             `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Data) > 0 {
		success := envelope.Success || envelope.Code == 0 || (envelope.Code >= 200 && envelope.Code < 300)
		if !success {
			errText := strings.TrimSpace(fmt.Sprint(envelope.Error))
			if errText == "" || errText == "<nil>" {
				errText = strings.TrimSpace(envelope.Message)
			}
			if errText == "" {
				errText = string(raw)
			}
			return raw, fmt.Errorf("taskqueue host request failed: code=%d error=%s body=%s", envelope.Code, errText, string(raw))
		}
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return raw, fmt.Errorf("taskqueue host response data decode failed: %w body=%s", err, string(raw))
		}
		return raw, nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return raw, fmt.Errorf("taskqueue host response decode failed: %w body=%s", err, string(raw))
	}
	return raw, nil
}

func (p *HostProvider) doAttempt(ctx context.Context, method, path string, bodyBytes []byte) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, p.endpoint(path), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent)
	if p.tenantUUID != "" {
		req.Header.Set("tenant_uuid", p.tenantUUID)
	}
	switch p.authScheme {
	case "apikey":
		if p.apiKey != "" {
			req.Header.Set("Authorization", "ApiKey "+p.apiKey)
		}
	default:
		token := p.token
		if token == "" && p.tokenProvider != nil {
			token, err = p.tokenProvider.Token(ctx)
			if err != nil {
				return nil, nil, err
			}
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp, nil, readErr
	}
	return resp, raw, nil
}

func (p *HostProvider) endpoint(path string) string {
	return p.baseURL + p.apiPrefix + path
}

func (p *HostProvider) invalidateToken() {
	if p == nil || p.tokenProvider == nil {
		return
	}
	if provider, ok := p.tokenProvider.(tokenInvalidator); ok {
		provider.InvalidateToken()
		return
	}
	if provider, ok := p.tokenProvider.(legacyTokenInvalidator); ok {
		provider.Invalidate()
	}
}

func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

type hostMessage struct {
	ID             string            `json:"id"`
	TenantKey      string            `json:"tenant_key"`
	SubscriberID   string            `json:"subscriber_id"`
	Topic          string            `json:"topic"`
	Payload        string            `json:"payload"`
	PayloadBase64  string            `json:"payload_base64"`
	Headers        map[string]string `json:"headers,omitempty"`
	Attempt        int               `json:"attempt,omitempty"`
	TraceID        string            `json:"trace_id,omitempty"`
	VisibleAt      time.Time         `json:"visible_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

func (m hostMessage) toMessage() Message {
	payload := []byte(m.Payload)
	if strings.TrimSpace(m.PayloadBase64) != "" {
		if decoded, err := base64.StdEncoding.DecodeString(m.PayloadBase64); err == nil {
			payload = decoded
		}
	}
	return Message{
		ID:             m.ID,
		TenantKey:      m.TenantKey,
		SubscriberID:   m.SubscriberID,
		Topic:          m.Topic,
		Payload:        payload,
		Headers:        m.Headers,
		Attempt:        m.Attempt,
		TraceID:        m.TraceID,
		VisibleAt:      m.VisibleAt,
		Metadata:       m.Metadata,
		IdempotencyKey: m.IdempotencyKey,
	}
}

func encodeMessageRequest(req EnqueueRequest) map[string]any {
	msg := req.Message
	body := map[string]any{
		"id":              strings.TrimSpace(msg.ID),
		"tenant_key":      strings.TrimSpace(msg.TenantKey),
		"subscriber_id":   strings.TrimSpace(msg.SubscriberID),
		"topic":           strings.TrimSpace(msg.Topic),
		"payload_base64":  base64.StdEncoding.EncodeToString(msg.Payload),
		"headers":         msg.Headers,
		"attempt":         msg.Attempt,
		"trace_id":        strings.TrimSpace(msg.TraceID),
		"metadata":        msg.Metadata,
		"idempotency_key": strings.TrimSpace(msg.IdempotencyKey),
	}
	if !msg.VisibleAt.IsZero() {
		body["visible_at"] = msg.VisibleAt.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{"message": body}
}

func normalizeAPIPrefix(raw string) string {
	prefix := strings.TrimSpace(raw)
	if prefix == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

func normalizeAuthScheme(raw string, apiKey string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "apikey", "api_key", "api-key":
		return "apikey"
	case "bearer":
		return "bearer"
	default:
		if strings.TrimSpace(apiKey) != "" {
			return "apikey"
		}
		return "bearer"
	}
}
