package contactfw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const contactCoreEndpoint = "core://customer/contacts"

// CapabilityGatewayInvoker is deliberately narrower than a transport client
// exposed to business code. CapabilityClient fixes every Contact invocation
// field below; consumers receive only DelegatedContactClient's typed methods.
type CapabilityGatewayInvoker interface {
	Invoke(context.Context, gateway.InvokeRequest) (*gateway.Response, error)
}

type CapabilityClientConfig struct{ Invoker CapabilityGatewayInvoker }

// CapabilityClient is the delegated Contact binding for the currently
// published Core capability transport. It never accepts an endpoint, HTTP
// method, header, tenant UUID, or arbitrary body from its caller.
type CapabilityClient struct{ invoker CapabilityGatewayInvoker }

func NewCapabilityClient(cfg CapabilityClientConfig) (*CapabilityClient, error) {
	if cfg.Invoker == nil {
		return nil, errors.New("contact capability invoker is required")
	}
	return &CapabilityClient{invoker: cfg.Invoker}, nil
}

func (c *CapabilityClient) Create(ctx context.Context, in CreateContactInput) (*Contact, error) {
	if err := ValidateCreateInput(in); err != nil {
		return nil, err
	}
	var out contactItemWire
	err := c.invoke(ctx, CapabilityContactsServiceManage, "create", contactOperation{CustomerUUID: in.CustomerUUID, DisplayName: &in.DisplayName, GivenName: &in.GivenName, FamilyName: &in.FamilyName, Status: statusPtr(in.Status), Roles: &in.Roles, Tags: &in.Tags, CreationIntent: string(in.CreationIntent)}, &out)
	return out.Item.contact(), err
}
func (c *CapabilityClient) Get(ctx context.Context, in GetContactInput) (*Contact, error) {
	if err := ValidateGetInput(in); err != nil {
		return nil, err
	}
	var out contactItemWire
	err := c.invoke(ctx, CapabilityContactsServiceRead, "get", contactOperation{CustomerUUID: in.CustomerUUID, ContactUUID: in.ContactUUID}, &out)
	return out.Item.contact(), err
}
func (c *CapabilityClient) Update(ctx context.Context, in UpdateContactInput) (*Contact, error) {
	if err := ValidateUpdateInput(in); err != nil {
		return nil, err
	}
	var out contactItemWire
	err := c.invoke(ctx, CapabilityContactsServiceManage, "update", contactOperation{CustomerUUID: in.CustomerUUID, ContactUUID: in.ContactUUID, DisplayName: in.DisplayName, GivenName: in.GivenName, FamilyName: in.FamilyName, Status: in.Status, Roles: in.Roles, Tags: in.Tags}, &out)
	return out.Item.contact(), err
}
func (c *CapabilityClient) ListByCustomer(ctx context.Context, in ListByCustomerInput) (ContactPage, error) {
	var err error
	in, err = NormalizeListInput(in)
	if err != nil {
		return ContactPage{}, err
	}
	var out contactPageWire
	err = c.invoke(ctx, CapabilityContactsServiceRead, "list_by_customer", contactOperation{CustomerUUID: in.CustomerUUID, Query: in.Query, Status: in.Status, Page: in.Page, PageSize: in.PageSize}, &out)
	if err != nil {
		return ContactPage{}, err
	}
	items := make([]Contact, 0, len(out.Items))
	for _, item := range out.Items {
		items = append(items, item.toContact())
	}
	return ContactPage{Items: items, Page: out.Page, PageSize: out.PageSize, Total: out.Total}, nil
}
func (c *CapabilityClient) ResolveIdentity(ctx context.Context, in ResolveContactIdentityInput) (*ContactIdentityResolution, error) {
	if err := ValidateResolveIdentityInput(in); err != nil {
		return nil, err
	}
	var out resolutionItemWire
	err := c.invoke(ctx, CapabilityContactsServiceRead, "resolve_identity", contactOperation{CustomerUUID: in.CustomerUUID, ChannelDictionaryItemUUID: in.ChannelDictionaryItemUUID, ExternalSubject: in.ExternalSubject}, &out)
	if err != nil {
		return nil, err
	}
	return &ContactIdentityResolution{Contact: out.Item.Contact.toContact(), Identity: out.Item.Identity.toIdentity()}, nil
}
func (c *CapabilityClient) BindIdentity(ctx context.Context, in BindContactIdentityInput) (*ContactIdentity, error) {
	if err := ValidateBindIdentityInput(in); err != nil {
		return nil, err
	}
	var out identityItemWire
	err := c.invoke(ctx, CapabilityContactsServiceManage, "bind_identity", contactOperation{CustomerUUID: in.CustomerUUID, ContactUUID: in.ContactUUID, ChannelDictionaryItemUUID: in.ChannelDictionaryItemUUID, ExternalSubject: in.ExternalSubject}, &out)
	if err != nil {
		return nil, err
	}
	item := out.Item.toIdentity()
	return &item, nil
}

