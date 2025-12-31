package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/event"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/event_bridge"
)

func TestEventBridge_FallbackToLocalWhenTaskBusUnavailable(t *testing.T) {
	cfg := config.EventBridgeConfig{
		Enabled:         true,
		Mode:            "taskbus",
		FallbackToLocal: true,
		LocalQueueSize:  10,
		SourcePlugin:    "com.powerx.plugins.base",
		PayloadVersion:  "v1",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	factory := event_bridge.NewFactory(cfg, logrus.NewEntry(logger)).
		WithTaskBusProvider(func(logger *logrus.Entry) (event_bridge.Emitter, error) {
			return nil, errors.New("taskbus unavailable")
		})

	emitter, err := factory.NewEmitter()
	require.NoError(t, err)

	meta := event.NewMetaBuilder(cfg.SourcePlugin, cfg.PayloadVersion)
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

	drainer, ok := emitter.(interface{ Drain() []event.Event })
	require.True(t, ok, "expected fallback emitter to support Drain()")

	drained := drainer.Drain()
	require.Len(t, drained, 1)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", drained[0].Meta.TenantUUID)
	require.Equal(t, "trace-1", drained[0].Meta.TraceID)
	require.True(t, drained[0].Meta.OccurredAt.Before(time.Now().Add(1*time.Minute)))
}
