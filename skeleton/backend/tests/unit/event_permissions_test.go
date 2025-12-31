package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/security"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/eventbridge"
)

func TestLoadEventPermissionsFromManifest_EnforcedWhenEventsPresent(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
id: com.powerx.plugins.base
events:
  publish:
    - powerx.channel.master.credential_inspection.v1
  subscribe: []
`), 0o644))

	perms, err := security.LoadEventPermissionsFromManifest(manifestPath, logrus.NewEntry(logrus.StandardLogger()))
	require.NoError(t, err)
	require.True(t, perms.Enforced())
	require.True(t, perms.CanPublish("powerx.channel.master.credential_inspection.v1"))
	require.False(t, perms.CanPublish("powerx.channel.master.kpi_refreshed.v1"))
}

func TestPermissionedEmitter_DeniesWhenNotAllowed(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
id: com.powerx.plugins.base
events:
  publish:
    - powerx.channel.master.credential_inspection.v1
  subscribe: []
`), 0o644))

	perms, err := security.LoadEventPermissionsFromManifest(manifestPath, logrus.NewEntry(logrus.StandardLogger()))
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	inner := fweventbridge.NewLocalEmitter(10)
	emitter := security.NewPermissionedEmitter(inner, perms, logrus.NewEntry(logger))

	mb := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	meta, err := mb.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-1")
	require.NoError(t, err)

	err = emitter.Emit(context.Background(), event.Event{
		Topic:   "powerx.channel.master.kpi_refreshed.v1",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","window":{"start_at":"2025-12-30T00:00:00Z","end_at":"2025-12-31T00:00:00Z"},"metrics":{"a":1}}`),
	})
	require.ErrorIs(t, err, security.ErrEventPermissionDenied)

	require.Empty(t, inner.Drain())
}

func TestPermissionedEmitter_AllowsWhenAllowed(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
id: com.powerx.plugins.base
events:
  publish:
    - powerx.channel.master.credential_inspection.v1
  subscribe: []
`), 0o644))

	perms, err := security.LoadEventPermissionsFromManifest(manifestPath, logrus.NewEntry(logrus.StandardLogger()))
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	inner := fweventbridge.NewLocalEmitter(10)
	emitter := security.NewPermissionedEmitter(inner, perms, logrus.NewEntry(logger))

	mb := event.NewMetaBuilder("com.powerx.plugins.base", "v1")
	meta, err := mb.Build("00000000-0000-0000-0000-000000000001", "req-1", "trace-1")
	require.NoError(t, err)

	err = emitter.Emit(context.Background(), event.Event{
		Topic:   "powerx.channel.master.credential_inspection.v1",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	})
	require.NoError(t, err)

	drained := inner.Drain()
	require.Len(t, drained, 1)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", drained[0].Meta.TenantUUID)
	require.Equal(t, "trace-1", drained[0].Meta.TraceID)
}
