package knowledge

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProvisioningIsNotImplicitlyAvailableOnLocalOrUndeclaredClients(t *testing.T) {
	provider := NewDelegatedProvider(DelegatedProviderConfig{Client: fakeDelegatedClient{}})
	_, err := provider.CreateSpace(context.Background(), CreateSpaceInput{})
	require.Equal(t, CodeUnsupportedCapability, CodeOf(err))
	var local any = NewMockProvider()
	_, ok := local.(SpaceProvisioningProvider)
	require.False(t, ok)
}
