package customerfw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

const (
	CapabilityCustomerExternalIdentitiesRead   = "com.corex.customer.external_identities.service_read"
	CapabilityCustomerExternalIdentitiesManage = "com.corex.customer.external_identities.service_manage"
)

type ExternalIdentityItem struct {
	IdentityUUID       string `json:"identity_uuid"`
	CustomerUUID       string `json:"customer_uuid"`
	ProviderSubject    string `json:"provider_subject"`
	Status             string `json:"status"`
	Type               string `json:"type"`
	PrimaryContactUUID string `json:"primary_contact_uuid,omitempty"`
}
type ExternalIdentityLookup struct {
	Found bool                  `json:"found"`
	Item  *ExternalIdentityItem `json:"item,omitempty"`
}
type ExternalIdentityPage struct {
	Items    []ExternalIdentityItem `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}
type ExternalIdentityCustomer struct {
	Type           string               `json:"type,omitempty"`
	DisplayName    string               `json:"display_name,omitempty"`
	Nickname       string               `json:"nickname,omitempty"`
	GivenName      string               `json:"given_name,omitempty"`
	FamilyName     string               `json:"family_name,omitempty"`
	PrimaryEmail   string               `json:"primary_email,omitempty"`
	PrimaryPhone   string               `json:"primary_phone,omitempty"`
	PrimaryContact *PrimaryContactInput `json:"primary_contact,omitempty"`
}

// ExternalIdentityStore can be implemented by a transactional plugin-local store;
// the delegated client below always uses Core and never writes plugin-local data.
type ExternalIdentityStore interface {
	Lookup(context.Context, string) (*ExternalIdentityLookup, error)
	ListByCustomer(context.Context, string, int, int) (*ExternalIdentityPage, error)
	Bind(context.Context, string, string) (*ExternalIdentityItem, error)
	CreateAndBind(context.Context, string, ExternalIdentityCustomer) (*ExternalIdentityItem, error)
}
type ExternalIdentityClient struct{ invoker AdminGatewayInvoker }

func NewExternalIdentityClient(invoker AdminGatewayInvoker) (*ExternalIdentityClient, error) {
	if invoker == nil {
		return nil, errors.New("CUSTOMER_IDENTITY_GATEWAY_REQUIRED")
	}
	return &ExternalIdentityClient{invoker: invoker}, nil
}
func (c *ExternalIdentityClient) invoke(ctx context.Context, capability string, body any, out any) error {
	if c == nil || c.invoker == nil {
		return errors.New("CUSTOMER_IDENTITY_GATEWAY_REQUIRED")
	}
	response, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: capability, PreferredProtocol: "core_internal", Payload: map[string]any{"method": "INVOKE", "endpoint": "core://customer/external-identities", "body": body}})
	if err != nil {
		return NormalizeIdentityError(err)
	}
	if response == nil || response.Data == nil {
		return errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
	}
	payload, ok := response.Data["payload"]
	if !ok {
		return errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func validIdentityItem(item *ExternalIdentityItem) bool {
	if item == nil {
		return false
	}
	for _, id := range []string{item.IdentityUUID, item.CustomerUUID} {
		parsed, e := uuid.Parse(id)
		if e != nil || parsed == uuid.Nil {
			return false
		}
	}
	return item.ProviderSubject != ""
}
func (c *ExternalIdentityClient) Lookup(ctx context.Context, subject string) (*ExternalIdentityLookup, error) {
	if !ValidExternalIdentitySubject(subject) {
		return nil, IdentityContractError("CUSTOMER_ACCOUNT_INVALID_ARGUMENT", 400)
	}
	var wire struct {
		Found *bool                 `json:"found"`
		Item  *ExternalIdentityItem `json:"item"`
	}
	if err := c.invoke(ctx, CapabilityCustomerExternalIdentitiesRead, map[string]any{"operation": "lookup", "provider_subject": subject}, &wire); err != nil {
		return nil, err
	}
	if wire.Found == nil || (*wire.Found && !validIdentityItem(wire.Item)) || (!*wire.Found && wire.Item != nil) {
		return nil, errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
	}
	return &ExternalIdentityLookup{Found: *wire.Found, Item: wire.Item}, nil
}
func (c *ExternalIdentityClient) ListByCustomer(ctx context.Context, customer string, page, size int) (*ExternalIdentityPage, error) {
	var out ExternalIdentityPage
	err := c.invoke(ctx, CapabilityCustomerExternalIdentitiesRead, map[string]any{"operation": "list_by_customer", "customer_uuid": customer, "page": page, "page_size": size}, &out)
	if err != nil {
		return nil, err
	}
	for i := range out.Items {
		if !validIdentityItem(&out.Items[i]) || out.Items[i].CustomerUUID != customer {
			return nil, errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
		}
	}
	return &out, nil
}
func (c *ExternalIdentityClient) write(ctx context.Context, body any) (*ExternalIdentityItem, error) {
	var out struct {
		Item *ExternalIdentityItem `json:"item"`
	}
	if err := c.invoke(ctx, CapabilityCustomerExternalIdentitiesManage, body, &out); err != nil {
		return nil, err
	}
	if !validIdentityItem(out.Item) {
		return nil, errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
	}
	return out.Item, nil
}
func (c *ExternalIdentityClient) Bind(ctx context.Context, customer, subject string) (*ExternalIdentityItem, error) {
	if !ValidExternalIdentitySubject(subject) {
		return nil, IdentityContractError("CUSTOMER_ACCOUNT_INVALID_ARGUMENT", 400)
	}
	item, err := c.write(ctx, map[string]any{"operation": "bind", "customer_uuid": customer, "provider_subject": subject})
	if err != nil {
		return nil, err
	}
	if item.CustomerUUID != customer || item.ProviderSubject != subject {
		return nil, errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
	}
	return item, nil
}
func (c *ExternalIdentityClient) CreateAndBind(ctx context.Context, subject string, profile ExternalIdentityCustomer) (*ExternalIdentityItem, error) {
	if err := ValidateIdentityCustomer(profile); err != nil {
		return nil, err
	}
	if !ValidExternalIdentitySubject(subject) {
		return nil, IdentityContractError("CUSTOMER_ACCOUNT_INVALID_ARGUMENT", 400)
	}
	if strings.TrimSpace(profile.Type) == "" {
		profile.Type = "person"
	}
	item, err := c.write(ctx, map[string]any{"operation": "create_and_bind", "provider_subject": subject, "customer": profile})
	if err != nil {
		return nil, err
	}
	if item.ProviderSubject != subject || item.PrimaryContactUUID == "" || (item.Type != "person" && item.Type != "company") {
		return nil, errors.New("CUSTOMER_IDENTITY_RESPONSE_INVALID")
	}
	return item, nil
}
