package devapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/mtls"
	"github.com/powerx-plugin/cli/internal/watch"
)

// ClientOptions holds configuration for the Dev API client
type ClientOptions struct {
	BaseURL        string
	APIKey         string
	ReloadToken    string
	Timeout        time.Duration
	MaxRetries     int
	MTLSCertPath   string
	MTLSKeyPath    string
	MTLSCACertPath string
	MTLSClient     *mtls.Client
}

// DevClient is the Dev API client
type DevClient struct {
	baseURL     string
	apiKey      string
	reloadToken string
	httpClient  *http.Client
	maxRetries  int
	mtlsClient  *mtls.Client
	tlsConfig   *tls.Config
	mtlsEnabled bool
	mtlsOwned   bool
}

// NewClient creates a new Dev API client
func NewClient(opts ClientOptions) *DevClient {
	client := &DevClient{
		baseURL:     opts.BaseURL,
		apiKey:      opts.APIKey,
		reloadToken: opts.ReloadToken,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		maxRetries: 3,
	}

	// Set max retries if specified
	if opts.MaxRetries > 0 {
		client.maxRetries = opts.MaxRetries
	}

	// Initialize mTLS
	switch {
	case opts.MTLSClient != nil:
		client.mtlsClient = opts.MTLSClient
		client.mtlsEnabled = true
		client.tlsConfig = opts.MTLSClient.GetTLSConfig()
	case opts.MTLSCertPath != "" && opts.MTLSKeyPath != "" && opts.MTLSCACertPath != "":
		mtlsConfig := &mtls.Config{
			CertPath:   opts.MTLSCertPath,
			KeyPath:    opts.MTLSKeyPath,
			CAPath:     opts.MTLSCACertPath,
			ServerName: extractHostname(opts.BaseURL),
		}

		mtlsClient, err := mtls.NewClient(mtlsConfig)
		if err != nil {
			fmt.Printf("Warning: failed to initialize mTLS: %v\n", err)
		} else {
			client.mtlsClient = mtlsClient
			client.mtlsEnabled = true
			client.tlsConfig = mtlsClient.GetTLSConfig()
			client.mtlsOwned = true
		}
	}

	if client.tlsConfig != nil {
		client.httpClient.Transport = &http.Transport{
			TLSClientConfig: client.tlsConfig,
		}
	}

	return client
}

// RegisterRequest is the request for registering a dev session
type RegisterRequest struct {
	PluginID     string            `json:"pluginId"`
	Version      string            `json:"version"`
	EntryPath    string            `json:"entryPath"`
	Tenant       string            `json:"tenant,omitempty"`
	TenantID     uint64            `json:"tenantId,omitempty"`
	DeveloperID  uint64            `json:"developerId,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
}

// RegisterResponse is the response from registering a dev session
type RegisterResponse struct {
	SessionID   string `json:"sessionId"`
	ReloadToken string `json:"reloadToken"`
	DevUrl      string `json:"devUrl,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
}

