package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAICatalogDirectoryFromPluginRoot(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "backend", "etc", "ai", "providers.d")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POWERX_PLUGIN_ROOT", root)

	if got := defaultAICatalogDirectory(); got != want {
		t.Fatalf("defaultAICatalogDirectory() = %q, want %q", got, want)
	}
}

func TestDefaultAICatalogDirectoryMissing(t *testing.T) {
	// 插件根目录下没有打包 catalog 时不应返回不存在的路径
	t.Setenv("POWERX_PLUGIN_ROOT", t.TempDir())

	if got := defaultAICatalogDirectory(); got != "" {
		if _, err := os.Stat(got); err != nil {
			t.Fatalf("defaultAICatalogDirectory() returned non-existent path %q", got)
		}
	}
}
