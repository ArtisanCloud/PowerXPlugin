package main

import (
	"strings"
	"testing"

	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestValidateDatabaseRuntimeRejectsDelegatedPublicSchema(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_PLUGIN_DB_SCHEMA", "public")
	t.Setenv("POWERX_DB_SCHEMA", "public")

	cfg := &config.Config{
		Database: &config.DatabaseConfig{Schema: "public"},
		Context:  &config.ContextConfig{ProviderMode: "delegated"},
	}
	resolver := pluginbootstrap.NewProviderResolver(cfg)
	if err := resolver.Err(); err != nil {
		t.Fatalf("NewProviderResolver() error = %v", err)
	}

	err := validateDatabaseRuntime(cfg, resolver)
	if err == nil || !strings.Contains(err.Error(), "protected schema") {
		t.Fatalf("validateDatabaseRuntime() err = %v, want protected schema", err)
	}
}

func TestValidateDatabaseRuntimeRejectsDelegatedSchemaMismatch(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_PLUGIN_DB_SCHEMA", "px_com_powerx_plugin_demo")
	t.Setenv("POWERX_DB_SCHEMA", "public")

	cfg := &config.Config{
		Database: &config.DatabaseConfig{Schema: "public"},
		Context:  &config.ContextConfig{ProviderMode: "delegated"},
	}
	resolver := pluginbootstrap.NewProviderResolver(cfg)
	if err := resolver.Err(); err != nil {
		t.Fatalf("NewProviderResolver() error = %v", err)
	}

	err := validateDatabaseRuntime(cfg, resolver)
	if err == nil || !strings.Contains(err.Error(), "schema mismatch") {
		t.Fatalf("validateDatabaseRuntime() err = %v, want schema mismatch", err)
	}
}

func TestValidateDatabaseRuntimeAllowsDelegatedPluginSchema(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_PLUGIN_DB_SCHEMA", "px_com_powerx_plugin_demo")
	t.Setenv("POWERX_DB_SCHEMA", "px_com_powerx_plugin_demo")

	cfg := &config.Config{
		Database: &config.DatabaseConfig{Schema: "px_com_powerx_plugin_demo"},
		Context:  &config.ContextConfig{ProviderMode: "delegated"},
	}
	resolver := pluginbootstrap.NewProviderResolver(cfg)
	if err := resolver.Err(); err != nil {
		t.Fatalf("NewProviderResolver() error = %v", err)
	}

	if err := validateDatabaseRuntime(cfg, resolver); err != nil {
		t.Fatalf("validateDatabaseRuntime() error = %v", err)
	}
}

func TestValidateDatabaseRuntimeAllowsLocalPublicSchema(t *testing.T) {
	t.Setenv("POWERX_PROVIDER_MODE", "local")
	t.Setenv("POWERX_PROXY", "0")

	cfg := &config.Config{
		Database: &config.DatabaseConfig{Schema: "public"},
		Context:  &config.ContextConfig{ProviderMode: "local"},
	}
	resolver := pluginbootstrap.NewProviderResolver(cfg)
	if err := resolver.Err(); err != nil {
		t.Fatalf("NewProviderResolver() error = %v", err)
	}

	if err := validateDatabaseRuntime(cfg, resolver); err != nil {
		t.Fatalf("validateDatabaseRuntime() error = %v", err)
	}
}
