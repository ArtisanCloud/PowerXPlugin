package integration

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

type taskBusProviderStub struct {
	emitter fweventbridge.Emitter
	calls   int32
}

func (p *taskBusProviderStub) NewEmitter() (fweventbridge.Emitter, error) {
	atomic.AddInt32(&p.calls, 1)
	return p.emitter, nil
}

func TestEventBridge_TaskBusMode_ProviderWiring(t *testing.T) {
	factory, err := fweventbridge.NewFactory(fweventbridge.Config{
		Enabled:         true,
		Mode:            "taskbus",
		FallbackToLocal: false,
		LocalQueueSize:  10,
	})
	require.NoError(t, err)

	stub := fweventbridge.NewTaskBusStub()
	provider := &taskBusProviderStub{emitter: stub}

	var consumed int32
	stub.Subscribe("_topic.template.update", func(ctx context.Context, ev event.Event) error {
		atomic.AddInt32(&consumed, 1)
		return nil
	})

	factory.WithTaskBusProvider(fweventbridge.NewTaskBusEmitterAdapter(provider))
	emitter, err := factory.NewEmitter()
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&provider.calls))

	mb := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	meta, err := mb.Build("00000000-0000-0000-0000-000000000001", "req-provider-1", "trace-provider-1")
	require.NoError(t, err)

	err = emitter.Emit(context.Background(), event.Event{
		Topic:   "_topic.template.update",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","status":"ok"}`),
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&consumed))
}
