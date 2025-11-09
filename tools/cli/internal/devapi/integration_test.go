package devapi

import (
	"context"
	"testing"
	"time"
)

// TestIntegration_FullWorkflow tests the complete dev workflow with mock API
func TestIntegration_FullWorkflow(t *testing.T) {
	// Create mock Dev API
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	// Create Dev API client pointing to mock server
	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	// Test 1: Register plugin
	t.Run("Register Plugin", func(t *testing.T) {
		req := &RegisterRequest{
			PluginID:  "test-plugin",
			Version:   "0.1.0",
			EntryPath: "/path/to/test-plugin",
			Tenant:    "default",
			Metadata: map[string]string{
				"buildCommand": "npm run build",
			},
		}

		resp, err := client.Register(ctx, req)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Verify response
		if resp == nil {
			t.Fatal("Register response is nil")
		}

		if resp.SessionID != "test-session" {
			t.Errorf("Expected session ID 'test-session', got %q", resp.SessionID)
		}

		if resp.ReloadToken == "" {
			t.Error("Expected non-empty reload token")
		}

		// Verify request was logged
		mockAPI.AssertRequest(t, "POST", "/api/v1/dev/register")
	})

	// Test 2: Reload plugin
	t.Run("Reload Plugin", func(t *testing.T) {
		mockAPI.ResetRequests() // Reset for this test

		reloadReq := &ReloadRequest{
			BundleHash:    "test-hash-123",
			BundleSize:    1234567,
			BuildDuration: 2500,
			Strategy:      "incremental",
			ChangedFiles: []FileEvent{
				{Path: "src/main.go", Type: "modify"},
			},
		}

		reloadResp, err := client.Reload(ctx, "test-session", reloadReq)
		if err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		// Verify response
		if reloadResp == nil {
			t.Fatal("Reload response is nil")
		}

		if reloadResp.Status != "success" {
			t.Errorf("Expected status 'success', got %q", reloadResp.Status)
		}

		if reloadResp.ReloadID == "" {
			t.Error("Expected non-empty reload ID")
		}

		// Verify request was logged
		mockAPI.AssertRequest(t, "POST", "/api/v1/dev/test-session/reload")
	})

	// Test 3: Get plugin status
	t.Run("Get Status", func(t *testing.T) {
		mockAPI.ResetRequests() // Reset for this test

		statusResp, err := client.GetStatus(ctx, "test-session")
		if err != nil {
			t.Fatalf("GetStatus failed: %v", err)
		}

		// Verify response
		if statusResp == nil {
			t.Fatal("Status response is nil")
		}

		if statusResp.Status != "active" {
			t.Errorf("Expected status 'active', got %q", statusResp.Status)
		}

		if statusResp.PluginID != "test-plugin" {
			t.Errorf("Expected plugin ID 'test-plugin', got %q", statusResp.PluginID)
		}

		if statusResp.ReloadCount == 0 {
			t.Error("Expected non-zero reload count")
		}

		// Verify request was logged
		mockAPI.AssertRequest(t, "GET", "/api/v1/dev/test-session/status")
	})

	// Test 4: Delete plugin
	t.Run("Delete Plugin", func(t *testing.T) {
		mockAPI.ResetRequests() // Reset for this test

		err := client.Delete(ctx, "test-session")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify request was logged
		mockAPI.AssertRequest(t, "DELETE", "/api/v1/dev/test-session")
	})
}

// TestIntegration_RegisterErrorHandling tests error handling in registration
func TestIntegration_RegisterErrorHandling(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	// Test: Registration conflict
	t.Run("Registration Conflict", func(t *testing.T) {
		mockAPI.ResetRequests()

		req := &RegisterRequest{
			PluginID:  "conflict-plugin", // This will trigger a conflict
			Version:   "0.1.0",
			EntryPath: "/path/to/conflict",
			Tenant:    "default",
		}

		resp, err := client.Register(ctx, req)
		if err == nil {
			t.Fatal("Expected error for conflicting registration, got none")
		}

		if resp != nil {
			t.Error("Expected nil response for conflicting registration")
		}

		// Verify error type
		if _, ok := err.(*DevAPIError); !ok {
			t.Errorf("Expected DevAPIError type, got %T", err)
		}
	})

	// Test: Missing required fields
	t.Run("Missing Required Fields", func(t *testing.T) {
		mockAPI.ResetRequests()

		req := &RegisterRequest{
			// Missing pluginId and version
			EntryPath: "/path/to/test",
			Tenant:    "default",
		}

		resp, err := client.Register(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing fields, got none")
		}

		if resp != nil {
			t.Error("Expected nil response for invalid request")
		}
	})
}

