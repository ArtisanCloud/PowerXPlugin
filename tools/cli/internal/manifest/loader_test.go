package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	content := `
id: com.example.demo
version: 1.2.3
backend:
  entry: backend/bin/plugin
`
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if m.ID != "com.example.demo" || m.Version != "1.2.3" {
		t.Fatalf("unexpected manifest: %+v", m)
	}
	if m.Backend.Entry != "backend/bin/plugin" {
		t.Fatalf("backend entry mismatch: %+v", m.Backend)
	}
}

func TestLoadManifestMissingFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte("name: missing\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatalf("expected error for missing id/version")
	}
}
