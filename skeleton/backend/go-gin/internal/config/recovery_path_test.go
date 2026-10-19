package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostRootConfigurationIgnoresCoordinatorWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("POWERX_PLUGIN_ROOT", root)
	t.Setenv("CONFIG_PATH", "")
	hostDirectory := filepath.Join(root, "config")
	t.Setenv("POWERX_PLUGIN_CONFIG_DIR", hostDirectory)
	if err := os.MkdirAll(hostDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostDirectory, "host-values.yaml"), []byte("env: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "backend", "etc")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("ai_catalog:\n  directory: ai/providers.d\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := getDefaultConfig()
	got, err := loadYAMLConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir || cfg.AICatalog == nil || cfg.AICatalog.Directory != "ai/providers.d" {
		t.Fatalf("unexpected host configuration: %s", got)
	}
	if err := os.Remove(filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := loadYAMLConfig(getDefaultConfig()); err == nil {
		t.Fatal("missing host file must not read coordinator config")
	}
	t.Setenv("CONFIG_PATH", filepath.Join(root, "explicit.yaml"))
	if got := resolveConfigCandidates()[0]; got != filepath.Join(root, "explicit.yaml") {
		t.Fatal("explicit configuration must remain first")
	}
}
