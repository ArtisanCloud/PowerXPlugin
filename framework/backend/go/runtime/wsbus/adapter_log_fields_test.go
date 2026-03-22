package wsbus

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

type fakeInnerPublisher struct {
	result PublishResult
}

func (f *fakeInnerPublisher) Publish(_ context.Context, _ string, _ any, _ PublishOptions) PublishResult {
	return f.result
}

func TestAdapterFailureLogContainsRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	pub := NewAdapter(&fakeInnerPublisher{
		result: FailureResult(ErrorCodeHostPublishFailed, "upstream down"),
	}, "", logger)

	res := pub.Publish(context.Background(), "powerx.channel.master.credential_inspection.v1", map[string]any{"k": "v"}, PublishOptions{
		TenantUUID: "tenant-1",
		TraceID:    "trace-1",
	})
	if res.OK {
		t.Fatalf("expected failed result")
	}

	out := buf.String()
	for _, key := range []string{
		`"trace_id":"trace-1"`,
		`"task_id":"trace-1"`,
		`"tenant_uuid":"tenant-1"`,
		`"tenant_key":"tenant-1"`,
		`"subscriber_id":"wsbus.adapter"`,
		`"topic":"powerx.channel.master.credential_inspection.v1"`,
		`"status":"failed"`,
	} {
		if !strings.Contains(out, key) {
			t.Fatalf("expected log to contain %s, got %s", key, out)
		}
	}
}
