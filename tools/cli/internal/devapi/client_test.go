package devapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/powerx-plugin/cli/internal/watch"
)

func TestClient_Register(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/dev/plugins/register" {
			t.Errorf("Expected path /internal/dev/plugins/register, got %s", r.URL.Path)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"sessionId": "sess-123",
			"reloadToken": "token-456",
			"adminPreviewUrl": "/admin/dev/preview"
		}`))
	}))
	defer server.Close()

	// Create client
	client := NewClient(ClientOptions{BaseURL: server.URL})

	req := &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "1.0.0",
		EntryPath: "/tmp/plugin",
		Tenant:    "test-tenant",
	}

	ctx := context.Background()
	resp, err := client.Register(ctx, req)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	if resp == nil {
		t.Error("Register response is nil")
	}

	if resp.SessionID != "sess-123" {
		t.Errorf("Expected session ID 'sess-123', got '%s'", resp.SessionID)
	}

	if resp.ReloadToken != "token-456" {
		t.Errorf("Expected reload token 'token-456', got '%s'", resp.ReloadToken)
	}
}

func TestClient_Reload(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/dev/plugins/reload" {
			t.Errorf("Expected path /internal/dev/plugins/reload, got %s", r.URL.Path)
		}

		// Check for x-reload-id header
		reloadID := r.Header.Get("x-reload-id")
		if reloadID == "" {
			t.Error("Expected x-reload-id header")
		}

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "logsRef": "/admin/logs/123"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	client := NewClient(ClientOptions{BaseURL: server.URL})
	client.SetReloadToken("token-456")

	req := &ReloadRequest{
		SessionID:  "sess-123",
		BundleHash: "hash",
		BundleSize: 123,
		ChangedFiles: []watch.FileEvent{
			{Path: "test.go", Type: watch.EventModify},
		},
	}

	_, err := client.Reload(ctx, req)
	if err != nil {
		t.Errorf("Reload failed: %v", err)
	}
}

func TestClient_Delete(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/dev/plugins/register/sess-123" {
			t.Errorf("Expected path /internal/dev/plugins/register/sess-123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx := context.Background()
	client := NewClient(ClientOptions{BaseURL: server.URL})
	client.SetReloadToken("token-456")

	// Test delete
	err := client.Delete(ctx, "sess-123")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestClient_WithMaxRetries(t *testing.T) {
	client := NewClient(ClientOptions{BaseURL: "http://test", MaxRetries: 5})
	if client.maxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", client.maxRetries)
	}
}

func TestClient_WithRetryDelay(t *testing.T) {
	client := NewClient(ClientOptions{BaseURL: "http://test", Timeout: 5 * time.Second})
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}
