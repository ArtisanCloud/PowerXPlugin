package devapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/powerx-plugin/cli/internal/watch"
)

func TestIntegration_FullWorkflow(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	registerReq := &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "0.1.0",
		EntryPath: "/path/to/test-plugin",
		Tenant:    "default",
	}

	registerResp, err := client.Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if registerResp.SessionID != "test-session" {
		t.Fatalf("unexpected session id: %s", registerResp.SessionID)
	}
	client.SetReloadToken(registerResp.ReloadToken)

	reloadReq := &ReloadRequest{
		SessionID:  registerResp.SessionID,
		BundleHash: "hash-123",
		BundleSize: 1024,
		Strategy:   "incremental",
		ChangedFiles: []watch.FileEvent{
			{Path: "src/main.go", Type: watch.EventModify},
		},
	}

	if _, err := client.Reload(ctx, reloadReq); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	statusResp, err := client.GetStatus(ctx, registerResp.SessionID)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if statusResp.PluginID != "test-plugin" {
		t.Fatalf("unexpected plugin id: %s", statusResp.PluginID)
	}

	if err := client.Delete(ctx, registerResp.SessionID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestIntegration_RegisterErrorHandling(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	t.Run("Conflict", func(t *testing.T) {
		req := &RegisterRequest{
			PluginID:  "conflict-plugin",
			Version:   "0.1.0",
			EntryPath: "/tmp/conflict",
		}
		if _, err := client.Register(ctx, req); err == nil {
			t.Fatal("expected conflict error")
		} else if apiErr, ok := err.(*DevAPIError); ok {
			if apiErr.Code != "DEV_CONFLICT" {
				t.Fatalf("unexpected code: %s", apiErr.Code)
			}
		} else {
			t.Fatalf("expected DevAPIError, got %T", err)
		}
	})

	t.Run("MissingFields", func(t *testing.T) {
		req := &RegisterRequest{
			EntryPath: "/tmp/missing",
		}
		if _, err := client.Register(ctx, req); err == nil {
			t.Fatal("expected validation error")
		} else if apiErr, ok := err.(*DevAPIError); ok {
			if apiErr.Code != "DEV_BAD_REQUEST" {
				t.Fatalf("unexpected code: %s", apiErr.Code)
			}
		} else {
			t.Fatalf("expected DevAPIError, got %T", err)
		}
	})
}

func TestIntegration_ReloadErrorHandling(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	t.Run("Unauthorized", func(t *testing.T) {
		req := &ReloadRequest{
			SessionID:  "test-session",
			BundleHash: "hash",
		}
		if _, err := client.Reload(ctx, req); err == nil {
			t.Fatal("expected unauthorized error")
		} else if apiErr, ok := err.(*DevAPIError); ok {
			if apiErr.Code != "DEV_UNAUTHORIZED" {
				t.Fatalf("unexpected code: %s", apiErr.Code)
			}
		} else {
			t.Fatalf("expected DevAPIError, got %T", err)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		registerResp, err := client.Register(ctx, &RegisterRequest{
			PluginID:  "test-plugin",
			Version:   "0.1.0",
			EntryPath: "/tmp/plugin",
		})
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}

		tokenClient := NewClient(ClientOptions{
			BaseURL:     mockAPI.URL(),
			ReloadToken: registerResp.ReloadToken,
		})

		req := &ReloadRequest{
			SessionID:  registerResp.SessionID,
			BundleHash: "conflict-hash",
		}
		if _, err := tokenClient.Reload(ctx, req); err == nil {
			t.Fatal("expected conflict error")
		} else if apiErr, ok := err.(*DevAPIError); ok {
			if apiErr.Code != "DEV_RELOAD_CONFLICT" {
				t.Fatalf("unexpected code: %s", apiErr.Code)
			}
		} else {
			t.Fatalf("expected DevAPIError, got %T", err)
		}
	})
}

func TestIntegration_StatusErrorHandling(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	if _, err := client.GetStatus(ctx, "test-session"); err == nil {
		t.Fatal("expected unauthorized error")
	} else if apiErr, ok := err.(*DevAPIError); ok {
		if apiErr.Code != "DEV_UNAUTHORIZED" {
			t.Fatalf("unexpected code: %s", apiErr.Code)
		}
	} else {
		t.Fatalf("expected DevAPIError, got %T", err)
	}
}

func TestIntegration_TimeoutHandling(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer slowServer.Close()

	client := NewClient(ClientOptions{
		BaseURL: slowServer.URL,
		Timeout: 50 * time.Millisecond,
	})

	ctx := context.Background()
	req := &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "0.1.0",
		EntryPath: "/tmp/plugin",
	}

	if _, err := client.Register(ctx, req); err == nil {
		t.Fatal("expected timeout error")
	} else if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_Concurrency(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()
	resp, err := client.Register(ctx, &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "0.1.0",
		EntryPath: "/tmp/plugin",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	tokenClient := NewClient(ClientOptions{
		BaseURL:     mockAPI.URL(),
		ReloadToken: resp.ReloadToken,
	})

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			req := &ReloadRequest{
				SessionID:  resp.SessionID,
				BundleHash: "concurrent-hash",
			}
			if _, err := tokenClient.Reload(ctx, req); err != nil {
				t.Errorf("reload failed: %v", err)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for reloads")
	}
}

func TestIntegration_Idempotency(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()
	resp, err := client.Register(ctx, &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "0.1.0",
		EntryPath: "/tmp/plugin",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	tokenClient := NewClient(ClientOptions{
		BaseURL:     mockAPI.URL(),
		ReloadToken: resp.ReloadToken,
	})

	req := &ReloadRequest{
		SessionID:  resp.SessionID,
		BundleHash: "idempotent-hash",
	}

	for i := 0; i < 3; i++ {
		if _, err := tokenClient.Reload(ctx, req); err != nil {
			t.Fatalf("reload attempt %d failed: %v", i+1, err)
		}
	}
}
