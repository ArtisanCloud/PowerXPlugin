package templates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedNuxtUsesBridgeV2(t *testing.T) {
	dir := t.TempDir()
	_, err := RenderAll(dir, Data{PluginID: "com.powerx.v2-test", PluginName: "Bridge V2 Test", PluginSlug: "v2-test", PluginDBName: "v2_test", Version: "0.1.0", GoVersion: "1.24", BackendModulePath: "example.com/v2-test/backend", BackendType: BackendGoGin, FrontendType: FrontendNuxt, AppFrontendType: FrontendNuxt, BackendPort: 8078, FrontendPort: 3131, FrameworkVersion: "v0.0.23", FrameworkAdminRef: "0.0.10", FrameworkClientRef: "file:../../framework/frontend/nuxt/framework-client"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "web-admin", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if got := manifest.Dependencies["@artisan-cloud/plugin-framework-client"]; got != "file:../../framework/frontend/nuxt/framework-client" {
		t.Fatalf("framework_client_ref_mismatch: %q", got)
	}
	if got := manifest.Dependencies["@artisan-cloud/plugin-framework-admin"]; got != "0.0.10" {
		t.Fatalf("framework_admin_ref_mismatch: %q", got)
	}
	for filename, marker := range map[string]string{
		"app/composables/useEmbeddedAuth.ts":      "ensureSnapshot",
		"app/composables/useHostBridgeAdapter.ts": "createPowerXRouterNavigator",
		"app/plugins/auth.client.ts":              "setupHostBridgeAdapter",
	} {
		content, err := os.ReadFile(filepath.Join(dir, "web-admin", filepath.FromSlash(filename)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), marker) {
			t.Fatalf("bridge_v2_marker_missing: %s %s", filename, marker)
		}
		if strings.Contains(string(content), "allowedOrigins: [\"*\"]") || strings.Contains(string(content), "applyPowerXHostAuthToken") {
			t.Fatalf("obsolete_auth_receiver_present: %s", filename)
		}
	}
	for filename, marker := range map[string]string{
		"backend/internal/transport/http/middleware/delegated_identity.go": "fwidentity.RefreshDelegatedIdentity",
		"backend/internal/router/router.go":                                "RefreshFrameworkIAMContext",
		"backend/cmd/plugin/main.go":                                       "SetIdentityResolver",
		"backend/internal/transport/http/wsbus/handler.go":                 "WSIdentityLifecycle.Open",
	} {
		content, readErr := os.ReadFile(filepath.Join(dir, filepath.FromSlash(filename)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(content), marker) {
			t.Fatalf("delegated_identity_marker_missing: %s", filename)
		}
	}
}
