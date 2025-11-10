package mtls

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("PX_MTLS_CERT_PATH", "~/certs/client.crt")
	t.Setenv("PX_MTLS_KEY_PATH", "~/certs/client.key")
	t.Setenv("PX_MTLS_CA_PATH", "~/certs/ca.crt")
	t.Setenv("PX_MTLS_SERVER_NAME", "dev.powerx.local")
	t.Setenv("PX_MTLS_SKIP_VERIFY", "true")
	t.Setenv("PX_MTLS_AUTO_ROTATE", "false")
	t.Setenv("PX_MTLS_ROTATION_CHECK", "30s")

	cfg, ok, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected env loader to report presence")
	}
	if cfg.ServerName != "dev.powerx.local" {
		t.Fatalf("unexpected server name: %s", cfg.ServerName)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatalf("expected skip verify from env")
	}
	if cfg.AutoRotate {
		t.Fatalf("expected auto rotate disabled from env")
	}
	if cfg.RotationCheck != 30*time.Second {
		t.Fatalf("unexpected rotation check duration: %s", cfg.RotationCheck)
	}
}

func TestConfigFromPaths(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "client.crt")
	key := filepath.Join(dir, "client.key")
	ca := filepath.Join(dir, "ca.crt")

	os.WriteFile(cert, []byte("cert"), 0600)
	os.WriteFile(key, []byte("key"), 0600)
	os.WriteFile(ca, []byte("ca"), 0600)

	cfg, err := ConfigFromPaths(cert, key, ca)
	if err != nil {
		t.Fatalf("ConfigFromPaths returned error: %v", err)
	}
	if cfg.CertPath != cert {
		t.Fatalf("unexpected cert path: %s", cfg.CertPath)
	}
}
