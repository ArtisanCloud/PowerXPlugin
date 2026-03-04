package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
)

func TestCapabilityAsyncModeEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	catalogPath := filepath.Join(tmp, "capabilities", "catalog.json")
	makeStubExposureTree(t, tmp)
	writeCatalogSnapshot(t, catalogPath, &capabilities.CatalogSnapshot{
		PluginID:        "com.powerx.test.async",
		ManifestVersion: "0.1.0",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Entries: []capabilities.CatalogEntry{
			{
				ID:         "com.powerx.demo.sync",
				Version:    "1.0.0",
				Descriptor: "contracts/capabilities/demo.sync.yaml",
				Schemas:    map[string]string{},
				Protocols:  map[string]interface{}{},
			},
			{
				ID:         "com.powerx.demo.async",
				Version:    "1.0.0",
				Descriptor: "contracts/capabilities/demo.async.yaml",
				Schemas:    map[string]string{},
				Protocols:  map[string]interface{}{},
				Execution: capabilities.ExecutionConfig{
					Mode:           "async",
					CallbackURL:    "https://callback.powerx.dev/template",
					SSEChannel:     "sse://template",
					StatusEndpoint: "https://status.powerx.dev/template",
					TimeoutSeconds: 30,
				},
			},
		},
	})

	t.Setenv("POWERX_CAPABILITY_CATALOG", catalogPath)
	chdirForTest(t, tmp)

	ctx := context.Background()
	mgr := capabilities.NewManager(nil, nil)

	entries, err := mgr.ListCapabilities(ctx)
	if err != nil {
		t.Fatalf("加载测试 catalog 失败: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("期望 2 条测试能力，得到 %d", len(entries))
	}
	if entries[0].Execution.Mode != "sync" {
		t.Fatalf("默认执行模式应为 sync，实际 %s", entries[0].Execution.Mode)
	}
	var asyncFound bool
	for _, entry := range entries {
		if entry.ID == "com.powerx.demo.async" {
			asyncFound = true
			if entry.Execution.Mode != "async" {
				t.Fatalf("async 能力执行模式应为 async，实际 %s", entry.Execution.Mode)
			}
			if entry.Execution.CallbackURL == "" || entry.Execution.StatusEndpoint == "" {
				t.Fatalf("async 能力缺少回调或状态查询字段")
			}
		}
	}
	if !asyncFound {
		t.Fatalf("未找到 async 能力")
	}

	assets, err := mgr.ExportProtocols(ctx)
	if err != nil {
		t.Fatalf("导出测试协议资产失败: %v", err)
	}
	if len(assets) < 5 {
		t.Fatalf("期望至少 5 个协议资产，实际 %d", len(assets))
	}

	client := &mockHostClient{}
	if err := mgr.RegisterWithHost(ctx, client); err != nil {
		t.Fatalf("注册测试 catalog 失败: %v", err)
	}
	if client.catalog == nil || client.catalog.PluginID != "com.powerx.test.async" {
		t.Fatalf("HostSyncClient 应收到完整 catalog")
	}
	if len(client.assets) != len(assets) {
		t.Fatalf("HostSyncClient 接收的资产数量不一致")
	}

	// 破坏 async/回调约束，应当报错
	writeCatalogSnapshot(t, catalogPath, &capabilities.CatalogSnapshot{
		PluginID:        "com.powerx.test.async",
		ManifestVersion: "0.1.0",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Entries: []capabilities.CatalogEntry{
			{
				ID:         "com.powerx.demo.sync",
				Version:    "1.0.0",
				Descriptor: "contracts/capabilities/demo.sync.yaml",
				Schemas:    map[string]string{},
				Protocols:  map[string]interface{}{},
			},
			{
				ID:         "com.powerx.demo.async",
				Version:    "1.0.0",
				Descriptor: "contracts/capabilities/demo.async.yaml",
				Schemas:    map[string]string{},
				Protocols:  map[string]interface{}{},
				Execution: capabilities.ExecutionConfig{
					Mode: "async",
				},
			},
		},
	})

	if _, err := mgr.ListCapabilities(ctx); err == nil {
		t.Fatalf("async 能力缺失回调/SSE/状态约束时应报错")
	}
}

func writeCatalogSnapshot(t *testing.T, path string, snapshot *capabilities.CatalogSnapshot) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("创建 catalog 目录失败: %v", err)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("序列化 catalog 失败: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("写入 catalog 失败: %v", err)
	}
}

func makeStubExposureTree(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"contracts/exposure/openapi.yaml":              "openapi: \"3.0.0\"\ninfo:\n  title: stub\n",
		"contracts/exposure/proto/demo.proto":          "syntax = \"proto3\";\npackage demo;\n",
		"contracts/exposure/workflow/demo.json":        "{\"id\":\"demo\"}\n",
		"contracts/exposure/mcp-tools.json":            "{\"plugin_id\":\"demo\",\"tools\":[]}\n",
		"contracts/exposure/agent-streams/demo.yaml":   "capability_id: demo\n",
		"dist/agent-sdk/manifest.json":                 "{\"plugin_id\":\"demo\",\"tools\":[]}\n",
	}
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("创建测试目录 %s 失败: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("写入测试文件 %s 失败: %v", rel, err)
		}
	}
}
