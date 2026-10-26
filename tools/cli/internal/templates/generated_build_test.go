package templates

import (
	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedBackendBuild(t *testing.T) {
	if os.Getenv("PX_TEST_GENERATED_BUILD") != "1" {
		t.Skip("PX_TEST_GENERATED_BUILD=1")
	}
	dir := t.TempDir()
	bindings, err := fwmanifest.LoadNotificationPresentations("../../../../docs/contracts/fixtures/notification-presentations.v2.yaml")
	if err != nil {
		t.Fatal(err)
	}
	version, proxy := "v0.0.27-dev.20261026.1", ""
	if os.Getenv("PX_TEST_FRAMEWORK_SNAPSHOT") == "1" {
		proxy, version = frameworkSnapshot(t)
	}
	_, err = RenderAll(dir, Data{NotificationPresentations: bindings, PluginID: "com.powerx.generated-test", PluginName: "Generated Test", PluginSlug: "generated-test", PluginDBName: "generated_test", Version: "0.1.0", GoVersion: "1.24", BackendModulePath: "example.com/generated/backend", BackendType: BackendGoGin, FrontendType: FrontendNuxt, AppFrontendType: FrontendNuxt, BackendPort: 8078, FrontendPort: 3131, FrameworkVersion: version, FrameworkAdminRef: "0.0.13-dev.20261026.1", FrameworkClientRef: "0.0.14-dev.20261026.1"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "backend", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifest), "replace github.com/ArtisanCloud/PowerXPlugin/framework/backend/go") {
		t.Fatal("FRAMEWORK_LOCAL_REPLACE_NOT_ALLOWED")
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("go", args...)
		cmd.Dir = filepath.Join(dir, "backend")
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if proxy != "" {
			upstream := os.Getenv("GOPROXY")
			if upstream == "" {
				upstream = "https://proxy.golang.org,direct"
			}
			cmd.Env = append(cmd.Env, "GOPROXY=file://"+filepath.ToSlash(proxy)+","+upstream, "GONOSUMDB="+frameworkModule)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
	}
	run("run", "-mod=mod", "./cmd/manifestcheck", "--plugin", "../plugin.yaml", "--plugin-only")
	run("test", "-mod=mod", "./cmd/plugin", "./internal/bootstrap", "./internal/services/iam/adapters/...", "./internal/transport/http/middleware", "./internal/transport/http/wsbus", "./internal/services/integration", "./internal/services/runtimeexample", "./internal/services/customer", "./internal/transport/http/admin/customer", "./internal/transport/http/admin/ai_settings", "./internal/transport/http/admin/host_contract", "./internal/services/notifications", "./internal/config", "./cmd/manifestcheck", "-count=1")
}
