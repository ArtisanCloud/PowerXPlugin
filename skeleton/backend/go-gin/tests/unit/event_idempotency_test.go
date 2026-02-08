package unit

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

func TestIdempotencyFilter_SkipsDuplicatesByTopicTenantTrace(t *testing.T) {
	filter := fweventbridge.NewIdempotencyFilter(100)
	dispatcher := fweventbridge.NewDispatcher(filter)

	var handled int32
	dispatcher.Register("powerx.channel.master.credential_inspection.v1", func(ctx context.Context, ev event.Event) error {
		atomic.AddInt32(&handled, 1)
		return nil
	})

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

	require.NoError(t, dispatcher.Dispatch(context.Background(), ev))
	require.NoError(t, dispatcher.Dispatch(context.Background(), ev))
	require.Equal(t, int32(1), atomic.LoadInt32(&handled))
}

func TestIdempotencyFilter_AllowsWhenTraceIDMissing(t *testing.T) {
	filter := fweventbridge.NewIdempotencyFilter(100)
	dispatcher := fweventbridge.NewDispatcher(filter)

	var handled int32
	dispatcher.Register("powerx.channel.master.credential_inspection.v1", func(ctx context.Context, ev event.Event) error {
		atomic.AddInt32(&handled, 1)
		return nil
	})

	ev := event.Event{
		Topic: "powerx.channel.master.credential_inspection.v1",
		Meta: event.Meta{
			TenantUUID:     "00000000-0000-0000-0000-000000000001",
			RequestID:      "req-1",
			TraceID:        "",
			SourcePlugin:   "com.powerx.plugins.base",
			OccurredAt:     time.Now().UTC(),
			PayloadVersion: "v1",
		},
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	}

	require.NoError(t, dispatcher.Dispatch(context.Background(), ev))
	require.NoError(t, dispatcher.Dispatch(context.Background(), ev))
	require.Equal(t, int32(2), atomic.LoadInt32(&handled))
}
