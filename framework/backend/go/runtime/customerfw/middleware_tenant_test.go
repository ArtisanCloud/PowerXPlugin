package customerfw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthenticateRequiresTenantForTenantScopedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", Authenticate(validatorFunc(func(context.Context, string, string) (*CustomerContext, error) {
		return &CustomerContext{CustomerUUID: "customer-a", Authenticated: true}, nil
	}), RequireTenant()))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestResolveTenantRejectsMismatch(t *testing.T) {
	_, err := ResolveTenant("tenant-a", "tenant-b", "", true)
	if CodeOf(err) != CodeCustomerTenantMismatch {
		t.Fatalf("expected tenant mismatch, got %v", err)
	}
}
