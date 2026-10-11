package hostapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/cache"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskcenter"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const tenant = "00000000-0000-4000-8000-000000000001"
const task = "00000000-0000-4000-8000-000000000002"

func tokens(context.Context) (hostapi.Credential, error) {
	return hostapi.Credential{Token: "test-sts", TenantUUID: tenant}, nil
}
func TestSixHostOperations(t *testing.T) {
	taskBody := fmt.Sprintf(`{"task_uuid":%q,"tenant_uuid":%q,"type":"export.orders","state":"running","progress":20,"revision":2,"message_key":"","result":null,"created_at":"2026-09-09T00:00:00Z","updated_at":"2026-09-09T00:00:01Z","completed_at":null}`, task, tenant)
	for _, op := range []struct {
		name, method, path, body string
		keys                     int
		call                     func(cache.Service, taskcenter.Service) error
	}{
		{"cache_get", "GET", "/api/v1/tenant/runtime/cache/entries", `{"found":true,"value_base64":"AAH/","expires_at":"2026-09-10T00:00:00Z"}`, 0, func(c cache.Service, _ taskcenter.Service) error {
			e, err := c.Get(context.Background(), cache.Scope{TenantUUID: tenant, Namespace: "license"}, cache.GetInput{Key: "a & b"})
			if err == nil && (len(e.Value) != 3 || e.Value[2] != 255) {
				return errors.New("binary_mismatch")
			}
			return err
		}},
		{"cache_set", "PUT", "/api/v1/tenant/runtime/cache/entries", `{}`, 4, func(c cache.Service, _ taskcenter.Service) error {
			return c.Set(context.Background(), cache.Scope{TenantUUID: tenant, Namespace: "license"}, cache.SetInput{Key: "key", Value: []byte{0, 1, 255}, TTL: time.Minute})
		}},
		{"cache_delete", "DELETE", "/api/v1/tenant/runtime/cache/entries", `{}`, 0, func(c cache.Service, _ taskcenter.Service) error {
			return c.Delete(context.Background(), cache.Scope{TenantUUID: tenant, Namespace: "license"}, cache.DeleteInput{Key: "a & b"})
		}},
		{"task_create", "POST", "/api/v1/tenant/runtime/tasks", taskBody, 3, func(_ cache.Service, s taskcenter.Service) error {
			_, e := s.Create(context.Background(), taskcenter.Scope{TenantUUID: tenant}, taskcenter.CreateInput{Type: "export.orders", IdempotencyKey: "once", Payload: json.RawMessage(`null`)})
			return e
		}},
		{"task_get", "GET", "/api/v1/tenant/runtime/tasks/" + task, taskBody, 0, func(_ cache.Service, s taskcenter.Service) error {
			_, e := s.Get(context.Background(), taskcenter.Scope{TenantUUID: tenant}, taskcenter.GetInput{TaskUUID: task})
			return e
		}},
		{"task_update", "PATCH", "/api/v1/tenant/runtime/tasks/" + task, taskBody, 5, func(_ cache.Service, s taskcenter.Service) error {
			_, e := s.Update(context.Background(), taskcenter.Scope{TenantUUID: tenant}, taskcenter.UpdateInput{TaskUUID: task, ExpectedRevision: 1, State: taskcenter.Running, Progress: 20})
			return e
		}},
	} {
		for _, status := range []int{200, 400, 401, 403, 404, 409, 503} {
			t.Run(fmt.Sprintf("%s/%d", op.name, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != op.method || r.URL.Path != op.path || r.Header.Get("Authorization") != "Bearer test-sts" || r.Header.Get("X-Tenant-UUID") != "" {
						t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
					}
					if op.keys > 0 {
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != op.keys {
							t.Errorf("body=%v err=%v", body, err)
						}
						if _, ok := body["tenant_uuid"]; ok {
							t.Error("tenant_override")
						}
						if op.name == "cache_set" && (body["ttl_ms"] != float64(60000) || body["value_base64"] != "AAH/") {
							t.Errorf("body=%v", body)
						}
					} else if r.ContentLength > 0 {
						t.Error("unexpected_body")
					}
					if op.name == "cache_get" || op.name == "cache_delete" {
						if r.URL.Query().Get("key") != "a & b" || r.URL.Query().Get("namespace") != "license" || len(r.URL.Query()) != 2 {
							t.Error("query")
						}
					}
					w.WriteHeader(status)
					if status != 200 {
						fmt.Fprint(w, `{"error_code":"TEST_REJECTED","reason_code":"TEST_REJECTED","request_id":"trace-test"}`)
						return
					}
					fmt.Fprintf(w, `{"code":200,"data":%s}`, op.body)
				}))
				defer server.Close()
				c, err := cache.NewHostProvider(hostapi.Config{BaseURL: server.URL}, hostapi.TokenProviderFunc(tokens), server.Client())
				if err != nil {
					t.Fatal(err)
				}
				s, err := taskcenter.NewHostProvider(hostapi.Config{BaseURL: server.URL}, hostapi.TokenProviderFunc(tokens), server.Client())
				if err != nil {
					t.Fatal(err)
				}
				err = op.call(c, s)
				if status == 200 {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					var e *hostapi.HTTPError
					if !errors.As(err, &e) || e.StatusCode != status || e.ReasonCode != "TEST_REJECTED" || e.RequestID != "trace-test" {
						t.Fatalf("err=%v", err)
					}
				}
			})
		}
	}
}
func TestTenantMismatchAndInvalidValuesNeverSend(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	c, _ := cache.NewHostProvider(hostapi.Config{BaseURL: server.URL}, hostapi.TokenProviderFunc(tokens), server.Client())
	_, err := c.Get(context.Background(), cache.Scope{TenantUUID: task, Namespace: "license"}, cache.GetInput{Key: "key"})
	var e *hostapi.HTTPError
	if !errors.As(err, &e) || e.StatusCode != 403 {
		t.Fatal(err)
	}
	for _, ttl := range []time.Duration{time.Nanosecond, time.Millisecond + 1, 25 * time.Hour} {
		if err := c.Set(context.Background(), cache.Scope{TenantUUID: tenant, Namespace: "license"}, cache.SetInput{Key: "key", TTL: ttl}); !errors.Is(err, cache.ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
	s, _ := taskcenter.NewHostProvider(hostapi.Config{BaseURL: server.URL}, hostapi.TokenProviderFunc(tokens), server.Client())
	_, err = s.Create(context.Background(), taskcenter.Scope{TenantUUID: tenant}, taskcenter.CreateInput{Type: "export", IdempotencyKey: "once", Payload: json.RawMessage(`{"x":1,"x":2}`)})
	if !errors.Is(err, taskcenter.ErrInvalidArgument) {
		t.Fatal(err)
	}
	_, err = s.Update(context.Background(), taskcenter.Scope{TenantUUID: tenant}, taskcenter.UpdateInput{TaskUUID: task, ExpectedRevision: 1 << 63, State: taskcenter.Running})
	if !errors.Is(err, taskcenter.ErrInvalidArgument) {
		t.Fatal(err)
	}
	if requests != 0 {
		t.Fatalf("requests=%d", requests)
	}
}
func TestStrictJSON(t *testing.T) {
	for _, raw := range []string{`{"x":1,"x":2}`, `{"x":"\u0000"}`, `[] []`, string([]byte{'"', 255, '"'})} {
		if hostapi.ValidJSON([]byte(raw)) {
			t.Fatalf("accepted=%q", raw)
		}
	}
	if !hostapi.ValidJSON([]byte(`{"x":[null,1,"ok"]}`)) {
		t.Fatal("valid_json")
	}
}

func TestCacheEmptyMissAndMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		body  string
		found bool
		bad   bool
	}{
		{`{"code":200,"data":{"found":false,"value_base64":"","expires_at":null}}`, false, false},
		{`{"code":200,"data":{"found":true,"value_base64":"","expires_at":"2026-09-10T00:00:00Z"}}`, true, false},
		{`{"code":200,"data":{"found":true,"value_base64":"!","expires_at":"2026-09-10T00:00:00Z"}}`, false, true},
		{`{"code":200,"data":{"found":false,"value_base64":"YQ==","expires_at":null}}`, false, true},
		{`{"code":200,"data":null}`, false, true},
		{`{"code":503,"data":{}}`, false, true},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, tc.body) }))
		c, _ := cache.NewHostProvider(hostapi.Config{BaseURL: server.URL}, hostapi.TokenProviderFunc(tokens), server.Client())
		e, err := c.Get(context.Background(), cache.Scope{TenantUUID: tenant, Namespace: "license"}, cache.GetInput{Key: "key"})
		server.Close()
		if tc.bad {
			if err == nil {
				t.Fatalf("accepted=%s", tc.body)
			}
		} else if err != nil || e.Found != tc.found || len(e.Value) != 0 {
			t.Fatalf("entry=%v err=%v", e, err)
		}
	}
}
func TestTransportCancellationAndRedirect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, _ := hostapi.New(hostapi.Config{BaseURL: "http://unused.invalid"}, hostapi.TokenProviderFunc(tokens), nil, "CACHE")
	if err := client.Do(ctx, tenant, "GET", "/test", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalls++ }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, _ = hostapi.New(hostapi.Config{BaseURL: server.URL}, hostapi.TokenProviderFunc(tokens), server.Client(), "CACHE")
	var typed *hostapi.HTTPError
	if err := client.Do(context.Background(), tenant, "GET", "/test", nil, nil); !errors.As(err, &typed) || typed.StatusCode != 307 || targetCalls != 0 {
		t.Fatalf("err=%v calls=%d", err, targetCalls)
	}
}