// TestIntegration_ReloadErrorHandling tests error handling in reload
func TestIntegration_ReloadErrorHandling(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	// Test: Reload without token (unauthorized)
	t.Run("Reload Unauthorized", func(t *testing.T) {
		mockAPI.ResetRequests()

		// Create client without reload token
		emptyClient := NewClient(ClientOptions{
			BaseURL: mockAPI.URL(),
			// No ReloadToken set
		})

		reloadReq := &ReloadRequest{
			BundleHash: "test-hash",
			BundleSize: 123456,
		}

		_, err := emptyClient.Reload(ctx, "test-session", reloadReq)
		if err == nil {
			t.Fatal("Expected error for unauthorized reload, got none")
		}

		// Verify error is DevAPIError with 401 status
		if apiErr, ok := err.(*DevAPIError); ok {
			if apiErr.Code != 1001 {
				t.Errorf("Expected error code 1001 (unauthorized), got %d", apiErr.Code)
			}
		} else {
			t.Errorf("Expected DevAPIError type, got %T", err)
		}
	})

	// Test: Reload conflict (already reloading)
	t.Run("Reload Conflict", func(t *testing.T) {
		mockAPI.ResetRequests()

		// First register to get token
		registerReq := &RegisterRequest{
			PluginID:  "test-plugin",
			Version:   "0.1.0",
			EntryPath: "/path/to/test",
			Tenant:    "default",
		}

		resp, err := client.Register(ctx, registerReq)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Create client with reload token
		tokenClient := NewClient(ClientOptions{
			BaseURL:     mockAPI.URL(),
			ReloadToken: resp.ReloadToken,
		})

		// Try to reload with conflict hash
		reloadReq := &ReloadRequest{
			BundleHash: "conflict-hash", // This will trigger conflict
			BundleSize: 123456,
		}

		_, err = tokenClient.Reload(ctx, "test-session", reloadReq)
		if err == nil {
			t.Fatal("Expected error for reload conflict, got none")
		}

		// Verify error is DevAPIError with 409 status
		if apiErr, ok := err.(*DevAPIError); ok {
			if apiErr.Code != 1002 {
				t.Errorf("Expected error code 1002 (conflict), got %d", apiErr.Code)
			}
		} else {
			t.Errorf("Expected DevAPIError type, got %T", err)
		}
	})
}

// TestIntegration_StatusErrorHandling tests error handling in status
func TestIntegration_StatusErrorHandling(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	// Create client without token
	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	// Test: Get status without token
	t.Run("Get Status Unauthorized", func(t *testing.T) {
		mockAPI.ResetRequests()

		_, err := client.GetStatus(ctx, "test-session")
		if err == nil {
			t.Fatal("Expected error for unauthorized status request, got none")
		}

		// Verify error is DevAPIError
		if _, ok := err.(*DevAPIError); !ok {
			t.Errorf("Expected DevAPIError type, got %T", err)
		}
	})
}

// TestIntegration_TimeoutHandling tests timeout handling
func TestIntegration_TimeoutHandling(t *testing.T) {
	// Create mock API with slow handler
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	slowServer := httptest.NewServer(slowHandler)
	defer slowServer.Close()

	// Create client with short timeout
	client := NewClient(ClientOptions{
		BaseURL: slowServer.URL,
		Timeout: 50 * time.Millisecond, // Shorter than handler delay
	})

	ctx := context.Background()

	// Test: Request times out
	t.Run("Request Timeout", func(t *testing.T) {
		req := &RegisterRequest{
			PluginID:  "test-plugin",
			Version:   "0.1.0",
			EntryPath: "/path/to/test",
		}

		_, err := client.Register(ctx, req)
		if err == nil {
			t.Fatal("Expected timeout error, got none")
		}

		// Verify it's a timeout error
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})
}

// TestIntegration_Concurrency tests concurrent requests
func TestIntegration_Concurrency(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	// First register to get token
	registerReq := &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "0.1.0",
		EntryPath: "/path/to/test",
	}

	resp, err := client.Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Create client with token
	tokenClient := NewClient(ClientOptions{
		BaseURL:     mockAPI.URL(),
		ReloadToken: resp.ReloadToken,
	})

	// Test: Multiple concurrent reloads
	t.Run("Concurrent Reloads", func(t *testing.T) {
		mockAPI.ResetRequests()

		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()

				reloadReq := &ReloadRequest{
					BundleHash: "concurrent-test-hash",
					BundleSize: 123456,
				}

				_, err := tokenClient.Reload(ctx, "test-session", reloadReq)
				if err != nil {
					t.Errorf("Concurrent reload failed: %v", err)
				}
			}()
		}

		// Wait for all goroutines to complete
		timeout := time.After(5 * time.Second)
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Continue
			case <-timeout:
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		// All requests should have been made
		requests := mockAPI.GetRequests()
		if len(requests) < numGoroutines {
			t.Errorf("Expected at least %d reload requests, got %d", numGoroutines, len(requests))
		}
	})
}

// TestIntegration_Idempotency tests that retry logic works correctly
func TestIntegration_Idempotency(t *testing.T) {
	mockAPI := NewMockDevAPI()
	defer mockAPI.Close()

	client := NewClient(ClientOptions{
		BaseURL: mockAPI.URL(),
	})

	ctx := context.Background()

	// First register to get token
	registerReq := &RegisterRequest{
		PluginID:  "test-plugin",
		Version:   "0.1.0",
		EntryPath: "/path/to/test",
	}

	resp, err := client.Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Create client with token
	tokenClient := NewClient(ClientOptions{
		BaseURL:     mockAPI.URL(),
		ReloadToken: resp.ReloadToken,
	})

	// Test: Multiple identical reloads (idempotency)
	t.Run("Idempotent Reloads", func(t *testing.T) {
		mockAPI.ResetRequests()

		reloadReq := &ReloadRequest{
			BundleHash: "idempotent-test-hash",
			BundleSize: 123456,
		}

		// Send the same reload request multiple times
		for i := 0; i < 3; i++ {
			_, err := tokenClient.Reload(ctx, "test-session", reloadReq)
			if err != nil {
				t.Fatalf("Reload attempt %d failed: %v", i+1, err)
			}
		}

		// Verify all requests were made
		requests := mockAPI.GetRequests()
		if len(requests) < 3 {
			t.Errorf("Expected 3 reload requests, got %d", len(requests))
		}
	})
}
