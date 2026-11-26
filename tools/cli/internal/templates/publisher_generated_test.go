package templates

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Package the current Framework SDK and independently build the CLI without a
// workspace/replace. Neither the SDK nor CLI is published by this test.
func TestPublisherPackagedFrameworkConsumer(t *testing.T) {
	if os.Getenv("PX_TEST_GENERATED_BUILD") != "1" {
		t.Skip("PX_TEST_GENERATED_BUILD=1")
	}
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	cli := filepath.Join(root, "cli")
	source := filepath.Join(repo, "tools/cli")
	err = filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == "bin" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(cli, rel), 0700)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(cli, rel), b, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	proxy, version := frameworkSnapshot(t)
	run := func(dir string, args ...string) string {
		t.Helper()
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GOWORK=off", "GONOSUMDB="+frameworkModule, "GOPROXY=file://"+filepath.ToSlash(proxy)+",https://proxy.golang.org,direct")
		b, err := c.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(b))
		}
		return string(b)
	}
	run(cli, "go", "mod", "edit", "-require="+frameworkModule+"@"+version, "-dropreplace=github.com/powerx-plugin/cli")
	binary := filepath.Join(root, "px-plugin")
	run(cli, "go", "build", "-mod=mod", "-o", binary, "./cmd/px-plugin")
	plugin := filepath.Join(root, "independent")
	run(root, binary, "init", "com.example.publisher-consumer", "--directory", plugin, "--backend", "go-gin", "--admin", "nuxt", "--module", "example.com/publisher-consumer", "--git-init=false", "--init-config=false")
	run(plugin, binary, "publisher", "plan", "create", "--entry", plugin, "--version", "1.2.0")
	run(plugin, binary, "publisher", "market", "add", "--entry", plugin, "--name", "fixture", "--api-root", "https://api.example/v1", "--console-base", "https://market.example")

	request := filepath.Join(root, "submission.json")
	raw, _ := json.Marshal(map[string]any{
		"connection_id": "00000000-0000-4000-8000-000000000001",
		"version_id":    "00000000-0000-4000-8000-000000000002", "version": "1.2.0",
		"metadata":           map[string]any{"description": "  generated consumer description  ", "license": "MIT", "min_core": "1.0.0", "support_url": "https://support.example", "icon_url": "", "security_statement": "consumer security statement", "privacy_statement": "consumer privacy statement"},
		"expected_artifacts": []any{map[string]any{"artifact_id": "00000000-0000-4000-8000-000000000003", "os": "linux", "arch": "amd64", "sha256": strings.Repeat("a", 64), "size": 10}},
	})
	if err := os.WriteFile(request, raw, 0600); err != nil {
		t.Fatal(err)
	}
	output := run(plugin, binary, "publisher", "review", "preview", "--entry", plugin, "--request-file", request)
	var preview struct {
		Data struct {
			Input struct {
				Metadata struct {
					Description string `json:"description"`
				} `json:"metadata"`
			} `json:"input"`
			SHA256 string `json:"request_sha256"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(output), &preview) != nil || preview.Data.Input.Metadata.Description != "generated consumer description" || len(preview.Data.SHA256) != 64 {
		t.Fatal("independent_submission_preview", output)
	}
	if _, err := os.ReadFile(filepath.Join(plugin, "web-admin/.env.example")); err != nil {
		t.Fatal("publisher_template_missing", "web-admin/.env.example", err)
	}
	for _, file := range []string{"web-admin/app/components/AppSidebar.vue", "web-admin/app/middleware/layout.global.ts", "web-admin/i18n/locales/en.json"} {
		b, err := os.ReadFile(filepath.Join(plugin, file))
		if err != nil {
			t.Fatal("publisher_template_missing", file, err)
		}
		if !strings.Contains(string(b), "publisher") && !strings.Contains(string(b), "PUBLISHER") {
			t.Fatal("publisher_template_marker_missing", file)
		}
	}
	nuxtConfig, err := os.ReadFile(filepath.Join(plugin, "web-admin/nuxt.config.ts"))
	if err != nil {
		t.Fatal("publisher_template_missing", "web-admin/nuxt.config.ts", err)
	}
	if !strings.Contains(string(nuxtConfig), "definePowerXAdminConfig(") {
		t.Fatal("publisher_framework_admin_config_missing")
	}
	run(filepath.Join(plugin, "backend"), "go", "mod", "edit", "-require="+frameworkModule+"@"+version)
	run(filepath.Join(plugin, "backend"), "go", "test", "-mod=mod", "./cmd/plugin")
	for _, dir := range []string{cli, filepath.Join(plugin, "backend")} {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil || strings.Contains(string(b), "replace ") {
			t.Fatal("independent_module_replace", err)
		}
	}
	t.Logf("framework_snapshot=%s consumer_mode=GOWORK_off_no_replace", version)
}
