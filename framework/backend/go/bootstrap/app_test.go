package bootstrap

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
)

func TestNewAppFromEnvDefaults(t *testing.T) {
	t.Setenv("POWERX_LISTEN", "")
	t.Setenv("POWERX_ENV", "")
	t.Setenv("STANDALONE", "")

	app := NewAppFromEnv()
	if got, want := app.Config.Listen, ":8078"; got != want {
		t.Fatalf("Listen = %q, want %q", got, want)
	}
	if got, want := app.Config.Env, "development"; got != want {
		t.Fatalf("Env = %q, want %q", got, want)
	}
	if !app.Config.Standalone {
		t.Fatalf("Standalone = false, want true")
	}
}

func TestNewAppFromEnvWithOverrides(t *testing.T) {
	t.Setenv("POWERX_LISTEN", ":9090")
	t.Setenv("POWERX_ENV", "production")
	t.Setenv("STANDALONE", "false")

	app := NewAppFromEnv()
	if got, want := app.Config.Listen, ":9090"; got != want {
		t.Fatalf("Listen = %q, want %q", got, want)
	}
	if got, want := app.Config.Env, "production"; got != want {
		t.Fatalf("Env = %q, want %q", got, want)
	}
	if app.Config.Standalone {
		t.Fatalf("Standalone = true, want false")
	}
}

func TestWithStandaloneDefaults(t *testing.T) {
	cfg := &Config{}
	WithStandaloneDefaults()(cfg)
	if !cfg.Standalone {
		t.Fatalf("Standalone = false, want true")
	}
	if cfg.Listen != ":8078" {
		t.Fatalf("Listen = %q, want :8078", cfg.Listen)
	}
	if cfg.Env != "development" {
		t.Fatalf("Env = %q, want development", cfg.Env)
	}
}

func TestAppRunWithoutServer(t *testing.T) {
	app := NewApp(nil)
	err := app.Run()
	if err == nil {
		t.Fatalf("Run() = nil, want error")
	}
	if !strings.Contains(err.Error(), "http server not attached") {
		t.Fatalf("Run() error = %v, want contains %q", err, "http server not attached")
	}
}

func TestRegisterManifestCopiesValue(t *testing.T) {
	app := NewApp(nil)
	m := manifest.Plugin{ID: "com.powerx.demo", Name: "Demo"}
	app.RegisterManifest(m)

	got := app.Manifest()
	if got == nil {
		t.Fatalf("Manifest() = nil, want value")
	}
	if got.ID != "com.powerx.demo" {
		t.Fatalf("Manifest().ID = %q, want com.powerx.demo", got.ID)
	}

	m.ID = "mutated"
	if got.ID != "com.powerx.demo" {
		t.Fatalf("Manifest mutated after source change, got %q", got.ID)
	}
}

func TestShutdownWithoutServer(t *testing.T) {
	app := NewApp(nil)
	if err := app.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}

func TestParseBoolEnv(t *testing.T) {
	t.Setenv("TEST_BOOL", "true")
	if !parseBoolEnv("TEST_BOOL", false) {
		t.Fatalf("parseBoolEnv true value expected true")
	}
	t.Setenv("TEST_BOOL", "invalid")
	if !parseBoolEnv("TEST_BOOL", true) {
		t.Fatalf("parseBoolEnv fallback expected true")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_VALUE", "")
	if got := getEnvOrDefault("TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("getEnvOrDefault returned %q, want fallback", got)
	}
	t.Setenv("TEST_VALUE", " value ")
	if got := getEnvOrDefault("TEST_VALUE", "fallback"); got != "value" {
		t.Fatalf("getEnvOrDefault returned %q, want value", got)
	}
}

func TestWithRuntimeDefaultsInjectsTenantMirrorFields(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	logger := withRuntimeDefaults(base, &Config{
		Gateway: GatewayConfig{TenantID: "tenant-a"},
	})
	logger.Info("probe")

	out := buf.String()
	if !strings.Contains(out, `"tenant_uuid":"tenant-a"`) {
		t.Fatalf("expected tenant_uuid in output, got %s", out)
	}
	if !strings.Contains(out, `"tenant_key":"tenant-a"`) {
		t.Fatalf("expected tenant_key mirror in output, got %s", out)
	}
	if !strings.Contains(out, `"subscriber_id":"bootstrap.app"`) {
		t.Fatalf("expected subscriber_id in output, got %s", out)
	}
	if !strings.Contains(out, `"component":"bootstrap.app"`) {
		t.Fatalf("expected component in output, got %s", out)
	}
}
