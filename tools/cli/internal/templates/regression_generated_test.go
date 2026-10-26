package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegressionGeneratedPlugins(t *testing.T) {
	if os.Getenv("REGRESSION_GENERATED_TEST") != "1" {
		t.Skip("REGRESSION_GENERATED_TEST")
	}
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"com.powerx.regression.alpha", "com.powerx.regression.beta"} {
		t.Run(id, func(t *testing.T) {
			destination := t.TempDir()
			if output := os.Getenv("REGRESSION_GENERATED_OUTPUT"); output != "" {
				destination = filepath.Join(output, strings.TrimPrefix(id, "com.powerx.regression."))
				if err := os.MkdirAll(destination, 0755); err != nil {
					t.Fatal(err)
				}
			}
			module := "example.test/" + strings.ReplaceAll(id, ".", "/") + "/backend"
			_, err := RenderAll(destination, Data{PluginID: id, PluginName: "regression.fixture.name", PluginSlug: "regression-fixture", PluginDBName: "regression_fixture", Version: "0.1.0", GoVersion: "1.24", BackendModulePath: module, BackendType: BackendGoGin, FrontendType: FrontendNuxt, BackendPort: 8078, FrontendPort: 3131, FrameworkVersion: "v0.0.25", FrameworkAdminRef: "file:" + filepath.Join(repo, "framework/frontend/nuxt/framework-admin"), FrameworkClientRef: "file:" + filepath.Join(repo, "framework/frontend/nuxt/framework-client"), AppFrontendType: FrontendNuxt}, Options{Force: os.Getenv("REGRESSION_GENERATED_OUTPUT") != ""})
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"backend/internal/services/admin/regression/service.go", "backend/internal/services/admin/regression/scenarios/template_lifecycle.go", "backend/internal/services/admin/regression/scenarios/template_business.go", "backend/internal/services/admin/regression/template_http.go", "backend/internal/transport/http/admin/regression/events.go", "backend/cmd/database/migrate/regression.go", "web-admin/app/pages/admin/regression/index.vue", "web-admin/app/pages/_p/" + id + "/admin/regression/index.vue"} {
				if _, err = os.Stat(filepath.Join(destination, path)); err != nil {
					t.Fatal(path, err)
				}
			}
			backend := filepath.Join(destination, "backend")
			for _, name := range []string{"handler_test.go", "browser_fixture_test.go"} {
				source, err := os.ReadFile(filepath.Join(repo, "skeleton/backend/go-gin/internal/transport/http/admin/regression", name))
				if err != nil {
					t.Fatal(err)
				}
				content := strings.ReplaceAll(string(source), "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend", module)
				content = strings.ReplaceAll(content, "../../../../../../../web-admin/nuxt/i18n/locales/en.json", "../../../../../../web-admin/i18n/locales/en.json")

				if err := os.WriteFile(filepath.Join(backend, "internal/transport/http/admin/regression", name), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// This deliberately verifies unreleased local Framework source. It makes no published-module claim.
			command := exec.Command("go", "mod", "edit", "-replace=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go="+filepath.Join(repo, "framework/backend/go"))
			command.Dir = backend
			command.Env = append(os.Environ(), "GOWORK=off")
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatal(err, string(output))
			}
			for _, name := range []string{"template_lifecycle_test.go", "template_catalog_test.go"} {
				source, err := os.ReadFile(filepath.Join(repo, "skeleton/backend/go-gin/tests/integration/regression", name))
				if err != nil {
					t.Fatal(err)
				}
				content := strings.ReplaceAll(string(source), "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend", module)
				// Run the exact business catalog acceptance against each generated plugin and its own locale.
				content = strings.ReplaceAll(content, "../../../../../web-admin/nuxt/i18n/locales/en.json", "../web-admin/i18n/locales/en.json")
				if err = os.WriteFile(filepath.Join(backend, "generated_"+name), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			command = exec.Command("go", "test", "-mod=mod", ".", "./internal/services/admin/regression/...", "./internal/transport/http/admin/regression", "./cmd/plugin")
			command.Dir = backend
			command.Env = append(os.Environ(), "GOWORK=off")
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatal(err, string(output))
			}
		})
	}
}
