package integration

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/eventbridge"
)

func TestEventBridge_TaskBusMode_E2EWithStub(t *testing.T) {
	factory, err := fweventbridge.NewFactory(fweventbridge.Config{
		Enabled:         true,
		Mode:            "taskbus",
		FallbackToLocal: false,
		LocalQueueSize:  10,
	})
	require.NoError(t, err)

	stub := fweventbridge.NewTaskBusStub()
	var consumed int32
	stub.Subscribe("powerx.channel.master.credential_inspection.v1", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&consumed, 1)
		return nil
	})

	factory.WithTaskBusProvider(func() (fweventbridge.Emitter, error) { return stub, nil })
	emitter, err := factory.NewEmitter()
	require.NoError(t, err)

	mb := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	meta, err := mb.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-1")
	require.NoError(t, err)

	err = emitter.Emit(context.Background(), event.Event{
		Topic:   "powerx.channel.master.credential_inspection.v1",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&consumed))
}
