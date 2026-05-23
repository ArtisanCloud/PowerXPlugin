package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultSchedulerJobsPath = "/admin/scheduler/jobs"

type HTTPHostClient struct {
	baseURL       string
	apiPrefix     string
	authScheme    string
	credential    string
	tokenProvider TokenProvider
	userAgent     string
	timeout       time.Duration
	httpClient    *http.Client
}

type schedulerEnvelope[T any] struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type schedulerEnvelopeProbe struct {
	Success *bool           `json:"success"`
	Code    *int            `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   json.RawMessage `json:"error"`
}

type schedulerListData struct {
	Items      []*Job `json:"items"`
	Total      int    `json:"total"`
	Pagination struct {
		Total int `json:"total"`
	} `json:"pagination"`
}

func NewHTTPHostClient(cfg HostProviderConfig) (*HTTPHostClient, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("scheduler host: base URL is required")
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	authScheme, credential, err := resolveSchedulerCredential(cfg.AuthScheme, cfg.Token, cfg.APIKey, cfg.TokenProvider != nil)
	if err != nil {
		return nil, err
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	return &HTTPHostClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		apiPrefix:     normalizeSchedulerAPIPrefix(cfg.APIPrefix),
		authScheme:    authScheme,
		credential:    credential,
		tokenProvider: cfg.TokenProvider,
		userAgent:     strings.TrimSpace(cfg.UserAgent),
		timeout:       timeout,
		httpClient:    httpClient,
	}, nil
}

func (c *HTTPHostClient) CreateJob(ctx context.Context, job JobSpec) (*Job, error) {
	job.TenantUUID = ""
	return requestScheduler[Job](c, ctx, http.MethodPost, defaultSchedulerJobsPath, nil, job)
}

func (c *HTTPHostClient) UpdateJob(ctx context.Context, job JobSpec) (*Job, error) {
	job.TenantUUID = ""
	return requestScheduler[Job](c, ctx, http.MethodPatch, defaultSchedulerJobsPath+"/"+url.PathEscape(strings.TrimSpace(job.JobID)), nil, job)
}

func (c *HTTPHostClient) PauseJob(ctx context.Context, jobID string, tenantUUID string) error {
	_, err := requestScheduler[map[string]any](c, ctx, http.MethodPost, defaultSchedulerJobsPath+"/"+url.PathEscape(strings.TrimSpace(jobID))+"/pause", nil, nil)
	return err
}

func (c *HTTPHostClient) ResumeJob(ctx context.Context, jobID string, tenantUUID string) error {
	_, err := requestScheduler[map[string]any](c, ctx, http.MethodPost, defaultSchedulerJobsPath+"/"+url.PathEscape(strings.TrimSpace(jobID))+"/resume", nil, nil)
	return err
}

func (c *HTTPHostClient) TriggerJob(ctx context.Context, jobID string, tenantUUID string) error {
	_, err := requestScheduler[map[string]any](c, ctx, http.MethodPost, defaultSchedulerJobsPath+"/"+url.PathEscape(strings.TrimSpace(jobID))+"/trigger", nil, nil)
	return err
}

func (c *HTTPHostClient) GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error) {
	return requestScheduler[Job](c, ctx, http.MethodGet, defaultSchedulerJobsPath+"/"+url.PathEscape(strings.TrimSpace(jobID)), nil, nil)
}

func (c *HTTPHostClient) ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error) {
	query := url.Values{}
	if strings.TrimSpace(in.OwnerType) != "" {
		query.Set("owner_type", strings.TrimSpace(in.OwnerType))
	}
	if strings.TrimSpace(in.OwnerID) != "" {
		query.Set("owner_id", strings.TrimSpace(in.OwnerID))
	}
	if strings.TrimSpace(in.Status) != "" {
		query.Set("status", strings.TrimSpace(in.Status))
	}
	data, err := requestScheduler[schedulerListData](c, ctx, http.MethodGet, defaultSchedulerJobsPath, query, nil)
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}

func requestScheduler[T any](c *HTTPHostClient, ctx context.Context, method, path string, query url.Values, body any) (*T, error) {
	if c == nil {
		return nil, ErrHostProviderUnavailable
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	endpoint := c.url(path, query)
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	authHeader, authErr := c.authHeader(ctx)
	if authErr != nil {
		return nil, authErr
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HostRequestError{
			StatusCode: resp.StatusCode,
			Endpoint:   endpoint,
			Body:       strings.TrimSpace(string(raw)),
		}
	}
	if schedulerLooksEnvelope(raw) {
		var probe schedulerEnvelopeProbe
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if len(probe.Error) > 0 && string(probe.Error) != "null" {
			var errBody struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(probe.Error, &errBody)
			return nil, fmt.Errorf("scheduler host request failed: %s", firstNonEmptyScheduler(errBody.Message, probe.Message, "request failed"))
		}
		if probe.Success != nil && !*probe.Success {
			return nil, fmt.Errorf("scheduler host request failed: %s", firstNonEmptyScheduler(probe.Message, "request failed"))
		}
		return decodeSchedulerData[T](probe.Data)
	}
	var data T
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func schedulerLooksEnvelope(raw []byte) bool {
	var probe schedulerEnvelopeProbe
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Success != nil || probe.Code != nil || len(probe.Data) > 0 || len(probe.Error) > 0
}

func (c *HTTPHostClient) url(path string, query url.Values) string {
	full := c.baseURL + c.apiPrefix + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	return full
}

func decodeSchedulerData[T any](raw json.RawMessage) (*T, error) {
	var zero T
	if len(raw) == 0 || string(raw) == "null" {
		return &zero, nil
	}
	var wrapped struct {
		Job json.RawMessage `json:"job"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Job) > 0 && string(wrapped.Job) != "null" {
		var job T
		if err := json.Unmarshal(wrapped.Job, &job); err != nil {
			return nil, err
		}
		return &job, nil
	}
	var direct T
	if err := json.Unmarshal(raw, &direct); err != nil {
		return nil, err
	}
	return &direct, nil
}

func (c *HTTPHostClient) authHeader(ctx context.Context) (string, error) {
	if c.authScheme == "apikey" {
		return "ApiKey " + c.credential, nil
	}
	if c.tokenProvider != nil {
		token, err := c.tokenProvider(ctx)
		if err != nil {
			return "", fmt.Errorf("scheduler host: STS token exchange failed: %w", err)
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return "", errors.New("scheduler host: STS token is empty")
		}
		return "Bearer " + token, nil
	}
	if token := strings.TrimSpace(c.credential); token != "" {
		return "Bearer " + token, nil
	}
	return "", errors.New("scheduler host: STS token provider is required for bearer mode")
}

func normalizeSchedulerAPIPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "/api/v1"
	}
	return "/" + strings.Trim(strings.TrimRight(trimmed, "/"), "/")
}

func resolveSchedulerCredential(authScheme, token, apiKey string, hasTokenProvider bool) (string, string, error) {
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
			return "", "", errors.New("scheduler host: api key is required")
		}
		return "apikey", key, nil
	case "bearer":
		bearer := strings.TrimSpace(token)
		if bearer == "" && !hasTokenProvider {
			return "", "", errors.New("scheduler host: STS token provider is required for bearer mode")
		}
		return "bearer", bearer, nil
	default:
		return "", "", fmt.Errorf("scheduler host: unsupported auth scheme: %s", scheme)
	}
}

func firstNonEmptyScheduler(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
