package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

func TestEventBridge_FallbackToLocalWhenTaskBusUnavailable(t *testing.T) {
	factory, err := fweventbridge.NewFactory(fweventbridge.Config{
		Enabled:         true,
		Mode:            "taskbus",
		FallbackToLocal: true,
		LocalQueueSize:  10,
	})
	require.NoError(t, err)
	factory.WithTaskBusProvider(func() (fweventbridge.Emitter, error) { return nil, errors.New("taskbus unavailable") })
	emitter, err := factory.NewEmitter()
	require.NoError(t, err)

	meta := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	evtMeta, err := meta.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-1")
	require.NoError(t, err)

	require.NoError(t, emitter.Emit(context.Background(), event.Event{
		Topic: "powerx.channel.master.credential_inspection.v1",
		Meta:  evtMeta,
		Payload: json.RawMessage(`{
  "channel_id": "c1",
  "credential_type": "api_key",
  "status": "ok",
  "observed_at_epoch": 1767052800
}`),
	}))

	drained := fweventbridge.Drain(emitter)
	require.Len(t, drained, 1)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", drained[0].Meta.TenantUUID)
	require.Equal(t, "trace-1", drained[0].Meta.TraceID)
	require.True(t, drained[0].Meta.OccurredAt.Before(time.Now().Add(1*time.Minute)))
}
