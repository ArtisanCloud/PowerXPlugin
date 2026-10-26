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
//go:embed data/media-contracts
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
		{path: "data/media-contracts/ai-settings-llm-catalog.v1.md", name: "docs/contracts/ai-settings-llm-catalog.v1.md"},
		{path: "data/media-contracts/fixtures/ai-settings-catalog.v1.json", name: "docs/contracts/fixtures/ai-settings-catalog.v1.json"},
		{path: "data/media-contracts/ai-media-v2.md", name: "docs/contracts/ai-media-v2.md"},
		{path: "data/media-contracts/ai-media-v2.openapi.yaml", name: "docs/contracts/ai-media-v2.openapi.yaml"},
		{path: "data/media-contracts/ai-media-v2-source.json", name: "docs/contracts/ai-media-v2-source.json"},
		{path: "data/media-contracts/fixtures/ai-media-v2.json", name: "docs/contracts/fixtures/ai-media-v2.json"},
		{path: "data/media-contracts/fixtures/ai-media-v2-registration.json", name: "docs/contracts/fixtures/ai-media-v2-registration.json"},
		{path: "data/media-contracts/fixtures/capability-version-upgrade-errors.v1.json", name: "docs/contracts/fixtures/capability-version-upgrade-errors.v1.json"},
		{path: "data/media-contracts/business-notification-card.openapi.yaml", name: "docs/contracts/business-notification-card.openapi.yaml"},
		{path: "data/media-contracts/image-derivatives.openapi.yaml", name: "docs/contracts/image-derivatives.openapi.yaml"},
		{path: "data/media-contracts/media-host.openapi.yaml", name: "docs/contracts/media-host.openapi.yaml"},
		{path: "data/media-contracts/fixtures/business-notification-card.v1.json", name: "docs/contracts/fixtures/business-notification-card.v1.json"},
		{path: "data/media-contracts/fixtures/business-notification-card.v2.json", name: "docs/contracts/fixtures/business-notification-card.v2.json"},
		{path: "data/media-contracts/fixtures/media-image-derivatives.v2.json", name: "docs/contracts/fixtures/media-image-derivatives.v2.json"},
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
