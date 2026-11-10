package manifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema/plugin-id.json
var pluginIDSchema []byte

//go:embed locales/plugin-id.json
var pluginIDLocales []byte

var pluginIDOnce sync.Once
var pluginIDCompiled *jsonschema.Schema
var pluginIDCompileError error

// IdentityError exposes stable codes; translate at the presentation boundary.
type IdentityError struct {
	Code     string
	Field    string
	Expected string
	Actual   string
}

func (e *IdentityError) Error() string { return e.Code }

// ValidatePluginID is shared by generation, packaging and runtime. Its schema
// is generated from PowerXDocs and is never independently maintained here.
func ValidatePluginID(id string) error {
	pluginIDOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		pluginIDCompileError = compiler.AddResource("plugin-id.json", bytes.NewReader(pluginIDSchema))
		if pluginIDCompileError == nil {
			pluginIDCompiled, pluginIDCompileError = compiler.Compile("plugin-id.json")
		}
	})
	if pluginIDCompileError != nil {
		return pluginIDCompileError
	}
	if err := pluginIDCompiled.Validate(id); err != nil {
		code := "PLUGIN_ID_INVALID"
		if id == "com.powerx.plugin" || strings.HasPrefix(id, "com.powerx.plugin.") {
			code = "PLUGIN_ID_LEGACY_PREFIX"
		}
		return &IdentityError{Code: code, Field: "id", Actual: id}
	}
	return nil
}

// ValidatePluginIdentity checks optional runtime identity and plugin audience
// against the canonical ID. Absent settings are not filled in or rewritten.
func ValidatePluginIdentity(id, runtimeID, audience string) error {
	if err := ValidatePluginID(id); err != nil {
		return err
	}
	for _, value := range []struct{ field, actual, expected string }{
		{"POWERX_PLUGIN_ID", runtimeID, id},
		{"POWERX_SECURITY_JWT_AUDIENCE", audience, "plugin:" + id},
	} {
		if value.actual != "" && value.actual != value.expected {
			return &IdentityError{Code: "PLUGIN_ID_MISMATCH", Field: value.field, Expected: value.expected, Actual: value.actual}
		}
	}
	return nil
}

// ValidatePluginEnvironment checks the identity settings consumed by backend
// and frontend builds. Supplying the lookup keeps process state outside policy.
func ValidatePluginEnvironment(id string, lookup func(string) string) error {
	if err := ValidatePluginIdentity(id, lookup("POWERX_PLUGIN_ID"), lookup("POWERX_SECURITY_JWT_AUDIENCE")); err != nil {
		return err
	}
	for _, key := range []string{"NUXT_PUBLIC_PLUGIN_ID", "NUXT_PUBLIC_POWERX_PLUGIN_ID"} {
		if value := lookup(key); value != "" && value != id {
			return &IdentityError{Code: "PLUGIN_ID_MISMATCH", Field: key, Actual: value, Expected: id}
		}
	}
	if value := lookup("NUXT_APP_BASE_URL"); value != "" && !strings.HasPrefix(value, "/_p/"+id+"/") {
		return &IdentityError{Code: "PLUGIN_ID_MISMATCH", Field: "NUXT_APP_BASE_URL", Actual: value, Expected: "/_p/" + id + "/"}
	}
	return nil
}

// ValidatePluginDocument checks identity without unifying complete Manifest
// formats. Arbitrary examples, imports and capability names are not scanned.
func ValidatePluginDocument(document map[string]any) error {
	id, ok := document["id"].(string)
	if !ok {
		return &IdentityError{Code: "PLUGIN_ID_INVALID", Field: "id"}
	}
	if err := ValidatePluginID(id); err != nil {
		return err
	}
	var walk func(any, string) error
	walk = func(node any, location string) error {
		switch value := node.(type) {
		case map[string]any:
			for key, child := range value {
				field := location + "." + key
				var expected string
				switch key {
				case "POWERX_PLUGIN_ID", "NUXT_PUBLIC_PLUGIN_ID", "NUXT_PUBLIC_POWERX_PLUGIN_ID":
					expected = id
				case "POWERX_SECURITY_JWT_AUDIENCE":
					expected = "plugin:" + id
				}
				if expected != "" && child != expected {
					return &IdentityError{Code: "PLUGIN_ID_MISMATCH", Field: field, Expected: expected, Actual: fmt.Sprint(child)}
				}
				if path, ok := child.(string); ok && strings.HasPrefix(path, "/_p/") && !strings.HasPrefix(path, "/_p/"+id+"/") {
					return &IdentityError{Code: "PLUGIN_ID_MISMATCH", Field: field, Expected: "/_p/" + id + "/", Actual: path}
				}
				if err := walk(child, field); err != nil {
					return err
				}
			}
		case []any:
			for i, child := range value {
				if err := walk(child, fmt.Sprintf("%s[%d]", location, i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, key := range []string{"runtime", "frontend", "frontends", "menus"} {
		if err := walk(document[key], key); err != nil {
			return err
		}
	}
	return nil
}

// FormatPluginIDError renders diagnostics through a locale catalog.
func FormatPluginIDError(err error, locale string) string {
	var identity *IdentityError
	if !errors.As(err, &identity) {
		return err.Error()
	}
	var catalogs map[string]map[string]string
	if json.Unmarshal(pluginIDLocales, &catalogs) != nil {
		return identity.Code
	}
	language := "en"
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		language = "zh"
	}
	return fmt.Sprintf(catalogs[language][identity.Code], identity.Field, identity.Actual, identity.Expected)
}
