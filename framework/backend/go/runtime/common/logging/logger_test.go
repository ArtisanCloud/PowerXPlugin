package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

type fakeLogrusEntry struct {
	fields map[string]any
	msgs   []string
}

func (f *fakeLogrusEntry) WithFields(fields map[string]any) LogrusEntry {
	n := &fakeLogrusEntry{
		fields: map[string]any{},
		msgs:   append([]string{}, f.msgs...),
	}
	for k, v := range f.fields {
		n.fields[k] = v
	}
	for k, v := range fields {
		n.fields[k] = v
	}
	return n
}

func (f *fakeLogrusEntry) Info(args ...any)  { f.msgs = append(f.msgs, "info") }
func (f *fakeLogrusEntry) Warn(args ...any)  { f.msgs = append(f.msgs, "warn") }
func (f *fakeLogrusEntry) Error(args ...any) { f.msgs = append(f.msgs, "error") }

func TestApplyRuntimeFallback(t *testing.T) {
	in := Fields{
		FieldTraceID: "trace-1",
	}
	got := ApplyRuntimeFallback(in)
	if got[FieldTaskID] != FallbackUnknown {
		t.Fatalf("expected %s=%s", FieldTaskID, FallbackUnknown)
	}
	if got[FieldSubscriber] != FallbackUnknown {
		t.Fatalf("expected %s=%s", FieldSubscriber, FallbackUnknown)
	}
	if got[FieldStatus] != StatusSkipped {
		t.Fatalf("expected %s=%s", FieldStatus, StatusSkipped)
	}
	if got[FieldReason] != ReasonMissingContext {
		t.Fatalf("expected %s=%s", FieldReason, ReasonMissingContext)
	}
	if got[FieldTenantUUID] != FallbackUnknown {
		t.Fatalf("expected %s=%s", FieldTenantUUID, FallbackUnknown)
	}
	if got[FieldTenantKey] != FallbackUnknown {
		t.Fatalf("expected %s=%s", FieldTenantKey, FallbackUnknown)
	}
}

func TestSlogAdapterWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	adapter := NewSlogAdapter(logger).WithFields(Fields{
		FieldTraceID: "trace-abc",
		FieldTopic:   "topic.demo",
	})
	adapter.Info("hello")
	out := buf.String()
	if !strings.Contains(out, `"trace_id":"trace-abc"`) {
		t.Fatalf("expected trace_id in slog output, got: %s", out)
	}
	if !strings.Contains(out, `"topic":"topic.demo"`) {
		t.Fatalf("expected topic in slog output, got: %s", out)
	}
}

func TestLogrusAdapterWithFieldsAndWarn(t *testing.T) {
	base := &fakeLogrusEntry{fields: map[string]any{}}
	adapter := NewLogrusAdapter(base).WithFields(Fields{
		FieldTenantUUID: "tenant-1",
	})
	l, ok := adapter.(*LogrusAdapter)
	if !ok || l.entry == nil {
		t.Fatalf("expected logrus adapter with entry")
	}
	l.Warn("warn")

	fe, ok := l.entry.(*fakeLogrusEntry)
	if !ok {
		t.Fatalf("expected fake entry")
	}
	if fe.fields[FieldTenantUUID] != "tenant-1" {
		t.Fatalf("expected tenant field")
	}
	if len(fe.msgs) == 0 || fe.msgs[len(fe.msgs)-1] != "warn" {
		t.Fatalf("expected warn call")
	}
}

func TestRuntimeFieldsContext(t *testing.T) {
	ctx := WithRuntime(context.Background(), Fields{
		FieldTraceID: "trace-ctx",
	})
	fields := RuntimeFieldsFromContext(ctx)
	if fields[FieldTraceID] != "trace-ctx" {
		t.Fatalf("expected trace_id from context")
	}
}

