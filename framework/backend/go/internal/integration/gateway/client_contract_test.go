package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractStatusLoadsDigest(t *testing.T) {
	t.Parallel()
	digestPath := writeContractDigest(t, `{
  "generatedAt": "2025-12-22T00:00:00Z",
  "manifest": {"version": "0.8.0"},
  "digest": {"bundlesHash": "hash-local"}
}`)
	client, err := NewClient(Config{
		BaseURL:            "https://example.com",
		AuthScheme:         "bearer",
		BearerToken:        "token",
		TenantUUID:         "tenant",
		ContractDigestPath: digestPath,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	status := client.ContractStatus()
	if status == nil {
		t.Fatalf("expected contract status")
	}
	if status.CurrentHash != "hash-local" {
		t.Fatalf("unexpected hash %s", status.CurrentHash)
	}
	if status.Outdated {
		t.Fatalf("expected status up-to-date")
	}
	if status.DigestSource != digestPath {
		t.Fatalf("expected digest source %s, got %s", digestPath, status.DigestSource)
	}
	if status.GeneratedAt == "" {
		t.Fatalf("expected generatedAt to be populated")
	}
}

func TestContractStatusDetectsMismatch(t *testing.T) {
	t.Parallel()
	digestPath := writeContractDigest(t, `{
  "generatedAt": "2025-12-22T00:00:00Z",
  "manifest": {"version": "0.8.0"},
  "digest": {"bundlesHash": "hash-local"}
}`)
	client, err := NewClient(Config{
		BaseURL:            "https://example.com",
		AuthScheme:         "bearer",
		BearerToken:        "token",
		TenantUUID:         "tenant",
		ContractVersion:    "hash-remote",
		ContractDigestPath: digestPath,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	status := client.ContractStatus()
	if status == nil {
		t.Fatalf("expected contract status")
	}
	if !status.Outdated {
		t.Fatalf("expected mismatch to mark status outdated")
	}
	if !strings.Contains(status.Message, "hash-remote") {
		t.Fatalf("expected warning message to mention expected version, got %s", status.Message)
	}
}

func TestInvokeMapsArrayDataToItemsAndRecords(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenant/invocations" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"traceId": "trace-array",
			"status":  "ok",
			"data": []any{
				map[string]any{"jobId": "job-1", "status": "completed"},
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{
		BaseURL:     server.URL,
		AuthScheme:  "bearer",
		BearerToken: "token",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := client.Invoke(context.Background(), InvokeRequest{
		CapabilityID:      "com.corex.rest.admin.gin.get_api_v1_admin_knowledge_spaces_spaceid_ingestion_jobs",
		Action:            "ListIngestionJobs",
		PreferredProtocol: "rest",
		Payload: map[string]any{
			"method":   http.MethodGet,
			"endpoint": "/api/v1/admin/knowledge-spaces/space-1/ingestion-jobs",
		},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	items, ok := resp.Data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected array data to be exposed as items, data=%+v raw=%s", resp.Data, string(resp.RawData))
	}
	records, ok := resp.Data["records"].([]any)
	if !ok || len(records) != 1 {
		t.Fatalf("expected array data to be exposed as records, data=%+v raw=%s", resp.Data, string(resp.RawData))
	}
	if !strings.Contains(string(resp.RawData), `"jobId":"job-1"`) {
		t.Fatalf("expected raw data to be preserved, raw=%s", string(resp.RawData))
	}
}

func writeContractDigest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "capability-contracts.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write digest: %v", err)
	}
	return path
}
