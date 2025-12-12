package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
)

type mockHostClient struct {
	catalog *capabilities.CatalogSnapshot
	assets  []capabilities.ProtocolAsset
}

func (m *mockHostClient) RegisterCatalog(ctx context.Context, catalog *capabilities.CatalogSnapshot, assets []capabilities.ProtocolAsset) error {
	m.catalog = catalog
	if len(assets) > 0 {
		m.assets = append(m.assets[:0], assets...)
	} else {
		m.assets = nil
	}
	return nil
}

func repoRootFromIntegration(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", ".."))
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("读取当前目录失败: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("切换到目录 %s 失败: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
}

func containsAssetPath(assets []capabilities.ProtocolAsset, rel string) bool {
	for _, asset := range assets {
		if asset.Path == rel {
			return true
		}
	}
	return false
}
