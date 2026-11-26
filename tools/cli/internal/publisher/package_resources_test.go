//go:build darwin || linux

package publisher

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNuxtWorkspaceResourcesUseInstalledPackagePaths(t *testing.T) {
	root, engine, plan := buildFixture(t)
	frontend := filepath.Join(root, "web-admin")
	if err := os.Rename(frontend, filepath.Join(root, "frontend-input")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(frontend, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "frontend-input"), filepath.Join(frontend, "nuxt")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"public/images/logo.svg", "i18n/locales/en.json"} {
		file := filepath.Join(frontend, "nuxt", rel)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	script := filepath.Join(root, "scripts/dist.mjs")
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "catalogs: {capabilities:", "assets: {public_dir: web-admin/public, webAdminPath: web-admin/.output}\\ncatalogs: {capabilities:", 1))
	if err = os.WriteFile(script, data, 0600); err != nil {
		t.Fatal(err)
	}
	build := runFixture(t, engine, plan)
	requireReady(t, engine, build)
	archive, err := zip.OpenReader(filepath.Join(engine.state.path, build.Artifact.Path))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	entries := map[string]bool{}
	for _, file := range archive.File {
		entries[file.Name] = true
	}
	for _, rel := range []string{"web-admin/public/images/logo.svg", "web-admin/i18n/locales/en.json"} {
		if !entries[plan.Version+"/"+rel] {
			t.Fatal("installed_resource_missing", rel)
		}
	}
}

func TestSchemaIdentityIsNotAFileReference(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "contracts/schema/input"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "contracts/schema/input/fixture.json")
	if err := os.WriteFile(file, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"provides": []any{map[string]any{"id": "schema/input/fixture.v1", "path": "schema/input/fixture.json"}}}
	if err := checkReferences(root, doc); err != nil {
		t.Fatal("schema_identity_treated_as_path", err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := checkReferences(root, doc); err == nil {
		t.Fatal("missing_schema_path_accepted")
	}
	if err := checkReferences(root, map[string]any{"schema": "contracts/schema/input/fixture.json"}); err == nil {
		t.Fatal("missing_nested_schema_reference_accepted")
	}
	if err := checkReferences(root, map[string]any{"schema": "./contracts/schema/input/fixture.json"}); err == nil {
		t.Fatal("missing_relative_schema_reference_accepted")
	}
}
