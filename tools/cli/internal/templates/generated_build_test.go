package templates

import (
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
	_, err := RenderAll(dir, Data{PluginID: "com.powerx.generated-test", PluginName: "Generated Test", PluginSlug: "generated-test", PluginDBName: "generated_test", Version: "0.1.0", GoVersion: "1.24", BackendModulePath: "example.com/generated/backend", BackendType: BackendGoGin, FrontendType: FrontendNuxt, AppFrontendType: FrontendNuxt, BackendPort: 8078, FrontendPort: 3131, FrameworkVersion: "v0.0.24", FrameworkAdminRef: "0.0.11", FrameworkClientRef: "0.0.12"}, Options{})
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
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
	}
	run("test", "-mod=mod", "./cmd/plugin", "./internal/bootstrap", "./internal/services/iam/adapters/...", "./internal/transport/http/middleware", "./internal/transport/http/wsbus", "./internal/services/integration", "./internal/services/runtimeexample", "./internal/transport/http/admin/host_contract", "-count=1")
}
