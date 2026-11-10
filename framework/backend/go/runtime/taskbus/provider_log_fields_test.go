package taskbus

import (
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHostFactoryCannotFallbackOrMirror(t *testing.T) {
	for _, mode := range []string{"taskbus", "dual"} {
		factory, err := eventbridge.NewFactory(eventbridge.Config{Enabled: true, Mode: mode, FallbackToLocal: true})
		require.NoError(t, err)
		factory.WithTaskBusProvider(eventbridge.NewTaskBusEmitterAdapter(NewHostProvider(HostProviderConfig{})))
		_, err = factory.NewEmitter()
		require.True(t, errors.Is(err, eventbridge.ErrReliableHostRequired))
		cfg := fixtureHostConfig("http://core.invalid")
		cfg.AuthScheme = "apikey"
		cfg.APIKey = "fixture"
		factory.WithTaskBusProvider(eventbridge.NewTaskBusEmitterAdapter(NewHostProvider(cfg)))
		_, err = factory.NewEmitter()
		require.True(t, errors.Is(err, eventbridge.ErrReliableHostRequired))
	}
}

func TestHostProviderRejectsLegacyPublishRoutes(t *testing.T) {
	cfg := fixtureHostConfig("http://core.invalid")
	cfg.AuthScheme = "apikey"
	cfg.APIKey = "fixture"
	cfg.PublishPath = "/admin/runtime/ws-bus/publish"
	_, err := NewHostProvider(cfg).NewEmitter()
	require.Error(t, err)
}
