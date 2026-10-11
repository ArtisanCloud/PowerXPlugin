package media

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostOperationRoutes(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	cases := []struct {
		name, method, path, body string
		call                     func(*HostClient) error
	}{
		{"list", "GET", "/assets", `{"items":[],"total":0,"page":1,"page_size":20}`, func(c *HostClient) error { _, e := c.ListAssets(context.Background(), ListAssetsInput{}); return e }},
		{"get", "GET", "/assets/" + id, `{"asset_uuid":"` + id + `"}`, func(c *HostClient) error { _, e := c.GetAsset(context.Background(), id); return e }},
		{"create", "POST", "/assets", `{"asset_uuid":"` + id + `"}`, func(c *HostClient) error { _, e := c.CreateAsset(context.Background(), CreateAssetInput{}); return e }},
		{"update", "PATCH", "/assets/" + id, `{"asset_uuid":"` + id + `"}`, func(c *HostClient) error {
			_, e := c.UpdateAsset(context.Background(), id, UpdateAssetInput{})
			return e
		}},
		{"delete", "DELETE", "/assets/" + id, "", func(c *HostClient) error { return c.DeleteAsset(context.Background(), id) }},
		{"upload", "POST", "/assets/" + id + "/presign-upload", `{"url":"https://storage.example/upload","method":"PUT"}`, func(c *HostClient) error { _, e := c.PresignUpload(context.Background(), id); return e }},
		{"complete", "POST", "/assets/" + id + "/complete-upload", `{"asset_uuid":"` + id + `","status":"ready"}`, func(c *HostClient) error {
			_, e := c.CompleteUpload(context.Background(), id, CompleteUploadInput{Checksum: "sha256:test"})
			return e
		}},
		{"download", "POST", "/assets/" + id + "/presign-download", `{"url":"https://storage.example/download","method":"GET"}`, func(c *HostClient) error { _, e := c.PresignDownload(context.Background(), id); return e }},
		{"variant_create", "POST", "/assets/" + id + "/variants", `{"variant_uuid":"` + id + `","asset_uuid":"` + id + `"}`, func(c *HostClient) error {
			_, e := c.CreateVariant(context.Background(), id, CreateVariantInput{VariantType: "thumbnail"})
			return e
		}},
		{"variant_get", "GET", "/assets/variants/" + id, `{"variant_uuid":"` + id + `","asset_uuid":"` + id + `"}`, func(c *HostClient) error { _, e := c.GetVariant(context.Background(), id); return e }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != "/api/v1/tenant/media"+tc.path {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer sts" || r.Header.Get("X-Tenant-UUID") != "" {
					t.Error("credential boundary")
				}
				if tc.body == "" {
					w.WriteHeader(204)
					return
				}
				fmt.Fprintf(w, `{"data":%s}`, tc.body)
			}))
			defer server.Close()
			c, e := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
			if e != nil {
				t.Fatal(e)
			}
			if e = tc.call(c); e != nil {
				t.Fatal(e)
			}
		})
	}
}
