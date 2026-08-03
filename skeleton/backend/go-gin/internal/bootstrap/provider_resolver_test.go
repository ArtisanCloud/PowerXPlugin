package bootstrap

import (
	"testing"

	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestProviderResolver_ConfigPriority(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "0")

	cfg := &config.Config{
		Context: &config.ContextConfig{ProviderMode: "local"},
	}
	resolver := NewProviderResolver(cfg)

	if resolver.Err() == nil {
		t.Fatalf("expected conflict error, got nil")
	}
	if !resolver.IsConflict() {
		t.Fatalf("expected conflict flag on resolver")
	}
}

func TestProviderResolver_ConfigDelegated(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "")
	t.Setenv("POWERX_PROXY", "0")

	cfg := &config.Config{
		Context: &config.ContextConfig{ProviderMode: "delegated"},
	}
	resolver := NewProviderResolver(cfg)

	if resolver.Err() != nil {
		t.Fatalf("unexpected error: %v", resolver.Err())
	}
	if got := resolver.Mode(); got != fwprovider.ModeDelegated {
		t.Fatalf("mode mismatch, got=%s want=%s", got, fwprovider.ModeDelegated)
	}
	if got := resolver.Source(); got != "config" {
		t.Fatalf("source mismatch, got=%s want=config", got)
	}
}

func TestProviderResolver_ProxyDoesNotSelectProviderMode(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "")
	t.Setenv("POWERX_PROXY", "1")

	resolver := NewProviderResolver(&config.Config{})
	if resolver.Err() != nil {
		t.Fatalf("unexpected error: %v", resolver.Err())
	}
	if got := resolver.Mode(); got != fwprovider.ModeLocal {
		t.Fatalf("mode mismatch, got=%s want=%s", got, fwprovider.ModeLocal)
	}
	if got := resolver.Source(); got != "default" {
		t.Fatalf("source mismatch, got=%s want=default", got)
	}
}

func TestShouldForceHostLoggingRequiresDelegatedProxyMode(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")

	localCfg := &config.Config{Context: &config.ContextConfig{ProviderMode: "local"}}
	if shouldForceHostLogging(localCfg) {
		t.Fatal("local + proxy should preserve configured logging output")
	}

	delegatedCfg := &config.Config{Context: &config.ContextConfig{ProviderMode: "delegated"}}
	if !shouldForceHostLogging(delegatedCfg) {
		t.Fatal("delegated + proxy should force host logging output")
	}

	t.Setenv("POWERX_PROXY", "0")
	if shouldForceHostLogging(delegatedCfg) {
		t.Fatal("non-proxy mode should not force host logging output")
	}
}
