package powerx

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
)

func TestBuildRequestPayloadReadsDiskPathAndKeepsPayloadPath(t *testing.T) {
	dir := t.TempDir()
	diskPath := filepath.Join(dir, "template-audit.yaml")
	content := []byte("id: template-audit\n")
	if err := os.WriteFile(diskPath, content, 0o644); err != nil {
		t.Fatalf("写入测试资产失败: %v", err)
	}

	client := &CapabilityClient{}
	payload, err := client.buildRequestPayload(&capabilities.CatalogSnapshot{
		PluginID: "com.powerx.plugins.base.local",
		Entries:  []capabilities.CatalogEntry{{ID: "com.powerx.plugins.base.local.template.audit"}},
	}, []capabilities.ProtocolAsset{{
		Type:     "agent_stream",
		Path:     "contracts/exposure/agent-streams/template-audit.yaml",
		DiskPath: diskPath,
	}})
	if err != nil {
		t.Fatalf("构建请求载荷失败: %v", err)
	}
	if len(payload.Assets) != 1 {
		t.Fatalf("期望 1 个资产，实际 %d", len(payload.Assets))
	}
	if payload.Assets[0].Path != "contracts/exposure/agent-streams/template-audit.yaml" {
		t.Fatalf("payload path 不应使用磁盘路径，实际 %s", payload.Assets[0].Path)
	}
	got, err := base64.StdEncoding.DecodeString(payload.Assets[0].Content)
	if err != nil {
		t.Fatalf("资产内容 base64 无效: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("资产内容不匹配，实际 %q", string(got))
	}
}
