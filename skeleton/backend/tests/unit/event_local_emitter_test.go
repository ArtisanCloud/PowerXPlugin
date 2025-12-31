package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/event"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/event_bridge"
)

func TestLocalEmitter_Emit_DoesNotPanicWhenQueueFull(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	emitter := event_bridge.NewLocalEmitter(1, logrus.NewEntry(logger))

	ev := event.Event{
		Topic: "powerx.channel.master.credential_inspection.v1",
		Meta: event.Meta{
			TenantUUID:     "00000000-0000-0000-0000-000000000001",
			RequestID:      "req-1",
			TraceID:        "trace-1",
			SourcePlugin:   "com.powerx.plugins.base",
			OccurredAt:     time.Now().UTC(),
			PayloadVersion: "v1",
		},
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	}

	require.NoError(t, emitter.Emit(context.Background(), ev))
	require.NoError(t, emitter.Emit(context.Background(), ev))

	drained := emitter.Drain()
	require.Len(t, drained, 1)
}
