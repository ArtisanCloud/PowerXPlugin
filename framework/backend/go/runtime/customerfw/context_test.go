package customerfw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestContextHelpersNormalizeAndRoundTrip(t *testing.T) {
	cc := &CustomerContext{
		TenantUUID:    " TENANT-A ",
		CustomerUUID:  " CUSTOMER-A ",
		Roles:         []string{"player", "player", ""},
		Source:        CustomerAuthSourceLocal,
		Authenticated: true,
	}
	ctx := WithContext(context.Background(), cc)
	got, ok := ContextFrom(ctx)
	if !ok {
		t.Fatal("expected customer context")
	}
	if got.TenantUUID != "tenant-a" || got.CustomerUUID != "customer-a" {
		t.Fatalf("unexpected normalized context: %#v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "player" {
		t.Fatalf("roles were not compacted: %#v", got.Roles)
	}
}

func TestGinContextHelpersFallbackToRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	cc := &CustomerContext{CustomerUUID: "customer-a", Authenticated: true}
	req = req.WithContext(WithContext(req.Context(), cc))
	c.Request = req

	got, ok := ContextFromGin(c)
	if !ok || got.CustomerUUID != "customer-a" {
		t.Fatalf("expected request customer context, got %#v ok=%v", got, ok)
	}
}