// ReloadRequest is the request for reloading a dev session
type ReloadRequest struct {
	SessionID     string                 `json:"sessionId"`
	ReloadToken   string                 `json:"reloadToken,omitempty"`
	BundleHash    string                 `json:"bundleHash"`
	BundleSize    int64                  `json:"bundleSize"`
	BuildDuration int64                  `json:"buildDuration,omitempty"`
	Strategy      string                 `json:"strategy,omitempty"`
	ChangedFiles  []watch.FileEvent      `json:"changedFiles,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ReloadResponse is the response from reloading a dev session
type ReloadResponse struct {
	Status        string `json:"status"`
	ReloadID      string `json:"reloadId"`
	EstimatedTime int64  `json:"estimatedTime"`
	Message       string `json:"message"`
	Error         string `json:"error,omitempty"`
}

// StatusResponse is the response from getting session status
type StatusResponse struct {
	SessionID    string      `json:"sessionId"`
	Status       string      `json:"status"`
	PluginID     string      `json:"pluginId"`
	Version      string      `json:"version"`
	RegisteredAt time.Time   `json:"registeredAt"`
	LastReload   *time.Time  `json:"lastReload,omitempty"`
	ReloadCount  int         `json:"reloadCount"`
	Uptime       int         `json:"uptime"`
	BuildStats   *BuildStats `json:"buildStats,omitempty"`
}

// BuildStats represents build statistics
type BuildStats struct {
	AvgBuildTime      int     `json:"avgBuildTime"`
	SuccessRate       float64 `json:"successRate"`
	LastBuildDuration int     `json:"lastBuildDuration"`
}

// DeleteSessionsRequest requests bulk deletion.
type DeleteSessionsRequest struct {
	PluginID  string `json:"pluginId,omitempty"`
	TenantID  uint64 `json:"tenantId,omitempty"`
	Status    string `json:"status,omitempty"`
	Force     bool   `json:"force,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

// DeleteSessionsResponse contains deletion summary.
type DeleteSessionsResponse struct {
	Deleted    int      `json:"deleted"`
	Force      bool     `json:"force"`
	SessionIDs []string `json:"sessionIds"`
}

// ListSessionsFilter allows querying Dev API sessions with optional filters.
type ListSessionsFilter struct {
	PluginID    string
	TenantID    uint64
	DeveloperID uint64
	Status      string
	SessionID   string
}

// SessionRecord describes a session returned by the Dev API list endpoint.
type SessionRecord struct {
	SessionID   string    `json:"sessionId"`
	PluginID    string    `json:"pluginId"`
	Version     string    `json:"version"`
	Tenant      string    `json:"tenant"`
	TenantID    uint64    `json:"tenantId"`
	Status      string    `json:"status"`
	DeveloperID uint64    `json:"developerId"`
	DevURL      string    `json:"devUrl"`
	ReloadToken string    `json:"reloadToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	LastReload  time.Time `json:"lastReload"`
}

// Register registers a plugin for development
func (c *DevClient) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	url := fmt.Sprintf("%s/internal/dev/plugins/register", c.baseURL)

	headers := map[string]string{}
	if token := strings.TrimSpace(c.apiKey); token != "" {
		headers["X-API-Key"] = token
		headers["Authorization"] = "Bearer " + token
	}

	resp, err := c.makeRequest(ctx, http.MethodPost, url, req, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data, err := decodeAPIData(body)
	if err != nil {
		return nil, err
	}
	var registerResp RegisterResponse
	if len(data) == 0 {
		return nil, fmt.Errorf("dev api register response missing data")
	}
	if err := json.Unmarshal(data, &registerResp); err != nil {
		return nil, fmt.Errorf("failed to decode register response: %w", err)
	}
	return &registerResp, nil
}

// SetReloadToken updates the reload token used for subsequent reload/delete calls.
func (c *DevClient) SetReloadToken(token string) {
	c.reloadToken = token
}

// Reload triggers a hot reload of the plugin
func (c *DevClient) Reload(ctx context.Context, req *ReloadRequest) (*ReloadResponse, error) {
	url := fmt.Sprintf("%s/internal/dev/plugins/reload", c.baseURL)

	headers := map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(c.reloadToken),
		"X-Reload-Id":   uuid.NewString(),
	}

	resp, err := c.makeRequest(ctx, http.MethodPost, url, req, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data, err := decodeAPIData(body)
	if err != nil {
		return nil, err
	}
	var reloadResp ReloadResponse
	if len(data) == 0 {
		return nil, fmt.Errorf("dev api reload response missing data")
	}
	if err := json.Unmarshal(data, &reloadResp); err != nil {
		return nil, fmt.Errorf("failed to decode reload response: %w", err)
	}
	return &reloadResp, nil
}

// GetStatus retrieves the current session status
func (c *DevClient) GetStatus(ctx context.Context, sessionID string) (*StatusResponse, error) {
	url := fmt.Sprintf("%s/internal/dev/plugins/%s", c.baseURL, sessionID)

	headers := map[string]string{}
	if strings.TrimSpace(c.reloadToken) != "" {
		headers["Authorization"] = "Bearer " + strings.TrimSpace(c.reloadToken)
	}

	resp, err := c.makeRequest(ctx, http.MethodGet, url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data, err := decodeAPIData(body)
	if err != nil {
		return nil, err
	}
	var statusResp StatusResponse
	if len(data) == 0 {
		return nil, fmt.Errorf("dev api status response missing data")
	}
	if err := json.Unmarshal(data, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}
	return &statusResp, nil
}

// Delete unregisters a plugin
func (c *DevClient) Delete(ctx context.Context, sessionID string) error {
	url := fmt.Sprintf("%s/internal/dev/plugins/register/%s", c.baseURL, sessionID)

	headers := map[string]string{}
	if apiToken := strings.TrimSpace(c.apiKey); apiToken != "" {
		headers["Authorization"] = "Bearer " + apiToken
		headers["X-API-Key"] = apiToken
	} else if reloadToken := strings.TrimSpace(c.reloadToken); reloadToken != "" {
		headers["Authorization"] = "Bearer " + reloadToken
	}
	headers["Content-Type"] = "application/json"
	resp, err := c.makeRequest(ctx, http.MethodDelete, url, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if _, err := decodeAPIData(body); err != nil {
		return err
	}
	return nil
}

// DeleteSessions bulk deletes sessions via Dev API.
func (c *DevClient) DeleteSessions(ctx context.Context, reqPayload *DeleteSessionsRequest) (*DeleteSessionsResponse, error) {
	urlStr := fmt.Sprintf("%s/internal/dev/plugins/sessions", c.baseURL)
	if reqPayload != nil {
		values := url.Values{}
		if reqPayload.PluginID != "" {
			values.Set("pluginId", reqPayload.PluginID)
		}
		if reqPayload.TenantID != 0 {
			values.Set("tenantId", strconv.FormatUint(reqPayload.TenantID, 10))
		}
		if reqPayload.Status != "" {
			values.Set("status", reqPayload.Status)
		}
		if reqPayload.SessionID != "" {
			values.Set("sessionId", reqPayload.SessionID)
		}
		if reqPayload.Force {
			values.Set("force", "true")
			if reqPayload.Confirm {
				values.Set("confirm", "true")
			}
		}
		if encoded := values.Encode(); encoded != "" {
			urlStr = urlStr + "?" + encoded
		}
	}

	headers := map[string]string{}
	if apiToken := strings.TrimSpace(c.apiKey); apiToken != "" {
		headers["Authorization"] = "Bearer " + apiToken
		headers["X-API-Key"] = apiToken
	} else if reloadToken := strings.TrimSpace(c.reloadToken); reloadToken != "" {
		headers["Authorization"] = "Bearer " + reloadToken
	}
	if reqPayload != nil && reqPayload.Force && reqPayload.Confirm {
		headers["X-Force-Delete"] = "true"
	}

	resp, err := c.makeRequest(ctx, http.MethodDelete, urlStr, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data, err := decodeAPIData(body)
	if err != nil {
		return nil, err
	}
	var deleteResp DeleteSessionsResponse
	if len(data) == 0 {
		return &deleteResp, nil
	}
	if err := json.Unmarshal(data, &deleteResp); err != nil {
		return nil, fmt.Errorf("failed to decode delete sessions response: %w", err)
	}
	return &deleteResp, nil
}

// ListSessions fetches remote Dev API sessions.
func (c *DevClient) ListSessions(ctx context.Context, filter *ListSessionsFilter) ([]SessionRecord, error) {
	urlStr := fmt.Sprintf("%s/internal/dev/plugins/sessions", c.baseURL)
	if filter != nil {
		values := url.Values{}
		if filter.PluginID != "" {
			values.Set("pluginId", filter.PluginID)
		}
		if filter.TenantID != 0 {
			values.Set("tenantId", strconv.FormatUint(filter.TenantID, 10))
		}
		if filter.DeveloperID != 0 {
			values.Set("developerId", strconv.FormatUint(filter.DeveloperID, 10))
		}
		if filter.Status != "" {
			values.Set("status", filter.Status)
		}
		if filter.SessionID != "" {
			values.Set("sessionId", filter.SessionID)
		}
		if encoded := values.Encode(); encoded != "" {
			urlStr = urlStr + "?" + encoded
		}
	}

	headers := map[string]string{}
	if apiToken := strings.TrimSpace(c.apiKey); apiToken != "" {
		headers["Authorization"] = "Bearer " + apiToken
		headers["X-API-Key"] = apiToken
	}

	resp, err := c.makeRequest(ctx, http.MethodGet, urlStr, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data, err := decodeAPIData(body)
	if err != nil {
		return nil, err
	}
	sessions, err := decodeSessionList(data)
	if err != nil {
		return nil, fmt.Errorf("unexpected session list response: %s", strings.TrimSpace(string(data)))
	}
	return sessions, nil
}

func decodeSessionList(data []byte) ([]SessionRecord, error) {
	var direct []SessionRecord
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}

	var wrapper struct {
		Sessions []SessionRecord `json:"sessions"`
		Items    []SessionRecord `json:"items"`
		Data     json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Sessions) > 0 {
		return wrapper.Sessions, nil
	}
	if len(wrapper.Items) > 0 {
		return wrapper.Items, nil
	}
	if len(wrapper.Data) > 0 {
		if sessions, err := decodeSessionList(wrapper.Data); err == nil {
			return sessions, nil
		}
	}
	return []SessionRecord{}, nil
}

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeAPIData(body []byte) (json.RawMessage, error) {
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode dev api envelope: %w", err)
	}
	if env.Code != 0 && env.Code != http.StatusOK && env.Code != http.StatusCreated {
		msg := env.Message
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return nil, fmt.Errorf("dev api error %d: %s", env.Code, msg)
	}
	return env.Data, nil
}

// makeRequest performs an HTTP request with authentication
func (c *DevClient) makeRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	delay := time.Second

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		var req *http.Request
		if payload != nil {
			req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, url, nil)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "px-plugin-go-cli/1.0")
		for k, v := range headers {
			if strings.TrimSpace(v) != "" {
				req.Header.Set(k, v)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == c.maxRetries {
				return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries, err)
			}
			time.Sleep(delay)
			delay *= 2
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		if resp.StatusCode >= 500 && attempt < c.maxRetries {
			resp.Body.Close()
			time.Sleep(delay)
			delay *= 2
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if apiErr := c.parseAPIError(resp.StatusCode, respBody); apiErr != nil {
			return nil, apiErr
		}

		return nil, fmt.Errorf("request %s %s failed: %s", method, url, strings.TrimSpace(string(respBody)))
	}

	return nil, fmt.Errorf("request failed after %d attempts", c.maxRetries)
}

func (c *DevClient) parseAPIError(status int, body []byte) *DevAPIError {
	if len(body) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	var payload struct {
		Error   string      `json:"error"`
		Message string      `json:"message"`
		Code    interface{} `json:"code"`
	}
	_ = json.Unmarshal(body, &payload)

	code := ""
	switch v := payload.Code.(type) {
	case string:
		code = v
	case float64:
		code = strconv.FormatInt(int64(v), 10)
	case int:
		code = strconv.Itoa(v)
	}
	if code == "" {
		code = payload.Error
	}
	message := payload.Message
	if message == "" {
		message = strings.TrimSpace(string(body))
	}

	details := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		if k == "error" || k == "message" || k == "code" {
			continue
		}
		var decoded interface{}
		if err := json.Unmarshal(v, &decoded); err == nil {
			details[k] = decoded
		}
	}

	errType := ErrAPI
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		errType = ErrAuth
	}

	return &DevAPIError{
		Type:      errType,
		Code:      code,
		Message:   message,
		Original:  fmt.Errorf("status %d", status),
		Retryable: status >= 500,
		Status:    status,
		Details:   details,
	}
}

// extractHostname extracts the hostname from a URL
func extractHostname(url string) string {
	// Simple extraction - in production, use net/url
	// For now, just return a reasonable default
	if url == "" {
		return "localhost"
	}

	// Remove protocol if present
	hostname := url
	if strings.HasPrefix(hostname, "https://") {
		hostname = strings.TrimPrefix(hostname, "https://")
	} else if strings.HasPrefix(hostname, "http://") {
		hostname = strings.TrimPrefix(hostname, "http://")
	}

	// Remove path and port if present
	if idx := strings.Index(hostname, "/"); idx != -1 {
		hostname = hostname[:idx]
	}
	if idx := strings.Index(hostname, ":"); idx != -1 {
		hostname = hostname[:idx]
	}

	return hostname
}

// IsMTLSEnabled returns whether mTLS is enabled for this client
func (c *DevClient) IsMTLSEnabled() bool {
	return c.mtlsEnabled
}

// GetMTLSInfo returns mTLS certificate information
func (c *DevClient) GetMTLSInfo() (*mtls.CertInfo, error) {
	if !c.mtlsEnabled || c.mtlsClient == nil {
		return nil, fmt.Errorf("mTLS is not enabled")
	}

	return c.mtlsClient.GetCertificateInfo()
}

// CheckMTLSCertificate checks if the mTLS certificate is valid
func (c *DevClient) CheckMTLSCertificate() error {
	if !c.mtlsEnabled || c.mtlsClient == nil {
		return nil
	}

	return c.mtlsClient.CheckValidity()
}

// ReloadMTLSCertificates reloads mTLS certificates
func (c *DevClient) ReloadMTLSCertificates() error {
	if !c.mtlsEnabled || c.mtlsClient == nil {
		return nil
	}

	return c.mtlsClient.ReloadCertificates()
}

// Close closes the client and any associated resources
func (c *DevClient) Close() error {
	if c.mtlsClient != nil && c.mtlsOwned {
		c.mtlsClient.Close()
	}
	return nil
}
