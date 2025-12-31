package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/event"
	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/event_bridge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/event_bridge"
)

func TestEventBridge_Emitter_RecordsMetricsOnSuccessAndFailure(t *testing.T) {
	ebmetrics.Reset()

	cfg := config.EventBridgeConfig{
		Enabled:        false,
		LocalQueueSize: 10,
		SourcePlugin:   "com.powerx.plugins.base",
		PayloadVersion: "v1",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	emitter, err := event_bridge.NewFactory(cfg, logrus.NewEntry(logger)).NewEmitter()
	require.NoError(t, err)

	mb := event.NewMetaBuilder(cfg.SourcePlugin, cfg.PayloadVersion)
	meta, err := mb.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-emit-1")
	require.NoError(t, err)

	ev := event.Event{
		Topic:   "powerx.channel.master.credential_inspection.v1",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	}
	require.NoError(t, emitter.Emit(context.Background(), ev))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, emitter.Emit(ctx, ev))

	var buf bytes.Buffer
	ebmetrics.RenderPrometheus(&buf)
	body := buf.String()

	require.Contains(t, body, "plugin_event_bridge_emit_total{plugin_id=\"com.powerx.plugins.base\",result=\"success\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1")
	require.Contains(t, body, "plugin_event_bridge_emit_total{plugin_id=\"com.powerx.plugins.base\",result=\"error\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1")
	require.True(t, strings.Contains(body, "plugin_event_bridge_latency_ms{op=\"emit\",plugin_id=\"com.powerx.plugins.base\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"}"))
}

func TestEventBridge_Dispatcher_RecordsMetricsOnSuccessAndFailure(t *testing.T) {
	ebmetrics.Reset()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	dispatcher := event_bridge.NewDispatcher(nil, logrus.NewEntry(logger))

	topic := event.Topic("powerx.channel.master.credential_inspection.v1")
	dispatcher.Register(topic, func(ctx context.Context, ev event.Event) error { return nil })
	dispatcher.Register(topic, func(ctx context.Context, ev event.Event) error { return errors.New("boom") })

	meta := event.Meta{
		TenantUUID:   "00000000-0000-0000-0000-000000000001",
		SourcePlugin: "com.powerx.plugins.base",
		TraceID:      "trace-consume-1",
		RequestID:    "req-1",
	}

	err := dispatcher.Dispatch(context.Background(), event.Event{
		Topic:   topic,
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1"}`),
	})
	require.Error(t, err)

	dispatcherSuccess := event_bridge.NewDispatcher(nil, logrus.NewEntry(logger))
	dispatcherSuccess.Register(topic, func(ctx context.Context, ev event.Event) error { return nil })
	require.NoError(t, dispatcherSuccess.Dispatch(context.Background(), event.Event{
		Topic: topic,
		Meta: event.Meta{
			TenantUUID:   meta.TenantUUID,
			SourcePlugin: meta.SourcePlugin,
			TraceID:      "trace-consume-2",
			RequestID:    "req-2",
		},
		Payload: json.RawMessage(`{"channel_id":"c1"}`),
	}))

	var buf bytes.Buffer
	ebmetrics.RenderPrometheus(&buf)
	body := buf.String()

	require.Contains(t, body, "plugin_event_bridge_consume_total{plugin_id=\"com.powerx.plugins.base\",result=\"error\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1")
	require.Contains(t, body, "plugin_event_bridge_consume_total{plugin_id=\"com.powerx.plugins.base\",result=\"success\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"} 1")
	require.True(t, strings.Contains(body, "plugin_event_bridge_latency_ms{op=\"consume\",plugin_id=\"com.powerx.plugins.base\",tenant_uuid=\"00000000-0000-0000-0000-000000000001\",topic=\"powerx.channel.master.credential_inspection.v1\"}"))
}
