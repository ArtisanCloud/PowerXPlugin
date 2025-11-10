package devapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockDevAPI provides a mock Dev API server for testing
type MockDevAPI struct {
	server      *httptest.Server
	requests    []RequestLog
	reloadToken string
}

// RequestLog tracks API requests for verification
type RequestLog struct {
	Method  string
	Path    string
	Body    map[string]interface{}
	Headers map[string]string
	Time    time.Time
}

// NewMockDevAPI creates a new mock Dev API server
func NewMockDevAPI() *MockDevAPI {
	m := &MockDevAPI{
		requests:    make([]RequestLog, 0),
		reloadToken: "test-reload-token-12345",
	}

	// Create test server
	m.server = httptest.NewServer(http.HandlerFunc(m.handler))
	return m
}

// Close closes the mock server
func (m *MockDevAPI) Close() {
	m.server.Close()
}

// URL returns the server URL
func (m *MockDevAPI) URL() string {
	return m.server.URL
}

// handler implements the mock API
func (m *MockDevAPI) handler(w http.ResponseWriter, r *http.Request) {
	// Log the request
	body := make(map[string]interface{})
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	m.requests = append(m.requests, RequestLog{
		Method:  r.Method,
		Path:    r.URL.Path,
		Body:    body,
		Headers: headers,
		Time:    time.Now(),
	})

	// Route requests to match Go client paths
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/internal/dev/plugins/register":
		m.handleRegister(w, r, body)
	case r.Method == http.MethodPost && r.URL.Path == "/internal/dev/plugins/reload":
		m.handleReload(w, r, body)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/internal/dev/plugins/register/"):
		m.handleDelete(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/internal/dev/plugins/"):
		m.handleStatus(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "NOT_FOUND",
			"code":    "DEV_NOT_FOUND",
			"message": fmt.Sprintf("Mock API: Path %s not found", r.URL.Path),
		})
	}
}

// handleRegister handles registration requests
func (m *MockDevAPI) handleRegister(w http.ResponseWriter, r *http.Request, body map[string]interface{}) {
	getString := func(key string) string {
		if v, ok := body[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
		return ""
	}

	// Validate required fields
	if getString("pluginId") == "" || getString("version") == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "BAD_REQUEST",
			"message": "Missing required fields: pluginId, version",
			"code":    "DEV_BAD_REQUEST",
		})
		return
	}

	// Check if already registered (simulate conflict)
	if body["pluginId"] == "conflict-plugin" {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "CONFLICT",
			"message": "Plugin already registered",
			"code":    "DEV_CONFLICT",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":   "test-session",
		"reloadToken": m.reloadToken,
		"devUrl":      fmt.Sprintf("%s/dev/test-session", m.server.URL),
		"expiresAt":   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})
}

// handleReload handles reload requests
func (m *MockDevAPI) handleReload(w http.ResponseWriter, r *http.Request, body map[string]interface{}) {
	// Validate reload token
	auth := r.Header.Get("Authorization")
	if auth == "" || auth != "Bearer "+m.reloadToken {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
			"code":    "DEV_UNAUTHORIZED",
		})
		return
	}

	// Check if already reloading (simulate conflict)
	if body["bundleHash"] == "conflict-hash" {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "CONFLICT",
			"message": "Plugin is already reloading",
			"code":    "DEV_RELOAD_CONFLICT",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"reloadId":      "reload-12345",
		"estimatedTime": 100,
		"message":       "Plugin reloaded successfully",
	})
}

// handleDelete handles delete requests
func (m *MockDevAPI) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Validate reload token
	auth := r.Header.Get("Authorization")
	if auth == "" || auth != "Bearer "+m.reloadToken {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
			"code":    "DEV_UNAUTHORIZED",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"message":         "Plugin unregistered successfully",
		"sessionDuration": 3600,
	})
}

// handleStatus handles status requests
func (m *MockDevAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Validate reload token
	auth := r.Header.Get("Authorization")
	if auth == "" || auth != "Bearer "+m.reloadToken {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
			"code":    "DEV_UNAUTHORIZED",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":    "test-session",
		"status":       "active",
		"pluginId":     "test-plugin",
		"version":      "0.1.0",
		"registeredAt": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"lastReload":   time.Now().Format(time.RFC3339),
		"reloadCount":  5,
		"uptime":       3600,
		"buildStats": map[string]interface{}{
			"avgBuildTime":      2500,
			"successRate":       0.95,
			"lastBuildDuration": 2200,
		},
	})
}

// GetRequests returns the list of requests made to the mock API
func (m *MockDevAPI) GetRequests() []RequestLog {
	return m.requests
}

// GetLastRequest returns the last request made to the mock API
func (m *MockDevAPI) GetLastRequest() *RequestLog {
	if len(m.requests) == 0 {
		return nil
	}
	return &m.requests[len(m.requests)-1]
}

// ResetRequests clears the request log
func (m *MockDevAPI) ResetRequests() {
	m.requests = make([]RequestLog, 0)
}

// AssertRequest verifies that a request was made
func (m *MockDevAPI) AssertRequest(t *testing.T, method, path string) {
	t.Helper()

	if len(m.requests) == 0 {
		t.Errorf("Expected at least one request, got none")
		return
	}

	lastRequest := m.requests[len(m.requests)-1]
	if lastRequest.Method != method {
		t.Errorf("Expected method %q, got %q", method, lastRequest.Method)
	}
	if lastRequest.Path != path {
		t.Errorf("Expected path %q, got %q", path, lastRequest.Path)
	}
}

// AssertRequestCount verifies the number of requests
func (m *MockDevAPI) AssertRequestCount(t *testing.T, expected int) {
	t.Helper()

	actual := len(m.requests)
	if actual != expected {
		t.Errorf("Expected %d requests, got %d", expected, actual)
	}
}
