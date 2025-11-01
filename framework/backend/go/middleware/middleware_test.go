package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	"github.com/powerx-plugin/framework/backend/go/router"
)

type stubContext struct {
	params map[string]string
	query  map[string]string
	body   any

	headers http.Header
	status  int
	payload any
	ctx     context.Context
}

func newStubContext() *stubContext {
	return &stubContext{
		headers: make(http.Header),
		ctx:     context.Background(),
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

	if ctx.Header(tenantHeaderName) != "1" {
		t.Fatalf("expected default tenant header 1, got %s", ctx.Header(tenantHeaderName))
	}
	id, ok := TenantIDFromContext(ctx.Context())
	if !ok || id != 1 {
		t.Fatalf("expected tenant id 1 from context, got %d (ok=%v)", id, ok)
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
	if !ok || env.Error == nil || env.Error.Code != "INVALID_TENANT_ID" {
		t.Fatalf("unexpected payload: %#v", ctx.payload)
	}
}
