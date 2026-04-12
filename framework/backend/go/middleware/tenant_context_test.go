package middleware

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

func TestTenantContextPriority_ContextFirst(t *testing.T) {
	ctx := newStubContext()
	ctx.SetContext(WithTenantUUID(context.Background(), "11111111-1111-1111-1111-111111111111"))
	ctx.SetHeader("Authorization", "Bearer "+buildTenantJWT("22222222-2222-2222-2222-222222222222"))
	ctx.SetHeader(tenantHeaderName, "33333333-3333-3333-3333-333333333333")

	called := false
	TenantContext()(func(c bootstrap.Context) {
		called = true
	})(ctx)
	if !called {
		t.Fatalf("expected next to be called")
	}
	tenantID, ok := TenantUUIDFromContext(ctx.Context())
	if !ok || tenantID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("tenant mismatch, got=%s ok=%v", tenantID, ok)
	}
	if got := ctx.Header(tenantHeaderName); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("tenant header mismatch, got=%s", got)
	}
}

func TestTenantContextPriority_ContextOverridesHeaderWhenConflict(t *testing.T) {
	ctx := newStubContext()
	ctx.SetContext(WithTenantUUID(context.Background(), "11111111-1111-1111-1111-111111111111"))
	ctx.SetHeader(tenantHeaderName, "22222222-2222-2222-2222-222222222222")

	called := false
	TenantContext()(func(c bootstrap.Context) {
		called = true
	})(ctx)
	if !called {
		t.Fatalf("expected next to be called")
	}
	if ctx.status != 0 {
		t.Fatalf("expected no error status, got=%d", ctx.status)
	}
	tenantID, ok := TenantUUIDFromContext(ctx.Context())
	if !ok || tenantID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("tenant mismatch, got=%s ok=%v", tenantID, ok)
	}
}

func buildTenantJWT(tenantUUID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"tid":"%s"}`, tenantUUID)))
	return header + "." + payload + "."
}
