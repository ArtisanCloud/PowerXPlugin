package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/event_bridge"
)

func TestLocalEmitter_QueueFull_RecordsDropMetric(t *testing.T) {
	ebmetrics.Reset()

	factory, err := fweventbridge.NewFactory(fweventbridge.Config{
		Enabled:         false,
		Mode:            "local",
		FallbackToLocal: true,
		LocalQueueSize:  1,
	})
	require.NoError(t, err)
	factory.WithMetrics(bridgeRecorder{})

	emitter, err := factory.NewEmitter()
	require.NoError(t, err)

	ev := event.Event{
		Topic: "_topic.template.update",
		Meta: event.Meta{
			TenantUUID:     "00000000-0000-0000-0000-000000000001",
			RequestID:      "req-drop-1",
			TraceID:        "trace-drop-1",
			SourcePlugin:   "com.powerx.plugins.base",
			OccurredAt:     time.Now().UTC(),
			PayloadVersion: "v1",
		},
		Payload: json.RawMessage(`{"channel_id":"c1","status":"ok"}`),
	}

	require.NoError(t, emitter.Emit(context.Background(), ev))
	require.NoError(t, emitter.Emit(context.Background(), ev))

	var buf bytes.Buffer
	ebmetrics.RenderPrometheus(&buf)
	body := buf.String()
	require.Contains(t, body, "plugin_event_bridge_drop_total{plugin_id=\"com.powerx.plugins.base\",reason=\"queue_full\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"_topic.template.update\"} 1")
}
