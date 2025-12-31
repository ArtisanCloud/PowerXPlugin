package event_bridge

import (
	"bytes"
	"strings"
	"testing"
)

func TestRecordersAndRender(t *testing.T) {
	Reset()

	RecordEmit("com.powerx.plugins.base", "00000000-0000-0000-0000-000000000001", "powerx.channel.master.credential_inspection.v1", "success")
	RecordEmit("com.powerx.plugins.base", "00000000-0000-0000-0000-000000000001", "powerx.channel.master.credential_inspection.v1", "error")
	RecordConsume("com.powerx.plugins.base", "00000000-0000-0000-0000-000000000001", "powerx.channel.master.credential_inspection.v1", "success")
	ObserveLatencyMs("com.powerx.plugins.base", "00000000-0000-0000-0000-000000000001", "powerx.channel.master.credential_inspection.v1", "emit", 12)

	var buf bytes.Buffer
	RenderPrometheus(&buf)
	body := buf.String()

	checks := []string{
		"plugin_event_bridge_emit_total{plugin_id=\"com.powerx.plugins.base\",result=\"success\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1",
		"plugin_event_bridge_emit_total{plugin_id=\"com.powerx.plugins.base\",result=\"error\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1",
		"plugin_event_bridge_consume_total{plugin_id=\"com.powerx.plugins.base\",result=\"success\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1",
		"plugin_event_bridge_latency_ms{op=\"emit\",plugin_id=\"com.powerx.plugins.base\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 12",
	}

	for _, snippet := range checks {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", snippet, body)
		}
	}
}
