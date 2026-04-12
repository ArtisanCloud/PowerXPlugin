package bootstrap

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

func TestIAMResolver_ConfigPriority(t *testing.T) {
	t.Setenv("IAM_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "0")

	cfg := &config.Config{
		Context: &config.ContextConfig{IAMMode: "local"},
	}
	resolver := NewIAMResolver(cfg)

	if resolver.Err() == nil {
		t.Fatalf("expected conflict error, got nil")
	}
	if !resolver.IsConflict() {
		t.Fatalf("expected conflict flag on resolver")
	}
}

func TestIAMResolver_ConfigDelegated(t *testing.T) {
	t.Setenv("IAM_MODE", "")
	t.Setenv("POWERX_PROXY", "0")

	cfg := &config.Config{
		Context: &config.ContextConfig{IAMMode: "delegated"},
	}
	resolver := NewIAMResolver(cfg)

	if resolver.Err() != nil {
		t.Fatalf("unexpected error: %v", resolver.Err())
	}
	if got := resolver.Mode(); got != iamservice.IAMModeDelegated {
		t.Fatalf("mode mismatch, got=%s want=%s", got, iamservice.IAMModeDelegated)
	}
	if got := resolver.Source(); got != "config" {
		t.Fatalf("source mismatch, got=%s want=config", got)
	}
}

func TestIAMResolver_ProxyFallbackDelegated(t *testing.T) {
	t.Setenv("IAM_MODE", "")
	t.Setenv("IAMMode", "")
	t.Setenv("POWERX_PROXY", "1")

	resolver := NewIAMResolver(&config.Config{})
	if resolver.Err() != nil {
		t.Fatalf("unexpected error: %v", resolver.Err())
	}
	if got := resolver.Mode(); got != iamservice.IAMModeDelegated {
		t.Fatalf("mode mismatch, got=%s want=%s", got, iamservice.IAMModeDelegated)
	}
	if got := resolver.Source(); got != "env:POWERX_PROXY" {
		t.Fatalf("source mismatch, got=%s want=env:POWERX_PROXY", got)
	}
}
