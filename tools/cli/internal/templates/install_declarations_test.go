package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedPluginInstallIdentity(t *testing.T) {
	for _, id := range []string{"com.powerx.plugins.install-alpha", "com.powerx.plugins.install-beta"} {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			_, err := RenderAll(dir, Data{
				PluginID: id, PluginName: "install_fixture", PluginSlug: "install-fixture", PluginDBName: "install_fixture",
				Version: "0.1.0", GoVersion: "1.24", BackendModulePath: "example.com/install/backend",
				BackendType: BackendGoGin, FrontendType: FrontendNuxt, AppFrontendType: FrontendNuxt,
				FrameworkVersion: "v0.0.28", FrameworkAdminRef: "0.0.13", FrameworkClientRef: "0.0.14",
			}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{
				"plugin.yaml", "skills/template/SKILL.md", "contracts/capabilities/com.powerx.plugins.base.template.create.yaml",
				"backend/cmd/database/seed/seed.go", "backend/internal/services/integration/capability_invoker.go",
			} {
				name = strings.ReplaceAll(name, "com.powerx.plugins.base", id)
				raw, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(raw), "com.powerx.plugins.base") || !strings.Contains(string(raw), id) {
					t.Fatal("GENERATED_INSTALL_IDENTITY_MISMATCH", name)
				}
			}
		})
	}
}
