package contracts

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
)

//go:embed data/manifest.json
//go:embed data/rbac.json
//go:embed data/openapi.yaml
var content embed.FS

const (
	manifestPath = "data/manifest.json"
	rbacPath     = "data/rbac.json"
	openapiPath  = "data/openapi.yaml"
)

// File 表示需要分发的契约文件。
type File struct {
	Name string
	Data []byte
}

// Files 返回契约文件内容列表。
func Files() ([]File, error) {
	sources := []struct {
		path string
		name string
	}{
		{path: manifestPath, name: "docs/contracts/manifest.json"},
		{path: rbacPath, name: "docs/contracts/rbac.json"},
		{path: openapiPath, name: "docs/contracts/openapi.yaml"},
	}

	var result []File
	for _, src := range sources {
		data, err := fs.ReadFile(content, filepath.ToSlash(src.path))
		if err != nil {
			return nil, fmt.Errorf("read contract %s: %w", src.name, err)
		}
		result = append(result, File{Name: src.name, Data: data})
	}
	return result, nil
}
