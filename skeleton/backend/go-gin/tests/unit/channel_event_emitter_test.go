package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/channel"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

func TestChannelEventEmitter_EmitsWithIdempotencyKeyMaterial(t *testing.T) {
	local := fweventbridge.NewLocalEmitter(10)

	metaBuilder := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	metaBuilder.Now = func() time.Time { return time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC) }

	emitter := channel.NewChannelEventEmitter(metaBuilder, local)

	err := emitter.EmitCredentialInspection(context.Background(),
		"00000000-0000-0000-0000-000000000001",
		"",
		"",
		channel.CredentialInspectionPayload{
			ChannelID:      "c1",
			CredentialType: "api_key",
			Status:         "ok",
		},
	)
	require.NoError(t, err)

	evs := local.Drain()
	require.Len(t, evs, 1)
	require.Equal(t, channel.TopicCredentialInspectionV1, evs[0].Topic)
	require.NotEmpty(t, evs[0].Meta.TenantUUID)
	require.NotEmpty(t, evs[0].Meta.TraceID)
	require.NotEmpty(t, evs[0].Meta.RequestID)
}
