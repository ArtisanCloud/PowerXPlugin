package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestFacadeEmitUsesRequestIDAsTraceFallback(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })

	ctx := WithRuntime(context.Background(), Fields{
		"request_id":   "req-123",
		FieldComponent: "middleware.request",
	})
	FromContext(ctx).Emit("info", "trace fallback test", Fields{
		FieldPluginID: "com.powerx.plugins.base",
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log json: %v", err)
	}
	if payload["trace_id"] != "req-123" {
		t.Fatalf("expected trace fallback req-123, got: %#v", payload["trace_id"])
	}
	if payload["component"] != "middleware.request" {
		t.Fatalf("expected component field, got: %#v", payload["component"])
	}
}

func TestFacadeInfoBuildsLabelWhitelistAndBlocksHighCardinality(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })

	ctx := WithRuntime(context.Background(), Fields{
		FieldComponent: "agent",
	})
	FromContext(ctx).Info("session message processed", Entry{
		Fields: Fields{
			"tenant_uuid": "tenant-001",
			"session_id":  "sess-123",
			"request_id":  "req-123",
		},
		Context: Fields{
			"module":    "agent",
			"biz_scene": "chat_session",
			"user_id":   "1001",
			"ignored":   "x",
		},
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log json: %v", err)
	}
	labelsAny, ok := payload["labels"]
	if !ok {
		t.Fatalf("expected labels object, got: %#v", payload)
	}
	labels, ok := labelsAny.(map[string]any)
	if !ok {
		t.Fatalf("labels should be object, got: %#v", labelsAny)
	}
	if labels["module"] != "agent" {
		t.Fatalf("expected module label=agent, got: %#v", labels["module"])
	}
	if _, exists := labels["plugin_id"]; exists {
		t.Fatalf("plugin_id should not be a label: %#v", labels)
	}
	if labels["level"] != "info" {
		t.Fatalf("expected level label info, got: %#v", labels["level"])
	}
	if _, exists := labels["user_id"]; exists {
		t.Fatalf("user_id should not be a label: %#v", labels)
	}
	if _, exists := labels["biz_scene"]; exists {
		t.Fatalf("biz_scene should not be a label: %#v", labels)
	}
	if payload["session_id"] != "sess-123" {
		t.Fatalf("expected session_id in body, got: %#v", payload["session_id"])
	}
}
