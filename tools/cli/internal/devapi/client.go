package devapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Dev API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	mtlsConfig *tls.Config
	maxRetries int
	retryDelay time.Duration
}

// RegisterRequest is the request for registering a dev session
type RegisterRequest struct {
	Manifest map[string]interface{} `json:"manifest"`
	Tenant   string                 `json:"tenant,omitempty"`
}

// RegisterResponse is the response from registering a dev session
type RegisterResponse struct {
	SessionID    string `json:"sessionId"`
	ReloadToken  string `json:"reloadToken"`
	AdminPreview string `json:"adminPreviewUrl,omitempty"`
}

// ReloadRequest is the request for reloading a dev session
type ReloadRequest struct {
	SessionID     string            `json:"sessionId"`
	ReloadToken   string            `json:"reloadToken"`
	ChangedFiles  []ChangedFile     `json:"changedFiles"`
	Diagnostics   []Diagnostic      `json:"diagnostics,omitempty"`
	ReloadID      string            `json:"-"` // Used for x-reload-id header
}

// ChangedFile represents a changed file
type ChangedFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// Diagnostic represents a diagnostic message
type Diagnostic struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// ReloadResponse is the response from reloading a dev session
type ReloadResponse struct {
	Status  string `json:"status"`
	LogsRef string `json:"logsRef,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewClient creates a new Dev API client
func NewClient(baseURL string, opts ...ClientOption) *Client {
	client := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ClientOption is a function that configures the client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithMTLS sets up mTLS configuration
func WithMTLS(certPath, keyPath, caPath string) ClientOption {
	return func(c *Client) {
		// Load client certificate
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			// In production, this should be handled better
			fmt.Printf("Warning: failed to load mTLS cert: %v\n", err)
			return
		}

		// Load CA certificate
		caCert, err := loadCACert(caPath)
		if err != nil {
			fmt.Printf("Warning: failed to load CA cert: %v\n", err)
			return
		}

		c.mtlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCert,
		}
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithRetryDelay sets the base retry delay
func WithRetryDelay(delay time.Duration) ClientOption {
	return func(c *Client) {
		c.retryDelay = delay
	}
}

// Register registers a new dev session
func (c *Client) Register(req *RegisterRequest) (*RegisterResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *RegisterResponse
	err = c.doRequest(http.MethodPost, "/internal/dev/plugins/register", body, &resp)
	if err != nil {
		return nil, fmt.Errorf("register failed: %w", err)
	}

	return resp, nil
}

// Reload triggers a reload of a dev session
func (c *Client) Reload(req *ReloadRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare headers
	headers := make(map[string]string)
	if req.ReloadID != "" {
		headers["x-reload-id"] = req.ReloadID
	}

	return c.doRequestWithHeaders(http.MethodPost, "/internal/dev/plugins/reload", body, headers, nil)
}

// Delete deletes a dev session
func (c *Client) Delete(sessionID string) error {
	return c.doRequest(http.MethodDelete, fmt.Sprintf("/internal/dev/plugins/register/%s", sessionID), nil, nil)
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(method, path string, body []byte, result interface{}) error {
	return c.doRequestWithHeaders(method, path, body, nil, result)
}

// doRequestWithHeaders performs an HTTP request with custom headers and retry logic
func (c *Client) doRequestWithHeaders(method, path string, body []byte, headers map[string]string, result interface{}) error {
	var err error
	delay := c.retryDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Create request
		var req *http.Request
		if body != nil {
			req, err = http.NewRequest(method, c.baseURL+path, bytes.NewReader(body))
		} else {
			req, err = http.NewRequest(method, c.baseURL+path, nil)
		}
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "px-plugin-cli/1.0")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		// Set up transport with mTLS if configured
		transport := &http.Transport{}
		if c.mtlsConfig != nil {
			transport.TLSClientConfig = c.mtlsConfig
		}
		c.httpClient.Transport = transport

		// Make request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == c.maxRetries {
				return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries, err)
			}
			// Wait before retry
			time.Sleep(delay)
			delay = delay * 2 // Exponential backoff
			continue
		}
		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return fmt.Errorf("failed to unmarshal response: %w", err)
				}
			}
			return nil
		}

		// Check for duplicate reload (409)
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("duplicate reload request (x-reload-id already processed)")
		}

		// Check if retryable (5xx)
		if resp.StatusCode >= 500 && attempt < c.maxRetries {
			time.Sleep(delay)
			delay = delay * 2
			continue
		}

		// Not retryable
		var errResp ErrorResponse
		if len(respBody) > 0 {
			json.Unmarshal(respBody, &errResp)
		}
		return fmt.Errorf("request failed: %d %s", resp.StatusCode, errResp.Message)
	}

	return nil
}
