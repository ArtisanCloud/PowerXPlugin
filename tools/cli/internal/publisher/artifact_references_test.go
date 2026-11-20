package publisher

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPackageSchemaReferenceSemantics(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	schemaDir := filepath.Join(root, "contracts", "schema", "input")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{filepath.Join(schemaDir, "fixture.json"), filepath.Join(root, "plugin.yaml")} {
		if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(schemaDir, "outside.json")); err != nil {
		t.Fatal(err)
	}

	var declaration map[string]any
	const capability = `provides:
  - id: schema/input/fixture.v1
    path: schema/input/fixture.json
description: contracts/examples/prose
metadata:
  protocols:
    rest:
      path: /api/v1/invoke
imports:
  - ./schema/input/fixture.json
  - descriptor: contracts/schema/input/fixture.json
`
	if err := yaml.Unmarshal([]byte(capability), &declaration); err != nil {
		t.Fatal(err)
	}
	if err := checkReferences(root, declaration); err != nil {
		t.Fatalf("valid capability rejected: %v", err)
	}

	for _, ref := range []string{
		"contracts/missing.json",
		"./schema/missing.json",
		"contracts/../plugin.yaml",
		"contracts/../../outside.json",
		"contracts//schema/input/fixture.json",
		"contracts/./schema/input/fixture.json",
		"schema/input/fixture.json/child",
		"schema/input/outside.json",
	} {
		t.Run(ref, func(t *testing.T) {
			testReference := map[string]any{
				"nested": []any{map[string]any{"imports": []any{"./schema/input/fixture.json", map[string]any{"path": ref}}}},
			}
			if err := checkReferences(root, testReference); err == nil || AsError(err).Code != ArtifactInvalid {
				t.Fatalf("expected %s for %q, got %v", ArtifactInvalid, ref, err)
			}
		})
	}
}
