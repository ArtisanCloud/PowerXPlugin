package gateway

import (
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
		ToolToken:          "token",
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
		ToolToken:          "token",
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

func writeContractDigest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "capability-contracts.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write digest: %v", err)
	}
	return path
}
