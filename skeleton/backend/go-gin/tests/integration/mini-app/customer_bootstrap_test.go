package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiniAppCustomerBootstrapResolveTenantHint(t *testing.T) {
	engine, _ := setupMiniAppAuthRouter(t)
	body, _ := json.Marshal(map[string]any{
		"tenant_hint": "00000000-0000-0000-0000-000000000001",
		"channel":     "wechat",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mini-app/bootstrap/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
