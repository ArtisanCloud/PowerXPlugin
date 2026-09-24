package metadata

import (
	"context"
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
