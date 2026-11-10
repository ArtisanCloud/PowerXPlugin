package manifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema/manifest.json
var manifestSchemaData []byte

var (
	manifestOnce     sync.Once
	manifestCompiler *jsonschema.Schema
	manifestErr      error
)

// Validate 检查 Manifest 是否符合 Schema 约束。
func Validate(p Plugin) error {
	if err := ValidatePluginID(p.ID); err != nil {
		return err
	}
	var validateMenus func([]Menu) error
	validateMenus = func(menus []Menu) error {
		for _, menu := range menus {
			if !strings.HasPrefix(menu.Path, "/_p/"+p.ID+"/") {
				return &IdentityError{Code: "PLUGIN_ID_MISMATCH", Field: "menus.path", Expected: "/_p/" + p.ID + "/", Actual: menu.Path}
			}
			if err := validateMenus(menu.Children); err != nil {
				return err
			}
		}
		return nil
	}
	if err := validateMenus(p.Menus); err != nil {
		return err
	}
	if err := ValidateNotificationPresentations(p.NotificationPresentations); err != nil {
		return err
	}
	manifestOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("manifest.json", io.NopCloser(bytes.NewReader(manifestSchemaData))); err != nil {
			manifestErr = err
			return
		}
		manifestCompiler, manifestErr = compiler.Compile("manifest.json")
	})
	if manifestErr != nil {
		return manifestErr
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	return manifestCompiler.Validate(payload)
}
