package customerfw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthenticateRejectsBootstrapTokenTenantMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	validator := validatorFunc(func(context.Context, string, string) (*CustomerContext, error) {
		return &CustomerContext{TenantUUID: "tenant-token", CustomerUUID: "customer-a", Authenticated: true}, nil
	})
	bootstrap := BootstrapResolver(func(context.Context, BootstrapInput) (*BootstrapContext, error) {
		return &BootstrapContext{TenantUUID: "tenant-bootstrap"}, nil
	})
	r.GET("/protected", Authenticate(
		validator,
		RequireTenant(),
		WithBootstrapResolver(bootstrap, func(*gin.Context) BootstrapInput { return BootstrapInput{Scene: "scene-a"} }),
	))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}
