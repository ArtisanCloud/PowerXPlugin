package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/powerx-plugin/cli/internal/config"
)

func TestRunDevAuthSetupCreatesConfigAndCerts(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	srcDir := filepath.Join(tmpHome, ".powerx", "cli")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	mustWrite(t, filepath.Join(srcDir, "client.crt"), []byte("CERT"))
	mustWrite(t, filepath.Join(srcDir, "client.key"), []byte("KEY"))
	mustWrite(t, filepath.Join(srcDir, "ca.crt"), []byte("CA"))

	opts := &DevOptions{
		AuthSetup:      true,
		DevAPI:         "https://dev-api.example/api/v1",
		Tenant:         "demo-tenant",
		Entry:          "/work/plugin",
		MTLSSkipVerify: true,
	}

	if err := runDevAuthSetup(opts); err != nil {
		t.Fatalf("runDevAuthSetup returned error: %v", err)
	}

	destCert := filepath.Join(tmpHome, ".px-plugin", "certs", "client.crt")
	destKey := filepath.Join(tmpHome, ".px-plugin", "certs", "client.key")
	destCA := filepath.Join(tmpHome, ".px-plugin", "certs", "ca.crt")

	assertFileContent(t, destCert, "CERT")
	assertFileContent(t, destKey, "KEY")
	assertFileContent(t, destCA, "CA")

	cfgPath := filepath.Join(tmpHome, ".px-plugin", "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if cfg.DevAPI.BaseURL != opts.DevAPI {
		t.Fatalf("expected dev api %s, got %s", opts.DevAPI, cfg.DevAPI.BaseURL)
	}
	if cfg.Dev.Tenant != opts.Tenant {
		t.Fatalf("expected tenant %s, got %s", opts.Tenant, cfg.Dev.Tenant)
	}
	if cfg.Dev.EntryPath != opts.Entry {
		t.Fatalf("expected entry path %s, got %s", opts.Entry, cfg.Dev.EntryPath)
	}
	if cfg.DevAPI.CertPath != destCert || cfg.DevAPI.KeyPath != destKey || cfg.DevAPI.CACertPath != destCA {
		t.Fatalf("devApi cert paths not updated: %+v", cfg.DevAPI)
	}
	if cfg.Security.CertDir != filepath.Dir(destCert) {
		t.Fatalf("certDir not updated, got %s", cfg.Security.CertDir)
	}
	if !cfg.Security.EnableMTLS || !cfg.DevAPI.EnableMTLS {
		t.Fatalf("expected mTLS to be enabled in config")
	}
	if !cfg.Security.InsecureSkipVerify {
		t.Fatalf("expected insecureSkipVerify to follow flag")
	}
	if !contains(cfg.Dev.Ignore, ".px-plugin/**") {
		t.Fatalf("expected default ignore paths to include .px-plugin/**, got %v", cfg.Dev.Ignore)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	if string(data) != expected {
		t.Fatalf("unexpected content for %s: %q", path, string(data))
	}
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
