package customerfw

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/stretchr/testify/require"
)

type managementInvoker struct {
	request gateway.InvokeRequest
	payload any
}

func (s *managementInvoker) Invoke(_ context.Context, in gateway.InvokeRequest) (*gateway.Response, error) {
	s.request = in
	return &gateway.Response{Data: map[string]any{"payload": s.payload}}, nil
}
func TestIdentityManagementReadAndWriteUseSeparateTypedCapabilities(t *testing.T) {
	stub := &managementInvoker{payload: map[string]any{"found": false}}
	client, err := NewExternalIdentityClient(stub)
	require.NoError(t, err)
	result, err := client.Lookup(context.Background(), "shop:example:customer:1")
	require.NoError(t, err)
	require.False(t, result.Found)
	require.Equal(t, CapabilityCustomerExternalIdentitiesRead, stub.request.CapabilityID)
	require.Empty(t, stub.request.TenantUUID)
	payload := stub.request.Payload.(map[string]any)
	require.Equal(t, "core://customer/external-identities", payload["endpoint"])
	require.Equal(t, "lookup", payload["body"].(map[string]any)["operation"])
	item := ExternalIdentityItem{IdentityUUID: "11111111-1111-4111-8111-111111111111", CustomerUUID: "22222222-2222-4222-8222-222222222222", ProviderSubject: "shop:example:customer:1", Status: "active", Type: "person", PrimaryContactUUID: "33333333-3333-4333-8333-333333333333"}
	stub.payload = map[string]any{"item": item}
	bound, err := client.Bind(context.Background(), item.CustomerUUID, item.ProviderSubject)
	require.NoError(t, err)
	require.Equal(t, item, *bound)
	require.Equal(t, CapabilityCustomerExternalIdentitiesManage, stub.request.CapabilityID)
	created, err := client.CreateAndBind(context.Background(), item.ProviderSubject, ExternalIdentityCustomer{PrimaryEmail: "only@example.test"})
	require.NoError(t, err)
	require.Equal(t, item, *created)
	stub.payload = map[string]any{}
	_, err = client.Lookup(context.Background(), item.ProviderSubject)
	require.Error(t, err)
}
func TestEmailOnlyLabelDoesNotBecomeRealName(t *testing.T) {
	req := NormalizeBasicAccountLabels(CreateBasicAccountRequest{PrimaryEmail: "only@example.test"})
	require.Equal(t, "person", req.Type)
	require.Equal(t, "only@example.test", req.DisplayName)
	require.Empty(t, req.GivenName)
	require.Empty(t, req.FamilyName)
	require.Empty(t, req.Nickname)
}

func TestIdentitySubjectIncludesInstanceAndCompleteGID(t *testing.T) {
	subject := ShopifyExternalIdentitySubject("example.myshopify.com", "gid://shopify/Customer/42")
	require.Equal(t, "shop:example.myshopify.com:customer:gid://shopify/Customer/42", subject)
	require.True(t, ValidExternalIdentitySubject(subject))
	for _, invalid := range []string{"customer:42", "shop::customer:42", "shop: example:customer:42"} {
		require.False(t, ValidExternalIdentitySubject(invalid))
	}
}
