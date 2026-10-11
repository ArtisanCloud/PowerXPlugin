package contactfw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

const (
	testCustomerUUID = "11111111-1111-4111-8111-111111111111"
	testContactUUID  = "22222222-2222-4222-8222-222222222222"
)

type storeStub struct{ label string }

func (s storeStub) Create(context.Context, CreateContactInput) (*Contact, error) {
	return &Contact{DisplayName: s.label}, nil
}
func (s storeStub) Get(context.Context, GetContactInput) (*Contact, error) {
	return &Contact{DisplayName: s.label}, nil
}
func (s storeStub) Update(context.Context, UpdateContactInput) (*Contact, error) {
	return &Contact{DisplayName: s.label}, nil
}
func (s storeStub) ListByCustomer(context.Context, ListByCustomerInput) (ContactPage, error) {
	return ContactPage{}, nil
}
func (s storeStub) ResolveIdentity(context.Context, ResolveContactIdentityInput) (*ContactIdentityResolution, error) {
	return nil, nil
}
func (s storeStub) BindIdentity(context.Context, BindContactIdentityInput) (*ContactIdentity, error) {
	return nil, nil
}

type failingBinding struct{ storeStub }

func (failingBinding) Get(context.Context, GetContactInput) (*Contact, error) {
	return nil, errors.New("core binding is unavailable")
}

type semanticFailureBinding struct{ storeStub }

func (semanticFailureBinding) Get(context.Context, GetContactInput) (*Contact, error) {
	return nil, NewError(CodeCustomerMismatch, nil)
}

type contactGatewayStub struct {
	request  gateway.InvokeRequest
	response *gateway.Response
	err      error
}

func (s *contactGatewayStub) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.request = req
	return s.response, s.err
}

func TestRuntimeSelectsOnlyStartupMode(t *testing.T) {
	local, delegated := storeStub{label: "local"}, storeStub{label: "delegated"}
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		r, err := NewRuntime(tc.mode, local, delegated)
		if err != nil {
			t.Fatal(err)
		}
		s, err := r.Store()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Get(context.Background(), GetContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID})
		if err != nil || got.DisplayName != tc.want {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	}
}

func TestRequiredDelegatedRuntimeFailsClosed(t *testing.T) {
	r, err := NewRuntime(provider.ModeDelegated, storeStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = r.ValidateRequired()
	if CodeOf(err) != CodeDelegateUnavailable {
		t.Fatalf("code=%s err=%v", CodeOf(err), err)
	}
}

func TestRuntimeDoesNotFallbackToLocal(t *testing.T) {
	r, err := NewRuntime(provider.ModeDelegated, storeStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Store(); err == nil {
		t.Fatal("delegated runtime fell back to local store")
	}
}

func TestValidateTags(t *testing.T) {
	if err := ValidateTags([]string{"vip", "region.cn"}); err != nil {
		t.Fatal(err)
	}
	if CodeOf(ValidateTags([]string{"not allowed"})) != CodeInvalidArgument {
		t.Fatal("invalid tag accepted")
	}
	if CodeOf(ValidateTags([]string{"Region"})) != CodeInvalidArgument {
		t.Fatal("uppercase tag accepted")
	}
}

func TestValidateCreateInputRequiresExplicitTemporaryIntentAndAudit(t *testing.T) {
	valid := CreateContactInput{CustomerUUID: testCustomerUUID, DisplayName: "Contact", Status: StatusTemporary, CreationIntent: CreationIntentExplicitTemporary}
	if err := ValidateCreateInput(valid); err != nil {
		t.Fatal(err)
	}
	valid.CreationIntent = CreationIntentExplicitCreate
	if CodeOf(ValidateCreateInput(valid)) != CodeInvalidArgument {
		t.Fatal("implicit temporary creation accepted")
	}
	valid.CreationIntent = CreationIntentExplicitTemporary
	valid.Roles = []Role{"designer"}
	if CodeOf(ValidateCreateInput(valid)) != CodeInvalidArgument {
		t.Fatal("business role accepted in contact roles")
	}
}

func TestNormalizeListInputUsesCorePagingContract(t *testing.T) {
	got, err := NormalizeListInput(ListByCustomerInput{CustomerUUID: testCustomerUUID})
	if err != nil || got.Page != 1 || got.PageSize != DefaultPageSize {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	_, err = NormalizeListInput(ListByCustomerInput{CustomerUUID: testCustomerUUID, PageSize: MaximumPageSize + 1})
	if CodeOf(err) != CodeInvalidArgument {
		t.Fatalf("code=%s err=%v", CodeOf(err), err)
	}
}

func TestCoreDelegatedClientMapsOnlyTransportFailure(t *testing.T) {
	client, err := NewCoreDelegatedClient(failingBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Get(context.Background(), GetContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID}); CodeOf(err) != CodeDelegateUnavailable {
		t.Fatalf("code=%s err=%v", CodeOf(err), err)
	}
	semantic, err := NewCoreDelegatedClient(semanticFailureBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = semantic.Get(context.Background(), GetContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID}); CodeOf(err) != CodeCustomerMismatch {
		t.Fatalf("semantic core error changed: code=%s err=%v", CodeOf(err), err)
	}
}

func TestCapabilityClientUsesFixedCoreInternalContactContract(t *testing.T) {
	stub := &contactGatewayStub{response: &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"uuid": testContactUUID, "tenant_uuid": "tenant", "customer_uuid": testCustomerUUID, "display_name": "Ada", "status": "active", "roles": []string{"primary"}, "tags": []string{"vip"}}}}}}
	client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.Get(context.Background(), GetContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID})
	if err != nil || item.ContactUUID != testContactUUID {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	if stub.request.CapabilityID != CapabilityContactsServiceRead || stub.request.PreferredProtocol != "core_internal" || stub.request.TenantUUID != "" {
		t.Fatalf("request=%+v", stub.request)
	}
	payload := stub.request.Payload.(map[string]any)
	if payload["method"] != "INVOKE" || payload["endpoint"] != contactCoreEndpoint {
		t.Fatalf("payload=%+v", payload)
	}
	body := payload["body"].(contactOperation)
	if body.Operation != "get" || body.CustomerUUID != testCustomerUUID || body.ContactUUID != testContactUUID {
		t.Fatalf("body=%+v", body)
	}
}

