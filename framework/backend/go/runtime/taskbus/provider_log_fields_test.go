package taskbus

import (
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHostFactoryPropagatesProviderFailure(t *testing.T) {
	factory, err := eventbridge.NewFactory(eventbridge.Config{Enabled: true, RuntimeMode: provider.RuntimeMode{ProviderMode: provider.ModeLocal, ProxyEnabled: true}})
	require.NoError(t, err)
	factory.WithTaskBusProvider(eventbridge.NewTaskBusEmitterAdapter(NewHostProvider(HostProviderConfig{})))
	emitter, err := factory.NewEmitter()
	require.True(t, errors.Is(err, eventbridge.ErrReliableHostRequired))
	require.Nil(t, emitter)
}

func TestHostProviderRejectsLegacyPublishRoutes(t *testing.T) {
	cfg := fixtureHostConfig("http://core.invalid")
	cfg.AuthScheme = "apikey"
	cfg.APIKey = "fixture"
	cfg.PublishPath = "/admin/runtime/ws-bus/publish"
	_, err := NewHostProvider(cfg).NewEmitter()
	require.Error(t, err)
}
