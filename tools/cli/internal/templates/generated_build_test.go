package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGeneratedBackendBuild(t *testing.T) {
	if os.Getenv("PX_TEST_GENERATED_BUILD") != "1" {
		t.Skip("PX_TEST_GENERATED_BUILD=1")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	_, err = RenderAll(dir, Data{PluginID: "com.powerx.generated-test", PluginName: "Generated Test", PluginSlug: "generated-test", PluginDBName: "generated_test", Version: "0.1.0", GoVersion: "1.24", BackendModulePath: "example.com/generated/backend", BackendType: BackendGoGin, FrontendType: FrontendNuxt, AppFrontendType: FrontendNuxt, BackendPort: 8078, FrontendPort: 3131, FrameworkVersion: "v0.0.22", FrameworkAdminRef: "0.0.10", FrameworkClientRef: "0.0.11"}, Options{})
	if err != nil {
		t.Fatal(err)
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
	run("mod", "edit", "-replace=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go="+filepath.Join(root, "framework/backend/go"))
	run("test", "-mod=mod", "./cmd/plugin", "./internal/services/runtimeexample", "./internal/services/integration", "./internal/transport/http/admin/host_contract", "-count=1")
}
