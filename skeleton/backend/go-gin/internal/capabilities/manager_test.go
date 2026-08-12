package capabilities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCatalogFilePathDoesNotFallbackToSkeletonCatalog(t *testing.T) {
	previousManifest := os.Getenv("POWERX_PLUGIN_MANIFEST")
	t.Cleanup(func() {
		_ = os.Setenv("POWERX_PLUGIN_MANIFEST", previousManifest)
	})
	_ = os.Unsetenv("POWERX_PLUGIN_MANIFEST")

	root := t.TempDir()
	installed := filepath.Join(root, "installed", "com.powerx.plugins.base")
	if err := os.MkdirAll(installed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "skeleton", "capabilities"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skeleton", "capabilities", "catalog.json"), []byte(`{"entries":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	previousCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousCWD)
	})
	if err := os.Chdir(installed); err != nil {
		t.Fatal(err)
	}

	got, source := resolveCatalogFilePath("")
	want := filepath.Join(installed, defaultCatalogPath)
	if normalizeDarwinPath(got) != normalizeDarwinPath(want) {
		t.Fatalf("catalog path mismatch, got=%s want=%s source=%s", got, want, source)
	}
	if source != "" {
		t.Fatalf("unexpected fallback source: %s", source)
	}
}

func TestResolveCatalogFilePathUsesManifestBundleCatalog(t *testing.T) {
	previousManifest := os.Getenv("POWERX_PLUGIN_MANIFEST")
	t.Cleanup(func() {
		_ = os.Setenv("POWERX_PLUGIN_MANIFEST", previousManifest)
	})

	root := t.TempDir()
	installed := filepath.Join(root, "installed", "com.powerx.plugins.base")
	if err := os.MkdirAll(filepath.Join(installed, "capabilities"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(installed, "plugin.yaml")
	if err := os.WriteFile(manifestPath, []byte("id: com.powerx.plugins.base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(installed, "capabilities", "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{"entries":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Setenv("POWERX_PLUGIN_MANIFEST", manifestPath)

	previousCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousCWD)
	})
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}

	got, source := resolveCatalogFilePath("")
	if normalizeDarwinPath(got) != normalizeDarwinPath(catalogPath) {
		t.Fatalf("catalog path mismatch, got=%s want=%s source=%s", got, catalogPath, source)
	}
	if source != "manifest/cwd fallback" {
		t.Fatalf("fallback source mismatch, got=%s", source)
	}
}

func TestLoadCatalogFromManifestMergesCapabilitiesCatalog(t *testing.T) {
	previousManifest := os.Getenv("POWERX_PLUGIN_MANIFEST")
	t.Cleanup(func() {
		_ = os.Setenv("POWERX_PLUGIN_MANIFEST", previousManifest)
	})

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "plugin.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte(`
id: com.powerx.plugins.base
version: 0.1.4
catalogs:
  capabilities: ./plugin.d/capabilities.yaml
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.d", "capabilities.yaml"), []byte(`
capabilities:
  provides:
    - id: com.powerx.plugins.base.template.list
      version: 1.0.0
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Setenv("POWERX_PLUGIN_MANIFEST", filepath.Join(root, "plugin.yaml"))

	snapshot, err := loadCatalogFromManifest(nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PluginID != "com.powerx.plugins.base" {
		t.Fatalf("unexpected plugin id: %s", snapshot.PluginID)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("expected one merged capability entry, got %d", len(snapshot.Entries))
	}
	if snapshot.Entries[0].ID != "com.powerx.plugins.base.template.list" {
		t.Fatalf("unexpected capability id: %s", snapshot.Entries[0].ID)
	}
}

func normalizeDarwinPath(value string) string {
	clean := filepath.Clean(value)
	if strings.HasPrefix(clean, "/private/var/") {
		return strings.TrimPrefix(clean, "/private")
	}
	return clean
}
