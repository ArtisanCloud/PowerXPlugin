package capabilities_test

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
)

func TestValidateExecutionDefaultsToSync(t *testing.T) {
	entries := []capabilities.CatalogEntry{{ID: "com.powerx.demo.template.create"}}
	if err := capabilities.ValidateExecution(entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].Execution.Mode != "sync" {
		t.Fatalf("expected execution mode sync, got %s", entries[0].Execution.Mode)
	}
}

func TestValidateExecutionAsyncMissingMetadataFails(t *testing.T) {
	entries := []capabilities.CatalogEntry{{
		ID:        "com.powerx.demo.async",
		Execution: capabilities.ExecutionConfig{Mode: "async"},
	}}
	if err := capabilities.ValidateExecution(entries); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateExecutionAsyncHappyPath(t *testing.T) {
	entries := []capabilities.CatalogEntry{{
		ID: "com.powerx.demo.async",
		Execution: capabilities.ExecutionConfig{
			Mode:           "async",
			CallbackURL:    "https://callback",
			StatusEndpoint: "https://status",
		},
	}}
	if err := capabilities.ValidateExecution(entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].Execution.Mode != "async" {
		t.Fatalf("expected execution mode async, got %s", entries[0].Execution.Mode)
	}
}
