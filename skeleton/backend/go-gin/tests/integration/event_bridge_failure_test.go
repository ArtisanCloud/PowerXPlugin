package integration

import (
	"errors"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEventBridgeProviderFailureStopsAssembly(t *testing.T) {
	factory, err := fweventbridge.NewFactory(fweventbridge.Config{Enabled: true, RuntimeMode: provider.RuntimeMode{ProviderMode: provider.ModeLocal}})
	require.NoError(t, err)
	unavailable := errors.New("FIXTURE_TASKBUS_UNAVAILABLE")
	factory.WithTaskBusProvider(func() (fweventbridge.Emitter, error) { return nil, unavailable })
	emitter, err := factory.NewEmitter()
	require.ErrorIs(t, err, unavailable)
	require.Nil(t, emitter)
}
