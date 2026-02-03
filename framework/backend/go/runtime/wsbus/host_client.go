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

const hostPublishPath = "/api/v1/admin/runtime/ws-bus/publish"

type HostClientConfig struct {
	BaseURL    string
	Token      string
	TenantUUID string
	UserAgent  string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type HostClient struct {
	baseURL    string
	token      string
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
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("wsbus host: token is required")
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
		token:      token,
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
	if tenantUUID == "" {
		return FailureResult(ErrorCodeTenantRequired, "tenant_uuid is required")
	}

	body := map[string]any{
		"topic":       topic,
		"payload":     payload,
		"tenant_uuid": tenantUUID,
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

	endpoint := c.baseURL + hostPublishPath
	if ctx == nil {
		ctx = context.Background()
	}
	ctxReq, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxReq, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return FailureResult(ErrorCodePublishRequestInvalid, "failed to build publish request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("X-PowerX-Tenant", tenantUUID)
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
		return FailureResult(ErrorCodePublishResponseInvalid, "failed to read publish response")
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
		return FailureResult(ErrorCodePublishResponseInvalid, "failed to decode publish response")
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
