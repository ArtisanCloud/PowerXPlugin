package taskbus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
)

type fakePublisher struct {
	result wsbus.PublishResult
}

func (f *fakePublisher) Publish(_ context.Context, _ string, _ any, _ wsbus.PublishOptions) wsbus.PublishResult {
	return f.result
}

type captureLogger struct {
	records *[]runtimelogging.Fields
}

func (l *captureLogger) WithFields(fields runtimelogging.Fields) runtimelogging.Logger {
	if l.records != nil {
		*l.records = append(*l.records, fields)
	}
	return l
}
func (l *captureLogger) Info(_ string)  {}
func (l *captureLogger) Warn(_ string)  {}
func (l *captureLogger) Error(_ string) {}

func TestHostEmitter_LogFieldsContainRequiredKeys(t *testing.T) {
	records := []runtimelogging.Fields{}
	logger := &captureLogger{records: &records}
	emitter := &hostEmitter{
		publisher:   &fakePublisher{result: wsbus.SuccessResult()},
		metaBuilder: event.NewMetaBuilder("plugin.test", "v1"),
		logger:      logger,
	}
	err := emitter.Emit(context.Background(), event.Event{
		Topic: "powerx.channel.master.credential_inspection.v1",
		Meta: event.Meta{
			TenantUUID: "tenant-1",
			RequestID:  "req-1",
			TraceID:    "trace-1",
		},
		Payload: json.RawMessage(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	if len(records) == 0 {
		t.Fatalf("expected captured fields")
	}
	last := records[len(records)-1]
	for _, key := range []string{
		runtimelogging.FieldTraceID,
		runtimelogging.FieldTaskID,
		runtimelogging.FieldTenantUUID,
		runtimelogging.FieldTenantKey,
		runtimelogging.FieldSubscriber,
		runtimelogging.FieldTopic,
		runtimelogging.FieldStatus,
	} {
		if _, ok := last[key]; !ok {
			t.Fatalf("missing key %s", key)
		}
	}
}
