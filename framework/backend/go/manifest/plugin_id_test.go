package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestPluginIDContract(t *testing.T) {
	var cases struct{ Valid, Invalid []string }
	raw, err := os.ReadFile("schema/plugin-id.cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, id := range cases.Valid {
		if err := ValidatePluginID(id); err != nil {
			t.Errorf("%q: %v", id, err)
		}
	}
	for _, id := range cases.Invalid {
		if err := ValidatePluginID(id); err == nil {
			t.Errorf("ACCEPTED_INVALID_ID: %q", id)
		}
	}
	var schema map[string]any
	if err := json.Unmarshal(pluginIDSchema, &schema); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"$schema", "title", "$comment"} {
		delete(schema, key)
	}
	for _, file := range []string{
		"framework/backend/go/manifest/schema/manifest.json", "docs/contracts/manifest.json", "tools/cli/internal/contracts/data/manifest.json",
		"docs/standards/powerx-plugin/lifecycle/contracts/manifest.schema.json", "docs/standards/powerx-plugin/integration/01_plugin_lifecycle/contracts/manifest.schema.json",
	} {
		t.Run(file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("../../../..", file))
			if os.IsNotExist(err) {
				t.Skip("SOURCE_TREE_REQUIRED")
			}
			if err != nil {
				t.Fatal(err)
			}
			var document struct{ Properties map[string]map[string]any }
			if err := json.Unmarshal(raw, &document); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(document.Properties["id"], schema) {
				t.Fatal("PLUGIN_ID_CONTRACT_DRIFT")
			}
			constraint, _ := json.Marshal(document.Properties["id"])
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource("id.json", bytes.NewReader(constraint)); err != nil {
				t.Fatal(err)
			}
			compiled, err := compiler.Compile("id.json")
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range cases.Valid {
				if err := compiled.Validate(id); err != nil {
					t.Errorf("%q: %v", id, err)
				}
			}
			for _, id := range cases.Invalid {
				if compiled.Validate(id) == nil {
					t.Errorf("ACCEPTED_INVALID_ID: %q", id)
				}
			}
		})
	}
}

func TestPluginIdentityConsistency(t *testing.T) {
	id := "com.powerx.plugins.demo"
	for _, values := range [][2]string{{id, "plugin:" + id}, {"", ""}} {
		if err := ValidatePluginIdentity(id, values[0], values[1]); err != nil {
			t.Fatal(err)
		}
	}
	for _, values := range [][2]string{{"com.example.other", ""}, {"", "plugin:com.example.other"}} {
		var mismatch *IdentityError
		if !errors.As(ValidatePluginIdentity(id, values[0], values[1]), &mismatch) || mismatch.Code != "PLUGIN_ID_MISMATCH" {
			t.Fatal("IDENTITY_MISMATCH_NOT_REJECTED")
		}
	}
	for _, field := range []string{"POWERX_PLUGIN_ID", "POWERX_SECURITY_JWT_AUDIENCE", "NUXT_PUBLIC_PLUGIN_ID"} {
		document := map[string]any{"id": id, "runtime": map[string]any{"env": map[string]any{field: "com.example.other"}}}
		if ValidatePluginDocument(document) == nil {
			t.Fatal("IDENTITY_MISMATCH_NOT_REJECTED", field)
		}
	}
	for _, field := range []string{"frontend", "frontends", "menus"} {
		document := map[string]any{"id": id, field: []any{map[string]any{"path": "/_p/com.example.other/admin/"}}}
		if ValidatePluginDocument(document) == nil {
			t.Fatal("IDENTITY_MISMATCH_NOT_REJECTED", field)
		}
	}
	if Validate(Plugin{ID: id, Name: "demo", Menus: []Menu{{Path: "/_p/com.example.other/admin/"}}}) == nil {
		t.Fatal("REGISTERED_ID_MISMATCH_NOT_REJECTED")
	}
	// Rejection fixtures and import paths are not production identities.
	if err := ValidatePluginDocument(map[string]any{"id": id, "description": "com.powerx.plugin.legacy", "imports": []any{"com.powerx.plugin.repo/backend"}}); err != nil {
		t.Fatal(err)
	}
}

func TestPluginIDDiagnostics(t *testing.T) {
	for _, id := range []string{"com.powerx.plugin.demo", "com..demo"} {
		err := ValidatePluginID(id)
		var identity *IdentityError
		if !errors.As(err, &identity) {
			t.Fatal(err)
		}
		if FormatPluginIDError(err, "zh_CN.UTF-8") == FormatPluginIDError(err, "en_US.UTF-8") {
			t.Fatal("PLUGIN_ID_LOCALE_MISSING")
		}
	}
}
