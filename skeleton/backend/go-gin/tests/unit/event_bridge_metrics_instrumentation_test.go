package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/event_bridge"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/eventbridge"
)

func TestEventBridge_MetricsInstrumentation_EmitAndConsume(t *testing.T) {
	ebmetrics.Reset()

	factory, err := fweventbridge.NewFactory(fweventbridge.Config{
		Enabled:         false,
		Mode:            "local",
		FallbackToLocal: true,
		LocalQueueSize:  10,
	})
	require.NoError(t, err)
	factory.WithMetrics(bridgeRecorder{})
	emitter, err := factory.NewEmitter()
	require.NoError(t, err)

	mb := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	mb.Now = func() time.Time { return time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC) }
	meta, err := mb.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-1")
	require.NoError(t, err)

	ev := event.Event{
		Topic:   "powerx.channel.master.credential_inspection.v1",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	}

	require.NoError(t, emitter.Emit(context.Background(), ev))

	dispatcher := fweventbridge.NewDispatcher(fweventbridge.NewIdempotencyFilter(100)).WithMetrics(bridgeRecorder{})
	dispatcher.Register(ev.Topic, func(ctx context.Context, ev event.Event) error { return nil })
	require.NoError(t, dispatcher.Dispatch(context.Background(), ev))

	var buf bytes.Buffer
	ebmetrics.RenderPrometheus(&buf)
	out := buf.String()

	require.True(t, strings.Contains(out, "plugin_event_bridge_emit_total{"), out)
	require.True(t, strings.Contains(out, "plugin_event_bridge_consume_total{"), out)
	require.True(t, strings.Contains(out, "plugin_event_bridge_latency_ms{"), out)
}
