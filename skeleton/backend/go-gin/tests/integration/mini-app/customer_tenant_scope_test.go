package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiniAppCustomerAuth_GlobalTokenRequiresTenant(t *testing.T) {
	engine, deps := setupMiniAppAuthRouter(t)
	customerUUID := "00000000-0000-0000-0000-000000000002"
	token := signCustomerJWT(t, deps.Config.CustomerAuth.JWTSecret, "", customerUUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMiniAppCustomerAuth_RejectsMultiTokenConflict(t *testing.T) {
	engine, deps := setupMiniAppAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	tokenA := signCustomerJWT(t, deps.Config.CustomerAuth.JWTSecret, tenantUUID, "00000000-0000-0000-0000-000000000002")
	tokenB := signCustomerJWT(t, deps.Config.CustomerAuth.JWTSecret, tenantUUID, "00000000-0000-0000-0000-000000000003")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("tenant_uuid", tenantUUID)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("X-Customer-Token", tokenB)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}
