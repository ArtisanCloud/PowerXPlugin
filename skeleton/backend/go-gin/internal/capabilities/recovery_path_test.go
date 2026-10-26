package capabilities

import (
	"path/filepath"
	"testing"
)

func TestHostRootCapabilityManifestUsesRestoredPackage(t *testing.T) {
	root := t.TempDir()
	t.Setenv("POWERX_PLUGIN_ROOT", root)
	t.Setenv("POWERX_PLUGIN_MANIFEST", "")
	if got := resolveManifestPath(); got != filepath.Join(root, "plugin.yaml") {
		t.Fatalf("wrong host manifest: %s", got)
	}
}
