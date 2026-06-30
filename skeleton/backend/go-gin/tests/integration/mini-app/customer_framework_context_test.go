package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiniAppCustomerAuth_InjectsFrameworkContext(t *testing.T) {
	engine, deps := setupMiniAppAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	customerUUID := "00000000-0000-0000-0000-000000000002"
	token := signCustomerJWT(t, deps.Config.CustomerAuth.JWTSecret, tenantUUID, customerUUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, _ := out["data"].(map[string]any)
	customer, _ := data["customer"].(map[string]any)
	if got, _ := customer["customer_uuid"].(string); got != customerUUID {
		t.Fatalf("expected framework customer_uuid=%s, got %#v", customerUUID, customer)
	}
	if got, _ := customer["tenant_uuid"].(string); got != tenantUUID {
		t.Fatalf("expected framework tenant_uuid=%s, got %#v", tenantUUID, customer)
	}
}
