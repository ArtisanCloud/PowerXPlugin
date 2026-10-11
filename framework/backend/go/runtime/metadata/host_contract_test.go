package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataErrorMatrix(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 502, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			want := map[int]string{400: "METADATA_INVALID_ARGUMENT", 401: "METADATA_UNAUTHORIZED", 403: "METADATA_FORBIDDEN", 404: "METADATA_NOT_FOUND", 409: "METADATA_CONFLICT", 502: "METADATA_UPSTREAM_DEPENDENCY", 503: "METADATA_UPSTREAM_DEPENDENCY"}[status]
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"reason_code":%q}`, want)
			}))
			defer server.Close()
			c, _ := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
			_, err := c.ListTags(context.Background(), ListTagsRequest{})
			e, ok := err.(*Error)
			if !ok || string(e.Code) != want || e.StatusCode != status {
				t.Fatalf("error=%#v", err)
			}
		})
	}
}
func TestMetadataRejectsIncompletePayload(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":{}}`, `{"data":{"payload":null}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c, _ := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
		_, err := c.ListTags(context.Background(), ListTagsRequest{})
		server.Close()
		if CodeOf(err) != CodeDecodeFailed {
			t.Fatalf("body=%s err=%v", body, err)
		}
	}
}
func TestResolveFindsLaterPage(t *testing.T) {
	calls := 0
	out, err := resolvePages(context.Background(), func(page int) (*Page[Tag], error) {
		calls++
		code := "other"
		if page == 2 {
			code = "target"
		}
		return &Page[Tag]{Items: []Tag{{Code: code}}, Pagination: Pagination{Page: page, PageSize: 1, Total: 2}}, nil
	}, func(tag Tag) bool { return tag.Code == "target" })
	if err != nil || out.Code != "target" || calls != 2 {
		t.Fatalf("result=%v err=%v calls=%d", out, err, calls)
	}
}
func TestReplaceBindingsAcceptsEmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/api/v1/tenant/metadata/tag-bindings:replace" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"payload":{"items":[]}}}`))
	}))
	defer server.Close()
	c, _ := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
	if _, err := c.ReplaceTagBindings(context.Background(), ReplaceTagBindingsRequest{ResourceType: "corex.customer", ResourceUUID: "11111111-1111-4111-8111-111111111111", TagUUIDs: []string{}}); err != nil {
		t.Fatal(err)
	}
}
