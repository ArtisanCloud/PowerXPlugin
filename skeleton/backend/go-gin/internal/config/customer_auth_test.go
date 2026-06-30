package config

import "testing"

func TestCustomerAuthProductionBlocksLocalWithoutBreakGlass(t *testing.T) {
	cfg := minimalCustomerAuthConfig()
	cfg.Logging.DebugMode = false
	cfg.CustomerAuth.Mode = "local"
	cfg.CustomerAuth.JWTSecret = "secret"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production local customer auth to be blocked")
	}
}

func TestCustomerAuthProductionAllowsLocalWithBreakGlassReason(t *testing.T) {
	cfg := minimalCustomerAuthConfig()
	cfg.Logging.DebugMode = false
	cfg.CustomerAuth.Mode = "local"
	cfg.CustomerAuth.JWTSecret = "secret"
	cfg.CustomerAuth.BreakGlassLocal = true
	cfg.CustomerAuth.BreakGlassReason = "migration window"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected break-glass local customer auth to pass: %v", err)
	}
}

func minimalCustomerAuthConfig() *Config {
	cfg := getDefaultConfig()
	cfg.Database = &DatabaseConfig{Driver: "sqlite", DSN: ":memory:"}
	cfg.Context = &ContextConfig{HMACSecret: "secret"}
	cfg.Logging = &LoggingConfig{DebugMode: true, Level: "info", Format: "json", Output: "stdout"}
	cfg.CustomerAuth = &CustomerAuthConfig{
		Mode:            "local",
		DelegateTimeout: "3s",
		JWTSecret:       "secret",
	}
	if cfg.Security == nil {
		cfg.Security = &SecurityConfig{}
	}
	return cfg
}
