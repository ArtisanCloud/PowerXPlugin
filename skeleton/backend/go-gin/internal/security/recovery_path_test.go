package security

import (
	"path/filepath"
	"testing"
)

func TestHostRootManifestDoesNotDependOnPinnedExecutable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("POWERX_PLUGIN_ROOT", root)
	t.Setenv("POWERX_PLUGIN_MANIFEST_PATH", "")
	t.Setenv("POWERX_PLUGIN_MANIFEST", "")
	if got := DefaultManifestPath(); got != filepath.Join(root, "plugin.yaml") {
		t.Fatalf("wrong host manifest: %s", got)
	}
	explicit := filepath.Join(root, "explicit.yaml")
	t.Setenv("POWERX_PLUGIN_MANIFEST_PATH", explicit)
	if DefaultManifestPath() != explicit {
		t.Fatal("explicit manifest must remain first")
	}
}