func (c *CapabilityClient) invoke(ctx context.Context, capabilityID, operation string, body contactOperation, out any) error {
	if c == nil || c.invoker == nil {
		return NewError(CodeDelegateUnavailable, errors.New("contact capability client unavailable"))
	}
	body.Operation = operation
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: capabilityID, PreferredProtocol: "core_internal", Payload: map[string]any{"method": "INVOKE", "endpoint": contactCoreEndpoint, "body": body}})
	if err != nil {
		return mapCapabilityError(err)
	}
	if resp == nil || resp.Data == nil {
		return NewError(CodeDelegateUnavailable, errors.New("contact capability response is empty"))
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return NewError(CodeDelegateUnavailable, errors.New("contact capability payload is missing"))
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return NewError(CodeDelegateUnavailable, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return NewError(CodeDelegateUnavailable, err)
	}
	return nil
}

func mapCapabilityError(err error) error {
	var invokeErr *gateway.InvocationError
	if errors.As(err, &invokeErr) {
		for _, item := range invokeErr.Errors {
			if code := ErrorCode(strings.TrimSpace(item.Code)); knownContactCode(code) {
				return NewError(code, err)
			}
		}
	}
	return NewError(CodeDelegateUnavailable, err)
}
func knownContactCode(code ErrorCode) bool {
	switch code {
	case CodeInvalidArgument, CodeNotFound, CodeCustomerMismatch, CodeCustomerMembershipInactive, CodeIdentityNotFound, CodeIdentityConflict, CodeChannelDictionaryInvalid, CodeIdentityChannelMigrationRequired, CodeCapabilityForbidden:
		return true
	}
	return false
}
func statusPtr(value Status) *Status {
	if value == "" {
		return nil
	}
	return &value
}

type contactOperation struct {
	Operation                 string    `json:"operation"`
	CustomerUUID              string    `json:"customer_uuid"`
	ContactUUID               string    `json:"contact_uuid,omitempty"`
	Query                     string    `json:"q,omitempty"`
	Status                    *Status   `json:"status,omitempty"`
	Page                      int       `json:"page,omitempty"`
	PageSize                  int       `json:"page_size,omitempty"`
	ChannelDictionaryItemUUID string    `json:"channel_dictionary_item_uuid,omitempty"`
	ExternalSubject           string    `json:"external_subject,omitempty"`
	DisplayName               *string   `json:"display_name,omitempty"`
	GivenName                 *string   `json:"given_name,omitempty"`
	FamilyName                *string   `json:"family_name,omitempty"`
	Roles                     *[]Role   `json:"roles,omitempty"`
	Tags                      *[]string `json:"tags,omitempty"`
	CreationIntent            string    `json:"creation_intent,omitempty"`
}
type contactWire struct {
	UUID         string         `json:"uuid"`
	TenantUUID   string         `json:"tenant_uuid"`
	CustomerUUID string         `json:"customer_uuid"`
	DisplayName  string         `json:"display_name"`
	GivenName    string         `json:"given_name"`
	FamilyName   string         `json:"family_name"`
	Status       Status         `json:"status"`
	Roles        []Role         `json:"roles"`
	Tags         []string       `json:"tags"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (w contactWire) toContact() Contact {
	return Contact{ContactUUID: w.UUID, TenantUUID: w.TenantUUID, CustomerUUID: w.CustomerUUID, DisplayName: w.DisplayName, GivenName: w.GivenName, FamilyName: w.FamilyName, Status: w.Status, Roles: w.Roles, Tags: w.Tags, Metadata: w.Metadata, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt}
}
func (w contactWire) contact() *Contact { item := w.toContact(); return &item }

type contactItemWire struct {
	Item contactWire `json:"item"`
}

type identityWire struct {
	UUID                      string         `json:"uuid"`
	TenantUUID                string         `json:"tenant_uuid"`
	CustomerUUID              string         `json:"customer_uuid"`
	ContactUUID               string         `json:"contact_uuid"`
	ChannelDictionaryItemUUID string         `json:"channel_dictionary_item_uuid"`
	ExternalSubject           string         `json:"external_subject"`
	Status                    IdentityStatus `json:"status"`
	VerifiedAt                *time.Time     `json:"verified_at"`
	Metadata                  map[string]any `json:"metadata"`
}

func (w identityWire) toIdentity() ContactIdentity {
	return ContactIdentity{IdentityUUID: w.UUID, TenantUUID: w.TenantUUID, CustomerUUID: w.CustomerUUID, ContactUUID: w.ContactUUID, ChannelDictionaryItemUUID: w.ChannelDictionaryItemUUID, ExternalSubject: w.ExternalSubject, Status: w.Status, VerifiedAt: w.VerifiedAt, Metadata: w.Metadata}
}

type resolutionWire struct {
	Contact  contactWire  `json:"contact"`
	Identity identityWire `json:"identity"`
}
type resolutionItemWire struct {
	Item resolutionWire `json:"item"`
}
type identityItemWire struct {
	Item identityWire `json:"item"`
}
type contactPageWire struct {
	Items    []contactWire `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

var _ DelegatedContactClient = (*CapabilityClient)(nil)