func TestCapabilityClientMapsCoreSemanticError(t *testing.T) {
	stub := &contactGatewayStub{err: &gateway.InvocationError{Errors: []gateway.GatewayError{{Code: string(CodeCustomerMismatch)}}}}
	client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Get(context.Background(), GetContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID})
	if CodeOf(err) != CodeCustomerMismatch {
		t.Fatalf("code=%s err=%v", CodeOf(err), err)
	}
}

func TestCapabilityClientValidatesBeforeDelegatedInvocation(t *testing.T) {
	stub := &contactGatewayStub{}
	client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ResolveIdentity(context.Background(), ResolveContactIdentityInput{
		CustomerUUID:              testCustomerUUID,
		ChannelDictionaryItemUUID: "not-a-uuid",
	})
	if CodeOf(err) != CodeInvalidArgument {
		t.Fatalf("code=%s err=%v", CodeOf(err), err)
	}
	if stub.request.CapabilityID != "" {
		t.Fatalf("invalid input invoked Core: %+v", stub.request)
	}
}

func TestCapabilityClientUsesOnlyChannelDictionaryItemUUID(t *testing.T) {
	channelDictionaryItemUUID := "33333333-3333-4333-8333-333333333333"
	stub := &contactGatewayStub{response: &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{
		"uuid": testContactUUID, "tenant_uuid": "tenant", "customer_uuid": testCustomerUUID, "contact_uuid": testContactUUID,
		"channel_dictionary_item_uuid": channelDictionaryItemUUID, "external_subject": "ada@example.test", "status": "active",
	}}}}}
	client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.BindIdentity(context.Background(), BindContactIdentityInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID, ChannelDictionaryItemUUID: channelDictionaryItemUUID, ExternalSubject: "ada@example.test"})
	if err != nil || item.ChannelDictionaryItemUUID != channelDictionaryItemUUID {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	body := stub.request.Payload.(map[string]any)["body"].(contactOperation)
	if body.ChannelDictionaryItemUUID != channelDictionaryItemUUID {
		t.Fatalf("body=%+v", body)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"channel":`) {
		t.Fatalf("legacy channel field sent: %s", raw)
	}
}

func TestCapabilityClientUsesPublishedCreateAndListWires(t *testing.T) {
	stub := &contactGatewayStub{response: &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"uuid": testContactUUID, "tenant_uuid": "tenant", "customer_uuid": testCustomerUUID, "display_name": "Ada", "status": "active"}}}}}
	client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Create(context.Background(), CreateContactInput{
		CustomerUUID:   testCustomerUUID,
		DisplayName:    "Ada",
		Status:         StatusActive,
		Roles:          []Role{RolePrimary},
		Tags:           []string{"vip"},
		CreationIntent: CreationIntentExplicitCreate,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := stub.request.Payload.(map[string]any)
	body := payload["body"].(contactOperation)
	if stub.request.CapabilityID != CapabilityContactsServiceManage || body.Operation != "create" || body.CreationIntent != string(CreationIntentExplicitCreate) || body.DisplayName == nil || *body.DisplayName != "Ada" {
		t.Fatalf("create request=%+v body=%+v", stub.request, body)
	}

	stub.response = &gateway.Response{Data: map[string]any{"payload": map[string]any{"items": []any{}, "total": 0, "page": 1, "page_size": DefaultPageSize}}}
	_, err = client.ListByCustomer(context.Background(), ListByCustomerInput{CustomerUUID: testCustomerUUID})
	if err != nil {
		t.Fatal(err)
	}
	payload = stub.request.Payload.(map[string]any)
	body = payload["body"].(contactOperation)
	if stub.request.CapabilityID != CapabilityContactsServiceRead || body.Operation != "list_by_customer" || body.Page != 1 || body.PageSize != DefaultPageSize {
		t.Fatalf("list request=%+v body=%+v", stub.request, body)
	}
}
