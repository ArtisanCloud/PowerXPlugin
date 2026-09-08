package pluginrelease

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
		call                     func(*Client) error
	}{
		{"start", "POST", "/install-sessions", `{"session_uuid":"` + id + `","status":"queued"}`, func(c *Client) error {
			_, e := c.StartInstallSession(context.Background(), StartInstallSessionInput{ArtifactURI: "pxp://" + id})
			return e
		}},
		{"get", "GET", "/install-sessions/" + id, `{"session_uuid":"` + id + `","status":"running"}`, func(c *Client) error { _, e := c.GetInstallSession(context.Background(), id); return e }},
		{"stop", "POST", "/install-sessions/" + id + "/stop", `{"session_uuid":"` + id + `"}`, func(c *Client) error {
			out, e := c.StopInstallSession(context.Background(), id, StopInstallSessionInput{})
			if e == nil && out.Status != "" {
				return fmt.Errorf("invented status=%s", out.Status)
			}
			return e
		}},
		{"import", "POST", "/import-jobs", `{"job_uuid":"` + id + `","status":"queued"}`, func(c *Client) error {
			_, e := c.StartImportJob(context.Background(), StartImportJobInput{PackageUUID: id, LicenseAccepted: true})
			return e
		}},
		{"job", "GET", "/import-jobs/" + id, `{"job_uuid":"` + id + `","status":"succeeded"}`, func(c *Client) error { _, e := c.GetImportJob(context.Background(), id); return e }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != "/api/v1/tenant/plugin-release"+tc.path {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer sts" || r.Header.Get("X-Tenant-UUID") != "" {
					t.Error("credential boundary")
				}
				fmt.Fprintf(w, `{"data":%s}`, tc.body)
			}))
			defer server.Close()
			c, e := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
			if e != nil {
				t.Fatal(e)
			}
			if e = tc.call(c); e != nil {
				t.Fatal(e)
			}
		})
	}
}
