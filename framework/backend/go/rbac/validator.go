package rbac

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	_ "embed"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema/rbac.json
var rbacSchemaData []byte

var (
	rbacOnce   sync.Once
	rbacSchema *jsonschema.Schema
	rbacErr    error
)

// Validate 检查权限声明是否符合 Schema。
func Validate(perms []Permission) error {
	rbacOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("rbac.json", io.NopCloser(bytes.NewReader(rbacSchemaData))); err != nil {
			rbacErr = err
			return
		}
		rbacSchema, rbacErr = compiler.Compile("rbac.json")
	})
	if rbacErr != nil {
		return rbacErr
	}
	payload := map[string]any{
		"pluginId": "com.powerx.sample",
		"roles": []map[string]any{
			{
				"name":        "__placeholder__",
				"permissions": extractKeys(perms),
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var anyPayload any
	if err := json.Unmarshal(data, &anyPayload); err != nil {
		return err
	}
	return rbacSchema.Validate(anyPayload)
}

func extractKeys(perms []Permission) []string {
	keys := make([]string, 0, len(perms))
	for _, perm := range perms {
		keys = append(keys, perm.Key)
	}
	return keys
}
