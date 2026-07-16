package config

import "testing"

func TestKnowledgeConfigDefaults(t *testing.T) {
	t.Setenv("POWERX_PROXY", "0")
	cfg := getDefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.Knowledge == nil || cfg.Knowledge.Mode != "local" || !cfg.Knowledge.RequireTenant {
		t.Fatalf("unexpected knowledge defaults: %+v", cfg.Knowledge)
	}
}

func TestKnowledgeConfigDefaultsDelegatedInProxyModeWithLocalIAM(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	cfg := getDefaultConfig()
	cfg.Context.ProviderMode = "local"
	cfg.Knowledge = nil
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.Knowledge == nil || cfg.Knowledge.Mode != "delegated" {
		t.Fatalf("expected delegated knowledge in proxy mode, got %+v", cfg.Knowledge)
	}
}

func TestKnowledgeConfigEnvOverride(t *testing.T) {
	t.Setenv("POWERX_KNOWLEDGE_MODE", "mock")
	cfg := getDefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.Knowledge == nil || cfg.Knowledge.Mode != "mock" {
		t.Fatalf("expected env knowledge mode override, got %+v", cfg.Knowledge)
	}
}

func TestKnowledgeConfigRejectsLocalProduction(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Logging.DebugMode = false
	cfg.Context.HMACSecret = "secret"
	cfg.CustomerAuth.BreakGlassLocal = true
	cfg.CustomerAuth.BreakGlassReason = "test"
	cfg.Knowledge = &KnowledgeConfig{Mode: "local"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production local knowledge config to fail")
	}
}

func TestKnowledgeConfigAllowsDelegatedProduction(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Logging.DebugMode = false
	cfg.Context.HMACSecret = "secret"
	cfg.CustomerAuth.BreakGlassLocal = true
	cfg.CustomerAuth.BreakGlassReason = "test"
	cfg.Knowledge = &KnowledgeConfig{Mode: "delegated", DelegateEndpoint: "https://powerx.example.com/api/knowledge", DelegateTimeout: "2s"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("delegated production should pass: %v", err)
	}
}
