package logger

import "testing"

func TestWithRuntimeFields(t *testing.T) {
	entry := WithRuntimeFields("plugin.demo", "tenant-1", "trace-abc", "component", Fields{"foo": "bar"})
	fields := entry.Data
	if fields["plugin_id"] != "plugin.demo" {
		t.Fatalf("plugin_id not set")
	}
	if fields["tenant_uuid"] != "tenant-1" {
		t.Fatalf("tenant_uuid not set")
	}
	if fields["tenant_key"] != "tenant-1" {
		t.Fatalf("tenant_key not set")
	}
	if fields["trace_id"] != "trace-abc" {
		t.Fatalf("trace_id not set")
	}
	if fields["component"] != "component" {
		t.Fatalf("component not set")
	}
	if fields["mode"] != "unknown" {
		t.Fatalf("mode default not set")
	}
	if fields["user_id"] != "" {
		t.Fatalf("user_id default not set")
	}
	if fields["permission"] != "" {
		t.Fatalf("permission default not set")
	}
	if fields["foo"] != "bar" {
		t.Fatalf("extra field missing")
	}
	if _, ok := fields["level"]; ok {
		t.Fatalf("level should be set by logger backend, not runtime fields")
	}
}
