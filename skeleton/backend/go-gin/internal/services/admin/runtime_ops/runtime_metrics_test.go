package runtime_ops

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/event_bridge"
)

func TestMetricsHTTPHandler_ExposesCoreSeries(t *testing.T) {
	resetMetrics()
	ebmetrics.Reset()

	IncRequest("plugin.demo", "bootstrap", 120*time.Millisecond, nil)
	SetQuotaUsage("plugin.demo", "tenant", "tenant-1", 0.75)
	AddCost("plugin.demo", "tenant-1", 3.5)
	SetMCPSessions("plugin.demo", 2)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	MetricsHTTPHandler().ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("expected text/plain content type, got %q", got)
	}

	body := rec.Body.String()

	expectedSnippets := []string{
		`powerx_plugin_request_total{capability="bootstrap",plugin_id="plugin.demo"} 1`,
		`powerx_plugin_quota_usage{plugin_id="plugin.demo",scope="tenant",scope_ref="tenant-1"} 0.75`,
		`powerx_plugin_cost_total{plugin_id="plugin.demo",tenant_uuid="tenant-1"} 3.5`,
		`powerx_mcp_sessions_total{plugin_id="plugin.demo"} 2`,
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected metrics output to contain %q\nbody:\n%s", snippet, body)
		}
	}
}

func TestMetricsHTTPHandler_SchedulerManualCronParitySeries(t *testing.T) {
	resetMetrics()
	ebmetrics.Reset()

	topic := "powerx.runtime.scheduler.triggered.v1"
	pluginID := "com.powerx.plugins.base"
	tenantID := "00000000-0000-0000-0000-000000000001"

	// 手动触发与调度触发统一记入同一 topic/result 指标序列。
	ebmetrics.RecordEmit(pluginID, tenantID, topic, "success") // manual
	ebmetrics.RecordEmit(pluginID, tenantID, topic, "success") // cron
	ebmetrics.ObserveLatencyMs(pluginID, tenantID, topic, "emit", 11)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	MetricsHTTPHandler().ServeHTTP(rec, req)

	body := rec.Body.String()
	expected := `plugin_event_bridge_emit_total{plugin_id="com.powerx.plugins.base",result="success",tenant_uuid="00000000-0000-0000-0000-000000000001",topic="powerx.runtime.scheduler.triggered.v1"} 2`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected parity metric series %q\nbody:\n%s", expected, body)
	}
	if !strings.Contains(body, `plugin_event_bridge_latency_ms{op="emit",plugin_id="com.powerx.plugins.base",tenant_uuid="00000000-0000-0000-0000-000000000001",topic="powerx.runtime.scheduler.triggered.v1"} 11`) {
		t.Fatalf("expected latency series for scheduler topic\nbody:\n%s", body)
	}
}
