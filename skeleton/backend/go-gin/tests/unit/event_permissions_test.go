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

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

func TestLoadEventPermissionsFromManifest_EnforcedWhenEventsPresent(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
id: com.powerx.plugins.base
events:
  publish:
    - _topic.template.update
  subscribe: []
`), 0o644))

	perms, err := security.LoadEventPermissionsFromManifest(manifestPath, logrus.NewEntry(logrus.StandardLogger()))
	require.NoError(t, err)
	require.True(t, perms.Enforced())
	require.True(t, perms.CanPublish("_topic.template.update"))
	require.False(t, perms.CanPublish("_topic.template.validate.completed"))
}

func TestLoadEventPermissionsFromManifest_EnforcedWhenTopicsPresent(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
id: com.powerx.plugins.base
events:
  topics:
    - key: _topic.template.update
      actions: [publish, subscribe]
    - key: _topic.template.batch_clone.completed
      actions: [publish]
`), 0o644))

	perms, err := security.LoadEventPermissionsFromManifest(manifestPath, logrus.NewEntry(logrus.StandardLogger()))
	require.NoError(t, err)
	require.True(t, perms.Enforced())
	require.True(t, perms.CanPublish("_topic.template.update"))
	require.True(t, perms.CanSubscribe("_topic.template.update"))
	require.True(t, perms.CanPublish("_topic.template.batch_clone.completed"))
	require.False(t, perms.CanSubscribe("_topic.template.batch_clone.completed"))
	require.Equal(t, []string{"_topic.template.batch_clone.completed", "_topic.template.update"}, perms.Topics())
}

func TestPermissionedEmitter_DeniesWhenNotAllowed(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
id: com.powerx.plugins.base
events:
  publish:
    - _topic.template.update
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
		Topic:   "_topic.template.validate.completed",
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
    - _topic.template.update
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
		Topic:   "_topic.template.update",
		Meta:    meta,
		Payload: json.RawMessage(`{"channel_id":"c1","credential_type":"api_key","status":"ok"}`),
	})
	require.NoError(t, err)

	drained := inner.Drain()
	require.Len(t, drained, 1)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", drained[0].Meta.TenantUUID)
	require.Equal(t, "trace-1", drained[0].Meta.TraceID)
}
