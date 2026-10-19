package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Build the actual CLI, generate a fresh plugin, then run behavior tests from
// that generated source against an immutable packaged Framework snapshot.
func TestIAMBaselineGeneratedPlugin(t *testing.T) {
	if os.Getenv("PX_TEST_GENERATED_BUILD") != "1" {
		t.Skip("PX_TEST_GENERATED_BUILD=1")
	}
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	binary := filepath.Join(directory, "px-plugin")
	build := exec.Command("go", "build", "-o", binary, "./cmd/px-plugin")
	build.Dir = filepath.Join(repo, "tools/cli")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	destination := filepath.Join(directory, "generated")
	generate := exec.Command(binary, "init", "com.powerx.baseline-generated", "--directory", destination, "--backend", "go-gin", "--admin", "nuxt", "--module", "github.com/powerx-fixture/baseline-generated", "--git-init=false", "--init-config=false")
	generate.Stdin = strings.NewReader("")
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	backend := filepath.Join(destination, "backend")
	manifest, err := os.ReadFile(filepath.Join(backend, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(manifest))
	if len(fields) < 2 || fields[0] != "module" {
		t.Fatal("GENERATED_MODULE_INVALID")
	}
	module := fields[1]
	for _, path := range []string{"internal/bootstrap/iam_resolver_test.go", "internal/bootstrap/authorization_baseline_test.go", "internal/transport/http/wsbus/final_send_test.go", "internal/transport/http/wsbus/delegated_identity_test.go"} {
		source, err := os.ReadFile(filepath.Join(repo, "skeleton/backend/go-gin", path))
		if err != nil {
			t.Fatal(err)
		}
		content := strings.ReplaceAll(string(source), "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend", module)
		if err := os.WriteFile(filepath.Join(backend, path), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	proxy, version := frameworkSnapshot(t)
	run := func(args ...string) {
		t.Helper()
		command := exec.Command("go", args...)
		command.Dir = backend
		upstream := os.Getenv("GOPROXY")
		if upstream == "" {
			upstream = "https://proxy.golang.org,direct"
		}
		command.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=file://"+filepath.ToSlash(proxy)+","+upstream, "GONOSUMDB="+frameworkModule)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatal(err, string(output))
		}
	}
	run("mod", "edit", "-require="+frameworkModule+"@"+version)
	raw, err := os.ReadFile(filepath.Join(backend, "go.mod"))
	if err != nil || strings.Contains(string(raw), "replace github.com/ArtisanCloud/PowerXPlugin/framework/backend/go") {
		t.Fatal(err, string(raw))
	}
	run("test", "-race", "-mod=mod", "./cmd/plugin", "./internal/bootstrap", "./internal/transport/http/wsbus", "-count=1", "-timeout=90s")
}
