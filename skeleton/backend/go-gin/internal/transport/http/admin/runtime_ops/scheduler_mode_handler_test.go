package runtime_ops

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	runtimeops "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/gin-gonic/gin"
)

func TestSchedulerModeHandlerValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           map[string]any
		expectStatus   int
		expectValid    *bool
		expectCode     string
		expectExecMode string
	}{
		{
			name: "valid standalone",
			body: map[string]any{
				"powerx_proxy":     "0",
				"taskbus_provider": "redis",
			},
			expectStatus:   http.StatusOK,
			expectValid:    boolPtr(true),
			expectExecMode: runtimeops.ExecutionModeStandaloneLocal,
		},
		{
			name: "valid delegated",
			body: map[string]any{
				"powerx_proxy":     "1",
				"taskbus_provider": "host",
			},
			expectStatus:   http.StatusOK,
			expectValid:    boolPtr(true),
			expectExecMode: runtimeops.ExecutionModeDelegatedProxy,
		},
		{
			name: "conflict",
			body: map[string]any{
				"powerx_proxy":     "1",
				"taskbus_provider": "redis",
			},
			expectStatus: http.StatusConflict,
			expectCode:   "mode_conflict",
		},
		{
			name: "missing provider",
			body: map[string]any{
				"powerx_proxy": "1",
			},
			expectStatus: http.StatusBadRequest,
			expectCode:   "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router := gin.New()
			h := NewSchedulerModeHandler(nil, runtimeops.NewService())
			router.POST("/scheduler/mode/validate", h.Validate)

			payload, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("marshal body failed: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/scheduler/mode/validate", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectStatus {
				t.Fatalf("status mismatch, got=%d want=%d, body=%s", recorder.Code, tt.expectStatus, recorder.Body.String())
			}

			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal response failed: %v", err)
			}

			if tt.expectValid != nil {
				value, ok := body["valid"].(bool)
				if !ok {
					t.Fatalf("response missing valid field: %#v", body)
				}
				if value != *tt.expectValid {
					t.Fatalf("valid mismatch, got=%v want=%v", value, *tt.expectValid)
				}
				mode, _ := body["execution_mode"].(string)
				if mode != tt.expectExecMode {
					t.Fatalf("execution_mode mismatch, got=%s want=%s", mode, tt.expectExecMode)
				}
			}

			if tt.expectCode != "" {
				code, _ := body["code"].(string)
				if code != tt.expectCode {
					t.Fatalf("code mismatch, got=%s want=%s", code, tt.expectCode)
				}
			}
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}
