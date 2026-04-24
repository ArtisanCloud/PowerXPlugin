package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
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

	out := buf.String()
	if !strings.Contains(out, `"trace_id":"req-123"`) {
		t.Fatalf("expected trace fallback, got: %s", out)
	}
	if !strings.Contains(out, `"component":"middleware.request"`) {
		t.Fatalf("expected component field, got: %s", out)
	}
}
