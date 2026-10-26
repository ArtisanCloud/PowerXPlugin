package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitFrameworkReferences(t *testing.T) {
	for _, test := range []struct {
		name      string
		env       bool
		override  bool
		goVersion string
		adminRef  string
		clientRef string
	}{
		{"defaults", false, false, defaultFrameworkVersion, defaultAdminVersion, defaultClientVersion},
		{"snapshot_environment", true, false, "v0.0.27-dev.fixture", "file:/sdk bundle/admin.tgz", "file:/sdk bundle/client.tgz"},
		{"explicit_flags_override_environment", true, true, "v0.0.27-dev.explicit", "file:/explicit/admin.tgz", "file:/explicit/client.tgz"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("POWERXPLUGIN_USE_LOCAL_FRONTEND", "")
			for _, key := range []string{"PX_PLUGIN_FRAMEWORK_VERSION", "PX_PLUGIN_FRAMEWORK_ADMIN_REF", "PX_PLUGIN_FRAMEWORK_CLIENT_REF"} {
				t.Setenv(key, "")
			}
			if test.env {
				t.Setenv("PX_PLUGIN_FRAMEWORK_VERSION", "v0.0.27-dev.fixture")
				t.Setenv("PX_PLUGIN_FRAMEWORK_ADMIN_REF", "file:/sdk bundle/admin.tgz")
				t.Setenv("PX_PLUGIN_FRAMEWORK_CLIENT_REF", "file:/sdk bundle/client.tgz")
			}
			dir := filepath.Join(t.TempDir(), "plugin")
			args := []string{"com.powerx.framework-test", "--directory", dir, "--backend", "go-gin", "--admin", "nuxt", "--git-init=false", "--init-config=false"}
			if test.override {
				args = append(args, "--framework-version", test.goVersion, "--framework-admin-ref", test.adminRef, "--framework-client-ref", test.clientRef)
			}
			if err := runInit(args); err != nil {
				t.Fatal(err)
			}
			manifest, err := os.ReadFile(filepath.Join(dir, "backend/go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(manifest), "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go "+test.goVersion) || strings.Contains(string(manifest), "replace ") {
				t.Fatal("FRAMEWORK_GENERATED_GO_REFERENCE_INVALID")
			}
			raw, err := os.ReadFile(filepath.Join(dir, "web-admin/package.json"))
			if err != nil {
				t.Fatal(err)
			}
			var pkg struct {
				Dependencies map[string]string `json:"dependencies"`
			}
			if err := json.Unmarshal(raw, &pkg); err != nil {
				t.Fatal(err)
			}
			if pkg.Dependencies["@artisan-cloud/plugin-framework-admin"] != test.adminRef || pkg.Dependencies["@artisan-cloud/plugin-framework-client"] != test.clientRef {
				t.Fatal("FRAMEWORK_GENERATED_NPM_REFERENCE_INVALID")
			}
		})
	}
}

func TestInitDependencyFailureIsReturned(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\nexit 23\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := filepath.Join(t.TempDir(), "plugin")
	err := runInit([]string{"com.powerx.framework-test", "--directory", dir, "--backend", "go-gin", "--admin", "nuxt", "--git-init=false", "--init-config=false", "--install-deps"})
	if err == nil || !strings.Contains(err.Error(), "PLUGIN_INIT_DEPENDENCIES_FAILED") {
		t.Fatal("FRAMEWORK_INIT_MUST_REPORT_DEPENDENCY_FAILURE")
	}
}
