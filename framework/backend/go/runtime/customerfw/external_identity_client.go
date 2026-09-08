package customerfw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const CapabilityCustomerExternalIdentitiesResolve = "com.corex.customer.external_identities.resolve"

type ExternalIdentityResolverConfig struct {
	Invoker AdminGatewayInvoker
}

type ExternalIdentityResolver struct {
	invoker AdminGatewayInvoker
}

type ResolveExternalIdentityRequest struct {
	ProviderSubject string
	DisplayName     string
	RequestID       string
}

type ExternalIdentityResolution struct {
	CustomerUUID   string `json:"customer_uuid"`
	MembershipUUID string `json:"membership_uuid"`
	DisplayName    string `json:"display_name"`
}

func NewExternalIdentityResolver(cfg ExternalIdentityResolverConfig) (*ExternalIdentityResolver, error) {
	if cfg.Invoker == nil {
		return nil, errors.New("customer external identity: gateway invoker is required")
	}
	return &ExternalIdentityResolver{invoker: cfg.Invoker}, nil
}

func (c *ExternalIdentityResolver) Resolve(ctx context.Context, req ResolveExternalIdentityRequest) (*ExternalIdentityResolution, error) {
	if c == nil || c.invoker == nil {
		return nil, errors.New("customer external identity: gateway invoker is required")
	}
	if strings.TrimSpace(req.ProviderSubject) == "" || strings.TrimSpace(req.DisplayName) == "" {
		return nil, errors.New("customer external identity: provider_subject and display_name are required")
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID: CapabilityCustomerExternalIdentitiesResolve, PreferredProtocol: "core_internal", RequestID: strings.TrimSpace(req.RequestID),
		Payload: map[string]any{"method": "INVOKE", "endpoint": "core://customer/external-identities/resolve", "body": map[string]string{"provider_subject": strings.TrimSpace(req.ProviderSubject), "display_name": strings.TrimSpace(req.DisplayName)}},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, errors.New("customer external identity: empty gateway response")
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return nil, errors.New("customer external identity: response payload is missing")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var result struct {
		Item ExternalIdentityResolution `json:"item"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.Item.CustomerUUID) == "" || strings.TrimSpace(result.Item.MembershipUUID) == "" {
		return nil, errors.New("customer external identity: incomplete response")
	}
	return &result.Item, nil
}

func (c *ExternalIdentityResolver) ResolveExternalIdentity(ctx context.Context, req ResolveExternalIdentityRequest) (*ExternalIdentityResolution, error) {
	return c.Resolve(ctx, req)
}
