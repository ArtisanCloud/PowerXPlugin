package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostClientListsTagBindingsUsingTenantContractAndServiceSTS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tenant/metadata/tag-bindings" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer service-sts" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"payload":{"items":[{"uuid":"00000000-0000-4000-8000-000000000001","tag_uuid":"00000000-0000-4000-8000-000000000002","resource_type":"corex.customer","resource_uuid":"00000000-0000-4000-8000-000000000003"}]}}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.ListTagBindings(context.Background(), ListTagBindingsRequest{ResourceType: "corex.customer", ResourceUUID: "00000000-0000-4000-8000-000000000003"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UUID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("items = %#v", items)
	}
}

func TestHostClientPreservesMetadataReasonCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"reason_code":"METADATA_FORBIDDEN"}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListDictionaryNamespaces(context.Background(), ListDictionaryNamespacesRequest{})
	var metadataErr *Error
	if !errors.As(err, &metadataErr) || metadataErr.Code != CodeForbidden {
		t.Fatalf("error = %#v", err)
	}
}

func TestHostClientAPIKeyUsesServiceCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "ApiKey development-key" {
			t.Fatalf("authorization = %q", got)
		}
		if r.URL.Path != "/api/v1/tenant/metadata/tags" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"payload":{"items":[],"pagination":{"page":1,"page_size":20,"total":0}}}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "development-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListTags(context.Background(), ListTagsRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestHostClientRejectsMissingTagBindingItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"payload":{}}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTagBindings(context.Background(), ListTagBindingsRequest{ResourceType: "corex.customer", ResourceUUID: "00000000-0000-4000-8000-000000000003"})
	if CodeOf(err) != CodeDecodeFailed {
		t.Fatalf("error = %v", err)
	}
}

func TestHostDictionaryVersionAndExtensionContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/tenant/metadata/dictionary-items/00000000-0000-4000-8000-000000000001" {
			t.Errorf("METADATA_ROUTE_MISMATCH")
		}
		if r.Header.Get("Authorization") != "Bearer service-sts" {
			t.Error("METADATA_STS_MISSING")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["expected_version"] != float64(1) {
			t.Error("METADATA_CAS_PRECONDITION_LOST")
		}
		_, _ = w.Write([]byte(`{"data":{"payload":{"uuid":"00000000-0000-4000-8000-000000000001","code":"basketball","status":"enabled","label_i18n":{"zh-CN":"fixture"},"metadata":{"kind":"term","dimension":"sport","version":2,"parent_uuid":"00000000-0000-4000-8000-000000000002"}}}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	expected := int64(1)
	attrs := map[string]any{"version": 2, "kind": "term", "dimension": "sport"}
	item, err := client.UpdateDictionaryItem(context.Background(), UpdateDictionaryItemRequest{ItemUUID: "00000000-0000-4000-8000-000000000001", ExpectedVersion: &expected, Metadata: &attrs})
	if err != nil {
		t.Fatal(err)
	}
	if item.Metadata["dimension"] != "sport" || item.Metadata["version"] != float64(2) || item.Metadata["parent_uuid"] == nil {
		t.Fatal("METADATA_EXTENSION_LOST")
	}
}

func TestHostClientNativeMetadataUpdatesPreserveAttributesAndLocks(t *testing.T) {
	attrs := map[string]any{"version": 2, "aliases": map[string]any{"en": "fixture_alias"}}
	expected := int64(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		m, ok := body["metadata"].(map[string]any)
		if !ok || m["version"] != float64(2) {
			t.Fatalf("missing metadata: %#v", body)
		}
		switch r.URL.Path {
		case "/api/v1/tenant/metadata/taxonomy-nodes/00000000-0000-4000-8000-000000000021":
			if body["version"] != float64(1) || body["move_parent"] != true || body["parent_uuid"] != nil {
				t.Fatalf("invalid atomic move: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"payload": TaxonomyNode{UUID: "00000000-0000-4000-8000-000000000021", Version: 2, Metadata: attrs}}})
		case "/api/v1/tenant/metadata/tags/00000000-0000-4000-8000-000000000022":
			if body["expected_version"] != float64(1) {
				t.Fatalf("missing tag CAS: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"payload": Tag{UUID: "00000000-0000-4000-8000-000000000022", Metadata: attrs}}})
		default:
			t.Fatalf("unexpected route: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "fixture", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	node, err := client.UpdateTaxonomyNode(context.Background(), UpdateTaxonomyNodeRequest{NodeUUID: "00000000-0000-4000-8000-000000000021", Version: 1, MoveParent: true, Metadata: &attrs})
	if err != nil {
		t.Fatal(err)
	}
	if node.Version != 2 || node.Metadata["version"] != float64(2) {
		t.Fatalf("node response: %#v", node)
	}
	tag, err := client.UpdateTag(context.Background(), UpdateTagRequest{TagUUID: "00000000-0000-4000-8000-000000000022", ExpectedVersion: &expected, Metadata: &attrs})
	if err != nil {
		t.Fatal(err)
	}
	if tag.Metadata["version"] != float64(2) {
		t.Fatalf("tag response: %#v", tag)
	}
}
