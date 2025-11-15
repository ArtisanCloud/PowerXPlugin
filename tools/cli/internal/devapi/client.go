package devapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	PluginID  string            `json:"pluginId"`
	Version   string            `json:"version"`
	EntryPath string            `json:"entryPath"`
	Tenant    string            `json:"tenant,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
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

// Register registers a plugin for development
func (c *DevClient) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	url := fmt.Sprintf("%s/internal/dev/plugins/register", c.baseURL)

	headers := map[string]string{}
	if strings.TrimSpace(c.apiKey) != "" {
		headers["X-API-Key"] = c.apiKey
	}

	resp, err := c.makeRequest(ctx, http.MethodPost, url, req, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var registerResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
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

	var reloadResp ReloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&reloadResp); err != nil {
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

	var statusResp StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &statusResp, nil
}

// Delete unregisters a plugin
func (c *DevClient) Delete(ctx context.Context, sessionID string) error {
	url := fmt.Sprintf("%s/internal/dev/plugins/register/%s", c.baseURL, sessionID)

	headers := map[string]string{}
	if strings.TrimSpace(c.reloadToken) != "" {
		headers["Authorization"] = "Bearer " + strings.TrimSpace(c.reloadToken)
	}

	resp, err := c.makeRequest(ctx, http.MethodDelete, url, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
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

	var payload struct {
		Error   string      `json:"error"`
		Message string      `json:"message"`
		Code    interface{} `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := ""
	switch v := payload.Code.(type) {
	case string:
		code = v
	case float64:
		code = strconv.FormatInt(int64(v), 10)
	}
	if code == "" {
		code = payload.Error
	}
	message := payload.Message
	if message == "" {
		message = strings.TrimSpace(string(body))
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
