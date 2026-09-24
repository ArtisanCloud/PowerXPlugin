package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestMetadataUUIDsRejectBeforeTransport(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	c, _ := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
	for _, id := range []string{"", "42", "not-a-uuid", "00000000-0000-0000-0000-000000000000", "../tags"} {
		ops := []func() error{
			func() error {
				_, e := c.ListDictionaryItems(context.Background(), ListDictionaryItemsRequest{NamespaceUUID: id})
				return e
			},
			func() error {
				_, e := c.UpdateDictionaryNamespace(context.Background(), UpdateDictionaryNamespaceRequest{NamespaceUUID: id})
				return e
			},
			func() error {
				_, e := c.CreateDictionaryItem(context.Background(), CreateDictionaryItemRequest{NamespaceUUID: id, Code: "test"})
				return e
			},
			func() error {
				_, e := c.UpdateDictionaryItem(context.Background(), UpdateDictionaryItemRequest{ItemUUID: id})
				return e
			},
			func() error {
				_, e := c.ListTaxonomyNodes(context.Background(), ListTaxonomyNodesRequest{TaxonomyUUID: id})
				return e
			},
			func() error {
				_, e := c.CreateTaxonomyNode(context.Background(), CreateTaxonomyNodeRequest{TaxonomyUUID: id, Code: "test"})
				return e
			},
			func() error {
				_, e := c.UpdateTaxonomyNode(context.Background(), UpdateTaxonomyNodeRequest{NodeUUID: id, Version: 1})
				return e
			},
			func() error {
				_, e := c.ListTagBindings(context.Background(), ListTagBindingsRequest{ResourceUUID: id, ResourceType: "test"})
				return e
			},
			func() error {
				_, e := c.ReplaceTagBindings(context.Background(), ReplaceTagBindingsRequest{ResourceUUID: id, ResourceType: "test", TagUUIDs: []string{id}})
				return e
			},
			func() error { _, e := c.UpdateTag(context.Background(), UpdateTagRequest{TagUUID: id}); return e },
			func() error {
				_, e := c.UpdateResourceType(context.Background(), UpdateResourceTypeRequest{ResourceTypeUUID: id})
				return e
			},
		}
		for index, op := range ops {
			if err := op(); CodeOf(err) != CodeInvalidArgument {
				t.Fatalf("id=%q operation=%d error=%v", id, index, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("transport_calls=%d", calls.Load())
	}
}

func TestDictionaryCreateTransmitsMetadata(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Error(err)
		}
		if in.Metadata["source"] != "plugin" {
			t.Errorf("metadata=%v", in.Metadata)
		}
		fmt.Fprintf(w, `{"data":{"payload":{"uuid":%q,"namespace_uuid":%q,"code":"test"}}}`, id, id)
	}))
	defer server.Close()
	c, _ := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
	if _, err := c.CreateDictionaryItem(context.Background(), CreateDictionaryItemRequest{NamespaceUUID: id, Code: "test", Metadata: map[string]any{"source": "plugin"}}); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataRejectsIncompleteObjectAndEmptyToken(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, `{"data":{"payload":{}}}`) }))
	defer server.Close()
	for _, token := range []string{"sts", " "} {
		c, _ := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return token, nil }), server.Client())
		_, err := c.CreateDictionaryNamespace(context.Background(), CreateDictionaryNamespaceRequest{Namespace: "test", Module: "test"})
		expected := CodeDecodeFailed
		if token == " " {
			expected = CodeClientUnavailable
		}
		if CodeOf(err) != expected {
			t.Fatalf("error=%v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("transport_calls=%d", calls.Load())
	}
}
