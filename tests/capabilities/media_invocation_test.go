package capabilities_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

func TestCapabilityProxySuccess(t *testing.T) {
	gatewayCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayCalled = true
		if r.URL.Path != "/tenant/invocations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if got := r.Header.Get("X-PowerX-Tenant"); got != "tenant-123" {
			t.Fatalf("unexpected tenant header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Trace-Id", "trace-media-success")
		_, _ = w.Write([]byte(`{"traceId":"trace-media-success","status":"ok","data":{"mediaId":"media-1"}}`))
	}))
	defer server.Close()

	app := bootstrap.NewApp(&bootstrap.Config{
		Gateway: bootstrap.GatewayConfig{
			BaseURL:   server.URL,
			ToolToken: "test-token",
			TenantID:  "tenant-123",
		},
	})
	recorder := newRecordingRouter()
	app.AttachRouter(recorder)
	router.RegisterFrameworkRoutes(app)

	handler := recorder.handler(http.MethodPost, router.APIPrefix+"/integration/capabilities/invoke")
	if handler == nil {
		t.Fatalf("capability proxy handler not registered")
	}

	body, _ := json.Marshal(map[string]any{
		"capabilityId": "com.corex.media.assets.manage",
		"action":       "List",
		"payload":      map[string]any{"folder": "inbox"},
	})
	ctx := newMockContext(body)
	ctx.reqHeaders["X-PowerX-Tenant"] = "tenant-123"
	ctx.reqHeaders["X-Request-ID"] = "req-success-1"

	handler(ctx)

	if !gatewayCalled {
		t.Fatalf("expected gateway to be invoked")
	}
	if ctx.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.status)
	}
	resp, ok := ctx.payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", ctx.payload)
	}
	if resp["traceId"] != "trace-media-success" {
		t.Fatalf("unexpected traceId: %v", resp["traceId"])
	}
	data, ok := resp["data"].(map[string]any)
	if !ok || data["mediaId"] != "media-1" {
		t.Fatalf("unexpected data payload: %v", resp["data"])
	}
	if ctx.respHeaders["X-Trace-Id"] != "trace-media-success" {
		t.Fatalf("missing trace header, got %v", ctx.respHeaders["X-Trace-Id"])
	}
}

func TestCapabilityProxyRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Trace-Id", "trace-media-rate")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"traceId":"trace-media-rate","status":"rate_limited","errors":[{"code":"RATE_LIMIT","message":"slow down"}]}`))
	}))
	defer server.Close()

	app := bootstrap.NewApp(&bootstrap.Config{
		Gateway: bootstrap.GatewayConfig{
			BaseURL:   server.URL,
			ToolToken: "test-token",
			TenantID:  "tenant-123",
		},
	})
	recorder := newRecordingRouter()
	app.AttachRouter(recorder)
	router.RegisterFrameworkRoutes(app)

	handler := recorder.handler(http.MethodPost, router.APIPrefix+"/integration/capabilities/invoke")
	if handler == nil {
		t.Fatalf("capability proxy handler not registered")
	}

	body, _ := json.Marshal(map[string]any{
		"capabilityId": "com.corex.media.assets.manage",
		"action":       "Create",
		"payload":      map[string]any{"assetName": "demo.pdf"},
	})
	ctx := newMockContext(body)
	ctx.reqHeaders["X-PowerX-Tenant"] = "tenant-123"

	handler(ctx)

	if ctx.status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", ctx.status)
	}
	resp, ok := ctx.payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", ctx.payload)
	}
	errorPayload, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload, got %v", resp)
	}
	if errorPayload["code"] != "RATE_LIMIT" {
		t.Fatalf("unexpected error code: %v", errorPayload["code"])
	}
	if resp["traceId"] != "trace-media-rate" {
		t.Fatalf("unexpected traceId: %v", resp["traceId"])
	}
	if ctx.respHeaders["X-Trace-Id"] != "trace-media-rate" {
		t.Fatalf("missing trace header, got %v", ctx.respHeaders["X-Trace-Id"])
	}
}

type recordingRouter struct {
	prefix   string
	handlers map[string]bootstrap.Handler
}

func newRecordingRouter() *recordingRouter {
	return &recordingRouter{
		prefix:   "",
		handlers: make(map[string]bootstrap.Handler),
	}
}

func (r *recordingRouter) Group(rel string) bootstrap.Router {
	return &recordingRouter{
		prefix:   joinRoute(r.prefix, rel),
		handlers: r.handlers,
	}
}

func (r *recordingRouter) Handle(method, p string, h bootstrap.Handler) {
	fullPath := joinRoute(r.prefix, p)
	key := strings.ToUpper(method) + " " + fullPath
	r.handlers[key] = h
}

func (r *recordingRouter) Use(...bootstrap.Middleware) {}

func (r *recordingRouter) handler(method, path string) bootstrap.Handler {
	key := strings.ToUpper(method) + " " + path
	return r.handlers[key]
}

func joinRoute(base, rel string) string {
	base = strings.TrimSpace(base)
	rel = strings.TrimSpace(rel)
	switch {
	case base == "" && rel == "":
		return "/"
	case base == "":
		if strings.HasPrefix(rel, "/") {
			return rel
		}
		return "/" + rel
	case rel == "":
		return base
	default:
		return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(rel, "/")
	}
}

type mockContext struct {
	body         []byte
	status       int
	payload      any
	reqHeaders   map[string]string
	respHeaders  map[string]string
	method       string
	currentCtx   context.Context
}

func newMockContext(body []byte) *mockContext {
	return &mockContext{
		body:        body,
		reqHeaders:  make(map[string]string),
		respHeaders: make(map[string]string),
		method:      http.MethodPost,
		currentCtx:  context.Background(),
	}
}

func (m *mockContext) Param(string) string            { return "" }
func (m *mockContext) Query(string) string            { return "" }
func (m *mockContext) BindJSON(v any) error           { return json.Unmarshal(m.body, v) }
func (m *mockContext) JSON(code int, v any)           { m.status = code; m.payload = v }
func (m *mockContext) Status(code int)                { m.status = code }
func (m *mockContext) Header(name string) string      { return m.reqHeaders[name] }
func (m *mockContext) SetHeader(name, value string)   { m.respHeaders[name] = value }
func (m *mockContext) Method() string                 { return m.method }
func (m *mockContext) Context() context.Context       { return m.currentCtx }
func (m *mockContext) SetContext(ctx context.Context) { m.currentCtx = ctx }
