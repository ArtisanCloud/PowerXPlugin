package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDevDefaultsLoadsConfigFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cfgDir := filepath.Join(tmpHome, ".px-plugin")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	configJSON := `{
  "dev": {
    "entryPath": "/repo/plugins/sample",
    "tenant": "cfg-tenant",
    "ignore": ["dist/**"]
  },
  "devApi": {
    "baseUrl": "https://devapi.example.com",
    "certPath": "/certs/client.crt",
    "keyPath": "/certs/client.key",
    "caCertPath": "/certs/ca.crt"
  },
  "security": {
    "insecureSkipVerify": true
  },
  "performance": {
    "memoryLimit": 104857600,
    "cpuThreshold": 12,
    "maxConcurrency": 6
  },
  "watch": {
    "maxFiles": 12345
  }
}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := &DevOptions{
		Ignore: []string{"node_modules/**"},
	}
	applyDevDefaults(opts)

	if opts.Entry != "/repo/plugins/sample" {
		t.Fatalf("expected entry from config, got %s", opts.Entry)
	}
	if opts.Tenant != "cfg-tenant" {
		t.Fatalf("expected tenant from config, got %s", opts.Tenant)
	}
	if opts.DevAPI != "https://devapi.example.com" {
		t.Fatalf("expected dev api from config, got %s", opts.DevAPI)
	}
	if opts.MTLSCert != "/certs/client.crt" || opts.MTLSKey != "/certs/client.key" || opts.MTLSCA != "/certs/ca.crt" {
		t.Fatalf("expected mTLS paths from config, got %s %s %s", opts.MTLSCert, opts.MTLSKey, opts.MTLSCA)
	}
	if !opts.MTLSSkipVerify {
		t.Fatalf("expected mtls skip verify true")
	}
	if opts.MTLSServerName != "devapi.example.com" {
		t.Fatalf("expected server name from base url, got %s", opts.MTLSServerName)
	}
	if len(opts.Ignore) != 2 || opts.Ignore[0] != "dist/**" || opts.Ignore[1] != "node_modules/**" {
		t.Fatalf("expected ignore merge, got %v", opts.Ignore)
	}
	if opts.MaxMemoryMB != 100 {
		t.Fatalf("expected memory limit 100MB, got %d", opts.MaxMemoryMB)
	}
	if opts.MaxCPUPercent != 12 {
		t.Fatalf("expected cpu threshold 12, got %d", opts.MaxCPUPercent)
	}
	if opts.MaxProcs != 6 {
		t.Fatalf("expected max procs 6, got %d", opts.MaxProcs)
	}
	if opts.MaxWatchFiles != 12345 {
		t.Fatalf("expected max watch files from config, got %d", opts.MaxWatchFiles)
	}
}

func TestApplyDevDefaultsTenantEnvOverride(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("PX_DEV_TENANT", "env-tenant")

	opts := &DevOptions{}
	applyDevDefaults(opts)

	if opts.Tenant != "env-tenant" {
		t.Fatalf("expected tenant from env, got %s", opts.Tenant)
	}
}
