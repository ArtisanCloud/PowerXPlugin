package templates

import (
	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	climanifest "github.com/powerx-plugin/cli/internal/manifest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGeneratedNotificationPresentationsRemainValid(t *testing.T) {
	bindings, err := fwmanifest.LoadNotificationPresentations("../../../../docs/contracts/fixtures/notification-presentations.v2.yaml")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	data := Data{PluginID: "com.powerx.notification-test", PluginName: "notification_fixture", PluginSlug: "notification-test", PluginDBName: "notification_test", Version: "0.1.0", GoVersion: "1.24", BackendModulePath: "example.com/notification/backend", BackendType: BackendGoGin, FrontendType: FrontendNuxt, AppFrontendType: FrontendNuxt, FrameworkVersion: "v0.0.26", FrameworkAdminRef: "0.0.12", FrameworkClientRef: "0.0.13", NotificationPresentations: bindings}
	if _, err := RenderAll(dir, data, Options{}); err != nil {
		t.Fatal(err)
	}
	manifest, err := climanifest.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.NotificationPresentations, bindings) {
		t.Fatal("GENERATED_PRESENTATIONS_LOST")
	}
	for _, path := range []string{"internal/transport/http/admin/host_contract/handler.go", "internal/config/config.go", "cmd/plugin/main.go"} {
		raw, err := os.ReadFile(filepath.Join(dir, "backend", path))
		if err != nil || len(raw) == 0 {
			t.Fatal("GENERATED_RUNTIME_MISSING")
		}
	}
	data.NotificationPresentations = append(bindings, bindings[0])
	bad := filepath.Join(t.TempDir(), "invalid")
	if _, err := RenderAll(bad, data, Options{}); err == nil {
		t.Fatal("INVALID_BINDINGS_GENERATED")
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Fatal("INVALID_BINDINGS_MUTATED_OUTPUT")
	}
}
