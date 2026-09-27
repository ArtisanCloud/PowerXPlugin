package customerfw

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

type accountSelectorInvoker struct {
	request gateway.InvokeRequest
	items   []any
}

func (s *accountSelectorInvoker) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.request = req
	if req.CapabilityID == CapabilityCustomerAccountsServiceManage {
		return &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"uuid": "11111111-1111-4111-8111-111111111111", "type": "person", "primary_contact_uuid": "22222222-2222-4222-8222-222222222222", "display_name": "Customer", "status": "active"}}}}, nil
	}
	items := s.items
	if items == nil {
		items = []any{map[string]any{"uuid": "11111111-1111-4111-8111-111111111111", "display_name": "Customer"}}
	}
	return &gateway.Response{Data: map[string]any{"payload": map[string]any{
		"items": items,
		"page":  2, "page_size": 10, "total": 1,
	}}}, nil
}

func TestAccountCreateUsesFixedManageCapabilityAndRejectsCallerTenant(t *testing.T) {
	stub := &accountSelectorInvoker{}
	client, err := NewAccountSelectorClient(stub)
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{Type: "person", DisplayName: "Customer", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if item.CustomerUUID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("unexpected account: %+v", item)
	}
	if stub.request.CapabilityID != CapabilityCustomerAccountsServiceManage || stub.request.PreferredProtocol != "core_internal" || stub.request.TenantUUID != "" {
		t.Fatalf("unexpected invocation: %+v", stub.request)
	}
	payload, ok := stub.request.Payload.(map[string]any)
	if !ok || payload["method"] != "INVOKE" || payload["endpoint"] != customerAccountsCoreEndpoint {
		t.Fatalf("unexpected payload: %#v", stub.request.Payload)
	}
	body, ok := payload["body"].(map[string]any)
	if !ok || body["operation"] != "create" || body["display_name"] != "Customer" {
		t.Fatalf("unexpected body: %#v", payload["body"])
	}
	if _, exists := body["tenant_uuid"]; exists {
		t.Fatal("caller tenant was forwarded")
	}
}

func TestAccountSelectorRejectsMissingCoreUUID(t *testing.T) {
	client, err := NewAccountSelectorClient(&accountSelectorInvoker{items: []any{map[string]any{"status": "active"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListAccounts(context.Background(), ListAccountsRequest{}); err == nil {
		t.Fatal("missing Core account uuid was accepted")
	}
}

func TestAccountSelectorUsesFixedServiceContract(t *testing.T) {
	stub := &accountSelectorInvoker{}
	client, err := NewAccountSelectorClient(stub)
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.ListAccounts(context.Background(), ListAccountsRequest{
		TenantUUID: "caller-controlled-tenant", Query: "Customer", Status: "active", Page: 2, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].CustomerUUID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("unexpected account page: %+v", page)
	}
	if stub.request.CapabilityID != CapabilityCustomerAccountsServiceRead || stub.request.PreferredProtocol != "core_internal" || stub.request.TenantUUID != "" {
		t.Fatalf("unexpected selector invocation: %+v", stub.request)
	}
	payload, ok := stub.request.Payload.(map[string]any)
	if !ok || len(payload) != 3 || payload["method"] != "INVOKE" || payload["endpoint"] != customerAccountsCoreEndpoint {
		t.Fatalf("unexpected selector envelope: %#v", stub.request.Payload)
	}
	body, ok := payload["body"].(map[string]any)
	if !ok || len(body) != 5 || body["operation"] != "list" || body["page"] != 2 || body["page_size"] != 10 || body["q"] != "Customer" || body["status"] != "active" {
		t.Fatalf("unexpected selector body: %#v", payload["body"])
	}
}

func TestAccountCreateDefaultsOmittedTypeToPerson(t *testing.T) {
	stub := &accountSelectorInvoker{}
	client, err := NewAccountSelectorClient(stub)
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{DisplayName: "Customer"})
	if err != nil {
		t.Fatal(err)
	}
	body := stub.request.Payload.(map[string]any)["body"].(map[string]any)
	if body["type"] != "person" || item.Type != "person" || item.PrimaryContactUUID == "" {
		t.Fatalf("default contract mismatch: %#v %#v", body, item)
	}
	if _, err = client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{Type: "invalid"}); err == nil {
		t.Fatal("invalid type accepted")
	}
	if _, err = client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{Type: "company"}); err == nil {
		t.Fatal("company accepted without natural person")
	}
}

func TestAccountUpdatePreservesPatchAndUsesServiceContract(t *testing.T) {
	stub := &accountSelectorInvoker{}
	client, _ := NewAccountSelectorClient(stub)
	empty := ""
	item, err := client.UpdateBasicAccount(context.Background(), UpdateBasicAccountRequest{CustomerUUID: "11111111-1111-4111-8111-111111111111", PrimaryPhone: &empty})
	if err != nil {
		t.Fatal(err)
	}
	payload := stub.request.Payload.(map[string]any)
	body := payload["body"].(map[string]any)
	if stub.request.CapabilityID != CapabilityCustomerAccountsServiceManage || stub.request.PreferredProtocol != "core_internal" || stub.request.TenantUUID != "" || payload["method"] != "INVOKE" || payload["endpoint"] != customerAccountsCoreEndpoint {
		t.Fatalf("unexpected request: %+v", stub.request)
	}
	if len(body) != 3 || body["operation"] != "update" || body["primary_phone"] != "" || body["customer_uuid"] != item.CustomerUUID {
		t.Fatalf("unexpected patch: %+v", body)
	}
	if item.PrimaryPhone != "" || item.PrimaryEmail != "" {
		t.Fatalf("omitted response fields must decode empty: %+v", item)
	}
}

func TestAccountUpdateRejectsUnsafePayloads(t *testing.T) {
	for _, body := range []string{`{"type":"person"}`, `{"tenant_uuid":"override"}`, `{"primary_contact_uuid":"override"}`, `{"Display_Name":"alias"}`, `{"primary_phone":null}`, `{"status":123}`} {
		var req UpdateBasicAccountRequest
		if err := json.Unmarshal([]byte(body), &req); err == nil {
			t.Fatalf("accepted: %s", body)
		}
	}
	client, _ := NewAccountSelectorClient(&accountSelectorInvoker{})
	for _, req := range []UpdateBasicAccountRequest{{CustomerUUID: "invalid"}, {CustomerUUID: "00000000-0000-0000-0000-000000000000"}, {CustomerUUID: "11111111-1111-4111-8111-111111111111"}} {
		if _, err := client.UpdateBasicAccount(context.Background(), req); err == nil {
			t.Fatal("accepted invalid patch")
		}
	}
}
