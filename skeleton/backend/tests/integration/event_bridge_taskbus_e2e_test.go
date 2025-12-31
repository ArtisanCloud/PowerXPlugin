package integration

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/event"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/event_bridge"
)

func TestEventBridge_TaskBusMode_E2EWithStub(t *testing.T) {
	cfg := config.EventBridgeConfig{
		Enabled:         true,
		Mode:            "taskbus",
		FallbackToLocal: false,
		LocalQueueSize:  10,
		SourcePlugin:    "com.powerx.plugins.base",
		PayloadVersion:  "v1",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	stub := event_bridge.NewTaskBusStub(logrus.NewEntry(logger))
	var consumed int32
	stub.Subscribe("powerx.channel.master.credential_inspection.v1", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&consumed, 1)
		return nil
	})

	factory := event_bridge.NewFactory(cfg, logrus.NewEntry(logger)).
		WithTaskBusProvider(func(logger *logrus.Entry) (event_bridge.Emitter, error) {
			return stub, nil
		})

	emitter, err := factory.NewEmitter()
	require.NoError(t, err)

	mb := event.NewMetaBuilder(cfg.SourcePlugin, cfg.PayloadVersion)
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
