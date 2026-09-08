package customerfw

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/stretchr/testify/require"
)

type externalIdentityInvokerStub struct{ request gateway.InvokeRequest }

func (s *externalIdentityInvokerStub) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.request = req
	return &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"customer_uuid": "11111111-1111-1111-1111-111111111111", "membership_uuid": "22222222-2222-2222-2222-222222222222", "display_name": "Customer"}}}}, nil
}

func TestExternalIdentityResolverUsesStrictCoreInternalContract(t *testing.T) {
	stub := &externalIdentityInvokerStub{}
	client, err := NewExternalIdentityResolver(ExternalIdentityResolverConfig{Invoker: stub})
	require.NoError(t, err)
	result, err := client.Resolve(context.Background(), ResolveExternalIdentityRequest{ProviderSubject: "shop:123:customer:456", DisplayName: "Customer"})
	require.NoError(t, err)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", result.CustomerUUID)
	require.Equal(t, CapabilityCustomerExternalIdentitiesResolve, stub.request.CapabilityID)
	require.Equal(t, "core_internal", stub.request.PreferredProtocol)
	require.Empty(t, stub.request.TenantUUID, "Core must derive tenant scope from the service credential")
	payload := stub.request.Payload.(map[string]any)
	require.Equal(t, "INVOKE", payload["method"])
	require.Equal(t, "core://customer/external-identities/resolve", payload["endpoint"])
	require.Equal(t, map[string]string{"provider_subject": "shop:123:customer:456", "display_name": "Customer"}, payload["body"])
}
