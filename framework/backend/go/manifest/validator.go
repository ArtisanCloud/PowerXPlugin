package manifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
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
