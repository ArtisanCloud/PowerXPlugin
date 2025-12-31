package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/event"
)

func TestMetaBuilder_Build_RequiresTenantUUID(t *testing.T) {
	builder := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	builder.Now = func() time.Time { return time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC) }

	_, err := builder.Build("", "", "")
	require.Error(t, err)
}

func TestMetaBuilder_Build_DefaultsAndGeneratesIDs(t *testing.T) {
	builder := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	fixedNow := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	builder.Now = func() time.Time { return fixedNow }

	meta, err := builder.Build("00000000-0000-0000-0000-000000000001", "", "")
	require.NoError(t, err)

	require.NotEmpty(t, meta.TraceID)
	require.Equal(t, meta.TraceID, meta.RequestID)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", meta.TenantUUID)
	require.Equal(t, "com.powerx.plugins.base", meta.SourcePlugin)
	require.Equal(t, "v1", meta.PayloadVersion)
	require.True(t, meta.OccurredAt.Equal(fixedNow))
}

func TestMetaBuilder_Build_UsesProvidedIDs(t *testing.T) {
	builder := event.NewMetaBuilder("com.powerx.plugins.base", "v1")

	meta, err := builder.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-1")
	require.NoError(t, err)
	require.Equal(t, "req-1", meta.RequestID)
	require.Equal(t, "trace-1", meta.TraceID)
}
