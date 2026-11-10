package taskqueue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
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

// HostError preserves machine diagnostics without switching the transport.
type HostError struct {
	StatusCode               int
	Code, TraceID, RequestID string
}

func (e *HostError) Error() string {
	return fmt.Sprintf("TASK_QUEUE_HOST_ERROR status=%d code=%s trace=%s request=%s", e.StatusCode, e.Code, e.TraceID, e.RequestID)
}
func hostError(status int, raw []byte) error {
	var v struct {
		ErrorCode string `json:"error_code"`
		Reason    string `json:"reason_code"`
		Trace     string `json:"trace_id"`
		Request   string `json:"request_id"`
	}
	_ = json.Unmarshal(raw, &v)
	code := v.ErrorCode
	if code == "" {
		code = v.Reason
	}
	if code == "" {
		code = "TASK_QUEUE_UPSTREAM_ERROR"
	}
	return &HostError{StatusCode: status, Code: code, TraceID: v.Trace, RequestID: v.Request}
}

type HostProvider struct {
	workerID      string
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
		workerID:      uuid.NewString(),
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

const enqueueCapability = "com.corex.runtime.taskqueue.service_enqueue"
const consumeCapability = "com.corex.runtime.taskqueue.service_consume"

func (p *HostProvider) invoke(ctx context.Context, capability string, body map[string]any, out any) error {
	body["contract_version"] = ContractVersion
	var response struct {
		Status   string          `json:"status"`
		Protocol string          `json:"protocol_used"`
		Fallback *bool           `json:"fallback_used"`
		Result   json.RawMessage `json:"result"`
	}
	_, err := p.do(ctx, http.MethodPost, "/tenant/invocations", map[string]any{"capability_id": capability, "preferred_protocol": "core_internal", "payload": map[string]any{"body": body}}, &response)
	if err != nil {
		return err
	}
	if response.Status != "completed" || response.Protocol != "core_internal" || response.Fallback == nil || *response.Fallback || len(response.Result) == 0 {
		return errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	var contract struct {
		Version string `json:"contract_version"`
	}
	if json.Unmarshal(response.Result, &contract) != nil || contract.Version != ContractVersion {
		return errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	if json.Unmarshal(response.Result, out) != nil {
		return errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	return nil
}
func (p *HostProvider) Enqueue(ctx context.Context, req EnqueueRequest) error {
	if err := validateMessage(req.Message); err != nil {
		return err
	}
	body := encodeMessageRequest(req)
	body["operation"] = "enqueue"
	body["tenant_key"] = req.Message.TenantKey
	var out struct {
		Accepted bool   `json:"accepted"`
		ID       string `json:"message_id"`
		Key      string `json:"idempotency_key"`
	}
	if err := p.invoke(ctx, enqueueCapability, body, &out); err != nil {
		return err
	}
	if !out.Accepted || out.ID != req.Message.ID || out.Key != req.Message.IdempotencyKey {
		return errors.New("TASK_QUEUE_ACCEPTANCE_INVALID")
	}
	return nil
}
func (p *HostProvider) Dequeue(ctx context.Context, req DequeueRequest) ([]Message, error) {
	if req.MaxItems == 0 {
		req.MaxItems = 1
	}
	if req.WaitTimeout == 0 {
		req.WaitTimeout = time.Second * 3
	}
	var out struct {
		Messages *[]hostMessage `json:"messages"`
	}
	err := p.invoke(ctx, consumeCapability, map[string]any{"operation": "dequeue", "tenant_key": req.TenantKey, "subscriber_id": req.SubscriberID, "worker_id": p.workerID, "max_items": req.MaxItems, "wait_timeout_ms": req.WaitTimeout.Milliseconds()}, &out)
	if err != nil {
		return nil, err
	}
	if out.Messages == nil {
		return nil, errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	messages := make([]Message, 0, len(*out.Messages))
	for _, v := range *out.Messages {
		msg, err := v.toMessage()
		if err != nil {
			return nil, err
		}
		if msg.TenantKey != req.TenantKey || msg.SubscriberID != req.SubscriberID || !validLease(msg.Lease) {
			return nil, errors.New("TASK_QUEUE_RESPONSE_INVALID")
		}
		messages = append(messages, msg)
	}
	return messages, nil
}
func (p *HostProvider) settle(ctx context.Context, operation, tenant, subscriber, id, receipt string, extra map[string]any) error {
	if receipt == "" {
		return errors.New("TASK_QUEUE_RECEIPT_REQUIRED")
	}
	extra["operation"] = operation
	extra["tenant_key"] = tenant
	extra["subscriber_id"] = subscriber
	extra["message_id"] = id
	extra["receipt_uuid"] = receipt
	var out struct {
		Confirmed bool   `json:"confirmed"`
		Receipt   string `json:"receipt_uuid"`
	}
	if err := p.invoke(ctx, consumeCapability, extra, &out); err != nil {
		return err
	}
	if !out.Confirmed || out.Receipt != receipt {
		return errors.New("TASK_QUEUE_CONFIRMATION_INVALID")
	}
	return nil
}
func (p *HostProvider) Ack(ctx context.Context, req AckRequest) error {
	return p.settle(ctx, "ack", req.TenantKey, req.SubscriberID, req.MessageID, req.ReceiptUUID, map[string]any{"metadata": req.Metadata, "outcome": req.Outcome})
}
func (p *HostProvider) Nack(ctx context.Context, req NackRequest) error {
	extra := map[string]any{"reason": req.Reason, "metadata": req.Metadata}
	if !req.RetryAt.IsZero() {
		extra["retry_at"] = req.RetryAt.UTC()
	}
	return p.settle(ctx, "nack", req.TenantKey, req.SubscriberID, req.MessageID, req.ReceiptUUID, extra)
}
func (p *HostProvider) Renew(ctx context.Context, req RenewRequest) (Lease, error) {
	var out struct {
		Lease *Lease `json:"lease"`
	}
	err := p.invoke(ctx, consumeCapability, map[string]any{"operation": "renew", "tenant_key": req.TenantKey, "subscriber_id": req.SubscriberID, "message_id": req.MessageID, "receipt_uuid": req.ReceiptUUID}, &out)
	if err != nil {
		return Lease{}, err
	}
	if !validLease(out.Lease) || out.Lease.ReceiptUUID != req.ReceiptUUID {
		return Lease{}, errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	return *out.Lease, nil
}

// Retry is a settlement of the current delivery, never a new enqueue.
func (p *HostProvider) Retry(ctx context.Context, req RetryRequest) error {
	if req.Message.Lease == nil {
		return errors.New("TASK_QUEUE_RECEIPT_REQUIRED")
	}
	return p.Nack(ctx, NackRequest{TenantKey: req.Message.TenantKey, SubscriberID: req.Message.SubscriberID, MessageID: req.Message.ID, ReceiptUUID: req.Message.Lease.ReceiptUUID, RetryAt: req.RetryAt, Reason: req.Reason})
}
func validLease(v *Lease) bool {
	if v == nil || v.Fence == 0 || !v.ExpiresAt.After(time.Now()) {
		return false
	}
	id, err := uuid.Parse(v.ReceiptUUID)
	return err == nil && id != uuid.Nil
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
		return raw, hostError(resp.StatusCode, raw)
	}
	var envelope struct {
		Success   *bool           `json:"success"`
		ErrorCode string          `json:"error_code"`
		Code      *int            `json:"code"`
		Data      json.RawMessage `json:"data"`
		Error     json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Code != nil && (*envelope.Code < 200 || *envelope.Code >= 300) {
		return raw, hostError(resp.StatusCode, raw)
	}
	if envelope.Success != nil && !*envelope.Success || envelope.ErrorCode != "" {
		return raw, hostError(resp.StatusCode, raw)
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Code == nil || *envelope.Code < 200 || *envelope.Code >= 300 || len(envelope.Data) == 0 || string(envelope.Data) == "null" || len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		return raw, errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	if out == nil || json.Unmarshal(envelope.Data, out) != nil {
		return raw, errors.New("TASK_QUEUE_RESPONSE_INVALID")
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
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
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
	Lease          *Lease            `json:"lease"`
	ID             string            `json:"id"`
	TenantKey      string            `json:"tenant_key"`
	SubscriberID   string            `json:"subscriber_id"`
	Topic          string            `json:"topic"`
	PayloadBase64  string            `json:"payload_base64"`
	Headers        map[string]string `json:"headers,omitempty"`
	Attempt        int               `json:"attempt,omitempty"`
	TraceID        string            `json:"trace_id,omitempty"`
	VisibleAt      time.Time         `json:"visible_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

func (m hostMessage) toMessage() (Message, error) {
	payload, err := base64.StdEncoding.Strict().DecodeString(m.PayloadBase64)
	if err != nil || base64.StdEncoding.EncodeToString(payload) != m.PayloadBase64 || m.ID == "" || m.IdempotencyKey == "" {
		return Message{}, errors.New("TASK_QUEUE_RESPONSE_INVALID")
	}
	return Message{ID: m.ID, TenantKey: m.TenantKey, SubscriberID: m.SubscriberID, Topic: m.Topic, Payload: payload, Headers: m.Headers, Attempt: m.Attempt, TraceID: m.TraceID, VisibleAt: m.VisibleAt, Metadata: m.Metadata, IdempotencyKey: m.IdempotencyKey, Lease: m.Lease}, nil
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
