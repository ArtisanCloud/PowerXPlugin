package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

type stubContext struct {
	params map[string]string
	query  map[string]string
	body   any

	headers http.Header
	status  int
	payload any
	ctx     context.Context
	method  string
}

func newStubContext() *stubContext {
	return &stubContext{
		headers: make(http.Header),
		ctx:     context.Background(),
		method:  http.MethodGet,
	}
}

func (s *stubContext) Param(name string) string {
	if s.params == nil {
		return ""
	}
	return s.params[name]
}

func (s *stubContext) Query(name string) string {
	if s.query == nil {
		return ""
	}
	return s.query[name]
}

func (s *stubContext) BindJSON(v any) error { return nil }

func (s *stubContext) JSON(code int, v any) {
	s.status = code
	s.payload = v
}

func (s *stubContext) Status(code int) { s.status = code }

func (s *stubContext) Header(name string) string {
	return s.headers.Get(name)
}

func (s *stubContext) SetHeader(name, value string) {
	if value == "" {
		s.headers.Del(name)
		return
	}
	s.headers.Set(name, value)
}

func (s *stubContext) Method() string {
	return s.method
}

func (s *stubContext) Context() context.Context {
	return s.ctx
}

func (s *stubContext) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func TestRequestIDMiddleware(t *testing.T) {
	ctx := newStubContext()
	var captured string

	RequestID()(func(c bootstrap.Context) {
		captured = RequestIDFromContext(c.Context())
	})(ctx)

	if ctx.Header(requestIDHeader) == "" {
		t.Fatalf("expected header %s to be set", requestIDHeader)
	}
	if captured == "" {
		t.Fatalf("expected context to contain request id")
	}
}

func TestTenantContextMiddleware_Default(t *testing.T) {
	ctx := newStubContext()
	TenantContext()(func(c bootstrap.Context) {})(ctx)

	if ctx.Header(tenantHeaderName) != defaultTenantUUID {
		t.Fatalf("expected default tenant header %s, got %s", defaultTenantUUID, ctx.Header(tenantHeaderName))
	}
	id, ok := TenantUUIDFromContext(ctx.Context())
	if !ok || id != defaultTenantUUID {
		t.Fatalf("expected tenant id %s from context, got %s (ok=%v)", defaultTenantUUID, id, ok)
	}
}

func TestTenantContextMiddleware_Invalid(t *testing.T) {
	ctx := newStubContext()
	ctx.SetHeader(tenantHeaderName, "abc")

	TenantContext()(func(c bootstrap.Context) {
		t.Fatalf("next handler should not be called on invalid tenant id")
	})(ctx)

	if ctx.status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", ctx.status)
	}
	env, ok := ctx.payload.(router.Envelope)
	if !ok || env.Error == nil || env.Error.Code != "INVALID_TENANT_UUID" {
		t.Fatalf("unexpected payload: %#v", ctx.payload)
	}
}

func TestCORSMiddleware_AllowsOrigin(t *testing.T) {
	ctx := newStubContext()
	ctx.SetHeader("Origin", "http://localhost:3031")

	CORS()(func(c bootstrap.Context) {
		c.Status(http.StatusOK)
	})(ctx)

	if ctx.Header("Access-Control-Allow-Origin") != "http://localhost:3031" {
		t.Fatalf("expected allow origin header to echo request origin, got %q", ctx.Header("Access-Control-Allow-Origin"))
	}
	if ctx.Header("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected allow credentials header to be true")
	}
	if ctx.Header("Access-Control-Allow-Methods") == "" {
		t.Fatalf("expected allow methods header to be set")
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	ctx := newStubContext()
	ctx.method = http.MethodOptions
	ctx.SetHeader("Origin", "http://localhost:3031")

	var called bool
	CORS()(func(c bootstrap.Context) {
		called = true
	})(ctx)

	if called {
		t.Fatalf("expected middleware to short-circuit preflight")
	}
	if ctx.status != http.StatusNoContent {
		t.Fatalf("expected status 204 for preflight, got %d", ctx.status)
	}
}
