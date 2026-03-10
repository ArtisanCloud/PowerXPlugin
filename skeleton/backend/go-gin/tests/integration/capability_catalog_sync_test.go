package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
)

func TestCapabilityCatalogSyncExportsWorkflowAssets(t *testing.T) {
	repoRoot := repoRootFromIntegration(t)
	ensureCapabilitySyncFixtures(t, repoRoot)
	catalogPath := filepath.Join(repoRoot, "capabilities", "catalog.json")
	if _, err := os.Stat(catalogPath); err != nil {
		t.Fatalf("缺少 catalog 文件，请先执行 capabilities:export: %v", err)
	}

	t.Setenv("POWERX_CAPABILITY_CATALOG", catalogPath)
	chdirForTest(t, repoRoot)

	ctx := context.Background()
	mgr := capabilities.NewManager(nil, nil)

	entries, err := mgr.ListCapabilities(ctx)
	if err != nil {
		t.Fatalf("读取能力目录失败: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("期望内置能力至少 1 个")
	}

	assets, err := mgr.ExportProtocols(ctx)
	if err != nil {
		t.Fatalf("导出协议资产失败: %v", err)
	}
	if len(assets) == 0 {
		t.Fatalf("期望导出至少一个协议资产")
	}

	required := []string{
		"contracts/exposure/openapi.yaml",
		"contracts/exposure/workflow/template-compose.json",
		"contracts/exposure/agent-streams/template-compose.yaml",
		"contracts/exposure/mcp-tools.json",
		"dist/agent-sdk/manifest.json",
	}
	for _, rel := range required {
		if !containsAssetPath(assets, rel) {
			t.Fatalf("协议资产缺失: %s", rel)
		}
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
			t.Fatalf("目标文件不存在 %s: %v", rel, err)
		}
	}

	client := &mockHostClient{}
	if err := mgr.RegisterWithHost(ctx, client); err != nil {
		t.Fatalf("注册能力目录到宿主失败: %v", err)
	}
	if client.catalog == nil || client.catalog.PluginID == "" {
		t.Fatalf("HostSyncClient 未收到 catalog 载荷")
	}
	if len(client.assets) != len(assets) {
		t.Fatalf("HostSyncClient 接收的资产数量不符，期望 %d 实际 %d", len(assets), len(client.assets))
	}
}

func ensureCapabilitySyncFixtures(t *testing.T, repoRoot string) {
	t.Helper()
	manifestPath := filepath.Join(repoRoot, "dist", "agent-sdk", "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("创建测试 fixture 目录失败: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{\"plugin_id\":\"com.powerx.plugins.base\",\"tools\":[]}\n"), 0o644); err != nil {
		t.Fatalf("写入测试 fixture 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(manifestPath)
	})
}
