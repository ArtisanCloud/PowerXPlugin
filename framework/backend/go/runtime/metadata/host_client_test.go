package metadata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostClientUsesTenantContractAndServiceSTS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/metadata/tag-bindings" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer service-sts" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"payload":{"binding_uuid":"00000000-0000-4000-8000-000000000001","tag_uuid":"00000000-0000-4000-8000-000000000002","resource_type":"corex.customer","resource_uuid":"00000000-0000-4000-8000-000000000003"}}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.CreateTagBinding(context.Background(), CreateTagBindingRequest{TagUUID: "00000000-0000-4000-8000-000000000002", ResourceType: "corex.customer", ResourceUUID: "00000000-0000-4000-8000-000000000003"})
	if err != nil {
		t.Fatal(err)
	}
	if item.BindingUUID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("item = %#v", item)
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
