package metadata

import (
	"context"
	"net/http"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

type stubInvoker struct {
	requests []gateway.InvokeRequest
	queue    []*gateway.Response
	err      error
}

func (s *stubInvoker) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.requests = append(s.requests, req)
	if s.err != nil {
		return nil, s.err
	}
	if len(s.queue) == 0 {
		return &gateway.Response{TraceID: "trace", Data: map[string]any{"payload": map[string]any{"items": []any{}, "pagination": map[string]any{"total": 0, "page": 1, "page_size": 100}}}}, nil
	}
	resp := s.queue[0]
	s.queue = s.queue[1:]
	return resp, nil
}

func TestListDictionaryItemsBuildsGatewayInvocation(t *testing.T) {
	invoker := &stubInvoker{queue: []*gateway.Response{{
		TraceID: "trace-1",
		Data: map[string]any{"payload": map[string]any{
			"items": []any{
				map[string]any{"uuid": "item-uuid", "namespace_uuid": "ns-uuid", "code": "vip", "label_i18n": map[string]any{"en": "VIP"}, "status": "enabled"},
			},
			"pagination": map[string]any{"total": 1, "page": 1, "page_size": 50},
		}},
	}}}
	client, err := NewClient(Config{Invoker: invoker, TenantUUID: "tenant-a", Locale: "en", PageSize: 50})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	page, err := client.ListDictionaryItems(context.Background(), ListDictionaryItemsRequest{NamespaceUUID: "ns-uuid"})
	if err != nil {
		t.Fatalf("ListDictionaryItems() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Code != "vip" {
		t.Fatalf("items = %+v", page.Items)
	}
	if len(invoker.requests) != 1 {
		t.Fatalf("requests len = %d", len(invoker.requests))
	}
	req := invoker.requests[0]
	if req.CapabilityID != CapabilityDictionaryRead {
		t.Fatalf("capability = %q", req.CapabilityID)
	}
	if req.PreferredProtocol != preferredProtocolREST {
		t.Fatalf("preferred protocol = %q", req.PreferredProtocol)
	}
	payload, ok := req.Payload.(restPayload)
	if !ok {
		t.Fatalf("payload type = %T", req.Payload)
	}
	if payload.Method != http.MethodGet || payload.Endpoint != "/api/v1/tenant/metadata/dictionaries/ns-uuid/items" {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Query["locale"] != "en" || payload.Query["page_size"] != 50 {
		t.Fatalf("query = %+v", payload.Query)
	}
}

func TestResolveDictionaryItemRequiresExactCode(t *testing.T) {
	invoker := &stubInvoker{queue: []*gateway.Response{
		{Data: map[string]any{"payload": map[string]any{
			"items":      []any{map[string]any{"uuid": "ns-uuid", "namespace": "corex.customer.level", "module": "customer", "name_i18n": map[string]any{"en": "Level"}, "status": "enabled"}},
			"pagination": map[string]any{"total": 1, "page": 1, "page_size": 100},
		}}},
		{Data: map[string]any{"payload": map[string]any{
			"items":      []any{map[string]any{"uuid": "item-uuid", "namespace_uuid": "ns-uuid", "code": "vip", "label_i18n": map[string]any{"en": "VIP"}, "status": "enabled"}},
			"pagination": map[string]any{"total": 1, "page": 1, "page_size": 100},
		}}},
	}}
	client, err := NewClient(Config{Invoker: invoker})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	item, err := client.ResolveDictionaryItem(context.Background(), "corex.customer.level", "vip")
	if err != nil {
		t.Fatalf("ResolveDictionaryItem() error = %v", err)
	}
	if item.UUID != "item-uuid" {
		t.Fatalf("item = %+v", item)
	}
}

func TestLegacyTagBindingReplacementFailsExplicitly(t *testing.T) {
	client, err := NewClient(Config{Invoker: &stubInvoker{}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ReplaceTagBindings(context.Background(), ReplaceTagBindingsRequest{
		ResourceType: "corex.customer",
		ResourceUUID: "customer-uuid",
		TagUUIDs:     []string{"tag-uuid"},
	})
	if CodeOf(err) != CodeInvalidArgument {
		t.Fatalf("CodeOf(err) = %s err=%v", CodeOf(err), err)
	}
}

func TestMissingPayloadFails(t *testing.T) {
	invoker := &stubInvoker{queue: []*gateway.Response{{Data: map[string]any{}}}}
	client, err := NewClient(Config{Invoker: invoker})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListResourceTypes(context.Background(), ListResourceTypesRequest{})
	if CodeOf(err) != CodeDecodeFailed {
		t.Fatalf("CodeOf(err) = %s err=%v", CodeOf(err), err)
	}
}
