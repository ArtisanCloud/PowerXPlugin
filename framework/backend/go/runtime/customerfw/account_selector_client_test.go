package customerfw

import (
	"context"
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
		return &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"uuid": "11111111-1111-4111-8111-111111111111", "display_name": "Customer", "status": "active"}}}}, nil
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
	item, err := client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{DisplayName: "Customer", Status: "active"})
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
