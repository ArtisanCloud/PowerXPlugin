package devapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/powerx-plugin/cli/internal/watch"
)

// ClientOptions holds configuration for the Dev API client
type ClientOptions struct {
	BaseURL       string
	APIKey        string
	ReloadToken   string
	Timeout       time.Duration
	MaxRetries    int
	MTLSCertPath  string
	MTLSKeyPath   string
	MTLSCACertPath string
}

// DevClient is the Dev API client
type DevClient struct {
	baseURL     string
	apiKey      string
	reloadToken string
	httpClient  *http.Client
	maxRetries  int
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
	BundleHash     int64              `json:"bundleHash"`
	BundleSize     int64              `json:"bundleSize"`
	BuildDuration  int64              `json:"buildDuration,omitempty"`
	Strategy       string             `json:"strategy,omitempty"`
	ChangedFiles   []watch.FileEvent  `json:"changedFiles,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
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
	url := fmt.Sprintf("%s/api/v1/dev/register", c.baseURL)

	resp, err := c.makeRequest(ctx, "POST", url, req, c.apiKey)
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

// Reload triggers a hot reload of the plugin
func (c *DevClient) Reload(ctx context.Context, sessionID string, req *ReloadRequest) (*ReloadResponse, error) {
	url := fmt.Sprintf("%s/api/v1/dev/%s/reload", c.baseURL, sessionID)

	resp, err := c.makeRequest(ctx, "POST", url, req, "", c.reloadToken)
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
	url := fmt.Sprintf("%s/api/v1/dev/%s/status", c.baseURL, sessionID)

	resp, err := c.makeRequest(ctx, "GET", url, nil, "", c.reloadToken)
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
	url := fmt.Sprintf("%s/api/v1/dev/%s", c.baseURL, sessionID)

	resp, err := c.makeRequest(ctx, "DELETE", url, nil, "", c.reloadToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// makeRequest performs an HTTP request with authentication
func (c *DevClient) makeRequest(ctx context.Context, method, url string, body interface{}, apiKey, reloadToken string) (*http.Response, error) {
	// Create request body
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create HTTP request
	var err error
	var req *http.Request
	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "px-plugin-go-cli/1.0")

	// Add authentication
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if reloadToken != "" {
		req.Header.Set("Authorization", "Bearer "+reloadToken)
	}

	// Make request with retries
	var resp *http.Response
	delay := time.Second

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			if attempt == c.maxRetries {
				return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries, err)
			}
			time.Sleep(delay)
			delay = delay * 2
			continue
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Check for retryable errors
		if resp.StatusCode >= 500 && attempt < c.maxRetries {
			time.Sleep(delay)
			delay = delay * 2
			continue
		}

		// Non-retryable error
		break
	}

	// Return response for error handling by caller
	return resp, nil
}
