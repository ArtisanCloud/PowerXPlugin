package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Test the real CLI, shared Make gate and independent generated consumers.
// Runtime installation and database migrations are deliberately not performed.
func TestPluginIDGeneratedConsumers(t *testing.T) {
	if os.Getenv("PX_TEST_PLUGIN_ID") != "1" {
		t.Skip("PX_TEST_PLUGIN_ID=1")
	}
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	binary := filepath.Join(directory, "px-plugin")
	build := exec.Command("go", "build", "-o", binary, "./tools/cli/cmd/px-plugin")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	proxy, version := frameworkSnapshot(t)
	upstream := os.Getenv("GOPROXY")
	if upstream == "" {
		upstream = "https://proxy.golang.org"
	}
	environment := append(os.Environ(), "GOWORK=off", "GOPROXY=file://"+filepath.ToSlash(proxy)+","+upstream, "GONOSUMDB="+frameworkModule, "PYTHONDONTWRITEBYTECODE=1", "POWERX_PLUGIN_ID=", "POWERX_SECURITY_JWT_AUDIENCE=", "NUXT_APP_BASE_URL=")
	for _, backend := range []string{BackendGoGin, BackendPythonFast} {
		for _, id := range []string{"com.powerx.plugins.demo", "com.example.demo"} {
			t.Run(backend+"/"+id, func(t *testing.T) {
				destination := filepath.Join(directory, backend, id)
				run := func(name string, args ...string) []byte {
					t.Helper()
					cmd := exec.Command(name, args...)
					cmd.Dir = destination
					cmd.Env = environment
					output, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatal(err, string(output))
					}
					return output
				}
				init := exec.Command(binary, "init", id, "--directory", destination, "--backend", backend, "--admin", "nuxt", "--framework-version", version, "--git-init=false", "--init-config=false")
				init.Env = environment
				if output, err := init.CombinedOutput(); err != nil {
					t.Fatal(err, string(output))
				}
				run(binary, "plugin-id", "check", "--manifest", "plugin.yaml")
				run("make", "plugin-id-check", "PX_PLUGIN="+binary)
				paths := []string{"web-admin/nuxt.config.ts"}
				if backend == BackendGoGin {
					paths = append(paths, "backend/internal/shared/app/consts.go", "backend/internal/manifestx/manifest.go")
				} else {
					paths = append(paths, "backend/app/manifestx/identity.py")
				}
				for _, path := range paths {
					raw, err := os.ReadFile(filepath.Join(destination, path))
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(raw), id) || strings.Contains(string(raw), "com.powerx.plugins.base") {
						t.Fatal("GENERATED_ID_MISMATCH", path)
					}
				}
				if backend == BackendGoGin {
					cmd := exec.Command("go", "run", "-mod=mod", "./cmd/manifestcheck", "--plugin", "../plugin.yaml", "--plugin-only")
					cmd.Dir = filepath.Join(destination, "backend")
					cmd.Env = environment
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatal(err, string(output))
					}
					// Real backend binaries are built and archived; the frontend is
					// excluded explicitly because this acceptance concerns identity.
					run(binary, "package", "--entry", destination, "--skip-frontend", "--output-dir", filepath.Join(destination, "package"))
					packages, err := filepath.Glob(filepath.Join(destination, "package", "*", "package.tar.gz"))
					if err != nil || len(packages) != 1 {
						t.Fatal("GENERATED_PACKAGE_MISSING", err)
					}
				} else {
					run("python3", "-c", `import importlib.util,json,sys
spec=importlib.util.spec_from_file_location("identity", "backend/app/manifestx/identity.py")
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
assert module.PLUGIN_ID==sys.argv[1]
cases=json.load(open(sys.argv[2]))
for value in cases["valid"]: module.validate_plugin_id(value)
for value in cases["invalid"]:
 try: module.validate_plugin_id(value)
 except ValueError: pass
 else: raise AssertionError("INVALID_ID_ACCEPTED")
module.validate_plugin_identity(module.PLUGIN_ID,module.PLUGIN_ID,"plugin:"+module.PLUGIN_ID)
for values in [("com.example.other",""),("","plugin:com.example.other")]:
 try: module.validate_plugin_identity(module.PLUGIN_ID,*values)
 except ValueError: pass
 else: raise AssertionError("MISMATCH_ACCEPTED")
`, id, filepath.Join(repo, "docs/standards/powerx-plugin/_shared/plugin-id.cases.json"))
				}
				manifestFile := filepath.Join(destination, "plugin.yaml")
				original, err := os.ReadFile(manifestFile)
				if err != nil {
					t.Fatal(err)
				}
				legacy := strings.Replace(string(original), "id: "+id, "id: com.powerx.plugin.legacy", 1)
				if err := os.WriteFile(manifestFile, []byte(legacy), 0600); err != nil {
					t.Fatal(err)
				}
				for _, invocation := range [][]string{{binary, "package", "--entry", destination, "--skip-backend", "--skip-frontend", "--output-dir", filepath.Join(destination, "must-not-exist")}, {"make", "plugin-id-check", "PX_PLUGIN=" + binary}, {"make", "plugin-yaml-check", "PX_PLUGIN=" + binary}, {"make", "dist", "PX_PLUGIN=" + binary}} {
					cmd := exec.Command(invocation[0], invocation[1:]...)
					cmd.Dir = destination
					cmd.Env = environment
					if output, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(output), "PLUGIN_ID_LEGACY_PREFIX") {
						t.Fatal("LEGACY_ID_NOT_REJECTED", err, string(output))
					}
				}
				if _, err := os.Stat(filepath.Join(destination, "must-not-exist")); !os.IsNotExist(err) {
					t.Fatal("INVALID_ID_WROTE_PACKAGE")
				}
			})
		}
	}
}
