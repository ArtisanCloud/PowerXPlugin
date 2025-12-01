package publish

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientSubmitSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/plugins/releases" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("missing auth header, got %s", got)
		}
		var payload SubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.PluginID != "com.test.demo" {
			t.Fatalf("pluginId mismatch: %s", payload.PluginID)
		}
		if payload.Channel != "beta" {
			t.Fatalf("channel mismatch: %s", payload.Channel)
		}
		if payload.TenantUUID != "tenant-uuid" {
			t.Fatalf("tenant uuid missing: %s", payload.TenantUUID)
		}
		if payload.BuildArtifact != "s3://bucket/pkg.tar.gz" {
			t.Fatalf("artifact mismatch: %s", payload.BuildArtifact)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"publishId": "PUB-123",
				"reviewUrl": "https://powerx/publish/PUB-123",
				"status":    "pending",
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(Options{
		BaseURL:  server.URL,
		APIToken: "test-token",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	resp, err := client.Submit(context.Background(), &SubmitRequest{
		TenantUUID:    "tenant-uuid",
		PluginID:      "com.test.demo",
		Version:       "0.1.0",
		Channel:       "beta",
		ReleaseNotes:  "demo",
		BuildArtifact: "s3://bucket/pkg.tar.gz",
		CLIVersion:    "test",
	})
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}
	if resp.PublishID != "PUB-123" {
		t.Fatalf("publish id mismatch: %s", resp.PublishID)
	}
	if resp.Status != "pending" {
		t.Fatalf("status mismatch: %s", resp.Status)
	}
	if resp.ReviewURL == "" {
		t.Fatal("expected review url")
	}
}

func TestClientSubmitEnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    409,
			"message": "duplicate",
		})
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, err = client.Submit(context.Background(), &SubmitRequest{
		TenantUUID:    "tenant-uuid",
		PluginID:      "com.test.demo",
		Version:       "0.1.0",
		Channel:       "dev",
		BuildArtifact: "s3://bucket/dev.tar.gz",
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestClientSubmitHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, err = client.Submit(context.Background(), &SubmitRequest{
		TenantUUID:    "tenant-uuid",
		PluginID:      "com.test.demo",
		Version:       "0.1.0",
		Channel:       "dev",
		BuildArtifact: "s3://bucket/pkg.tar.gz",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}
