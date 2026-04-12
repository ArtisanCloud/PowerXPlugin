package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	runtimehttp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/runtime_ops"
	"github.com/gin-gonic/gin"
)

func TestSchedulerRetryRecoveryFlowIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1/admin/runtime")
	deps := &app.Deps{Config: &config.Config{RuntimeOps: &config.RuntimeOpsDefaults{}, Operations: &config.OperationsConfig{Scheduler: config.OperationsSchedulerConfig{RetryMaxAttempts: 3, ResumeRoleRequired: "ops_admin_only"}}}}
	runtimehttp.RegisterRoutes(api, deps)

	dispatchID := "dispatch-int-us3"
	retryPath := "/api/v1/admin/runtime/scheduler/dispatches/" + dispatchID + "/retry"
	pausePath := "/api/v1/admin/runtime/scheduler/dispatches/" + dispatchID + "/pause"

	for i := 0; i < 2; i++ {
		resp := postJSON(t, r, retryPath, map[string]any{"error_code": "AUTH_FORBIDDEN", "error_message": "topic not allowed"})
		if resp.Code != http.StatusAccepted {
			t.Fatalf("retry attempt %d status mismatch, got=%d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}

	exhaustedResp := postJSON(t, r, retryPath, map[string]any{"error_code": "AUTH_FORBIDDEN", "error_message": "topic not allowed"})
	if exhaustedResp.Code != http.StatusConflict {
		t.Fatalf("retry exhausted should return 409, got=%d body=%s", exhaustedResp.Code, exhaustedResp.Body.String())
	}

	pauseResp := postJSON(t, r, pausePath, map[string]any{"paused_job_id": "job-int-001"})
	if pauseResp.Code != http.StatusCreated {
		t.Fatalf("pause should return 201, got=%d body=%s", pauseResp.Code, pauseResp.Body.String())
	}
	ticketID := mustTicketIDFromEnvelope(t, pauseResp.Body.Bytes())

	resumePath := "/api/v1/admin/runtime/scheduler/tickets/" + ticketID + "/resume"
	forbiddenResp := postJSON(t, r, resumePath, map[string]any{"operator_role": "viewer", "operator_id": "qa", "reason": "try"})
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("viewer resume should return 403, got=%d body=%s", forbiddenResp.Code, forbiddenResp.Body.String())
	}

	resumeResp := postJSON(t, r, resumePath, map[string]any{"operator_role": "admin", "operator_id": "ops-admin-1", "reason": "permission fixed"})
	if resumeResp.Code != http.StatusOK {
		t.Fatalf("admin resume should return 200, got=%d body=%s", resumeResp.Code, resumeResp.Body.String())
	}

	retryAfterResume := postJSON(t, r, retryPath, map[string]any{"error_code": "AUTH_FORBIDDEN"})
	if retryAfterResume.Code != http.StatusAccepted {
		t.Fatalf("retry after resume should return 202, got=%d body=%s", retryAfterResume.Code, retryAfterResume.Body.String())
	}
}

func mustTicketIDFromEnvelope(t *testing.T, body []byte) string {
	t.Helper()
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			TicketID string `json:"ticket_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected success response, got=%s", string(body))
	}
	if envelope.Data.TicketID == "" {
		t.Fatalf("ticket id missing in response: %s", string(body))
	}
	return envelope.Data.TicketID
}
