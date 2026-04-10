package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	runtimehttp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/runtime_ops"
	"github.com/gin-gonic/gin"
)

func TestSchedulerModeSwitchIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1/admin/runtime")
	deps := &app.Deps{Config: &config.Config{RuntimeOps: &config.RuntimeOpsDefaults{}}}
	runtimehttp.RegisterRoutes(api, deps)

	tests := []struct {
		name         string
		requestBody  map[string]any
		expectStatus int
		expectCode   string
	}{
		{
			name: "standalone_local success",
			requestBody: map[string]any{
				"powerx_proxy":     "0",
				"taskbus_provider": "redis",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "delegated_proxy success",
			requestBody: map[string]any{
				"powerx_proxy":     "1",
				"taskbus_provider": "host",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "mode conflict fail fast",
			requestBody: map[string]any{
				"powerx_proxy":     "1",
				"taskbus_provider": "redis",
			},
			expectStatus: http.StatusConflict,
			expectCode:   "mode_conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, r, "/api/v1/admin/runtime/scheduler/mode/validate", tt.requestBody)
			if resp.Code != tt.expectStatus {
				t.Fatalf("status mismatch, got=%d want=%d, body=%s", resp.Code, tt.expectStatus, resp.Body.String())
			}
			if tt.expectCode != "" {
				var body map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode response failed: %v", err)
				}
				code, _ := body["code"].(string)
				if code != tt.expectCode {
					t.Fatalf("code mismatch, got=%q want=%q", code, tt.expectCode)
				}
			}
		})
	}
}

func postJSON(t *testing.T, router http.Handler, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
