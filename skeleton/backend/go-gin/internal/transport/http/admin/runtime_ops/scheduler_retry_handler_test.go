package runtime_ops

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func TestSchedulerRetryHandlerResumeRoleBoundaryAndAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := runtimeops.NewService()
	h := NewSchedulerRetryHandler(&app.Deps{Config: &config.Config{Operations: &config.OperationsConfig{Scheduler: config.OperationsSchedulerConfig{RetryMaxAttempts: 3, ResumeRoleRequired: "ops_admin_only"}}}}, svc)

	r := gin.New()
	r.POST("/scheduler/dispatches/:dispatchId/retry", h.Retry)
	r.POST("/scheduler/dispatches/:dispatchId/pause", h.Pause)
	r.POST("/scheduler/tickets/:ticketId/resume", h.Resume)

	dispatchID := "dispatch-handler-us3"

	for i := 0; i < 2; i++ {
		resp := postJSONToRouter(t, r, http.MethodPost, "/scheduler/dispatches/"+dispatchID+"/retry", map[string]any{"error_code": "AUTH_FORBIDDEN"})
		if resp.Code != http.StatusAccepted {
			t.Fatalf("retry attempt %d status mismatch, got=%d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}
	resp := postJSONToRouter(t, r, http.MethodPost, "/scheduler/dispatches/"+dispatchID+"/retry", map[string]any{"error_code": "AUTH_FORBIDDEN"})
	if resp.Code != http.StatusConflict {
		t.Fatalf("retry exhausted should be 409, got=%d body=%s", resp.Code, resp.Body.String())
	}

	pauseResp := postJSONToRouter(t, r, http.MethodPost, "/scheduler/dispatches/"+dispatchID+"/pause", map[string]any{"paused_job_id": "job-handler-001"})
	if pauseResp.Code != http.StatusCreated {
		t.Fatalf("pause status mismatch, got=%d body=%s", pauseResp.Code, pauseResp.Body.String())
	}
	ticketID := extractTicketID(t, pauseResp.Body.Bytes())
	if ticketID == "" {
		t.Fatal("ticket id should not be empty")
	}

	forbiddenResp := postJSONToRouter(t, r, http.MethodPost, "/scheduler/tickets/"+ticketID+"/resume", map[string]any{"operator_role": "viewer", "operator_id": "qa-user", "reason": "try"})
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("non ops/admin should be forbidden, got=%d body=%s", forbiddenResp.Code, forbiddenResp.Body.String())
	}

	resumeResp := postJSONToRouter(t, r, http.MethodPost, "/scheduler/tickets/"+ticketID+"/resume", map[string]any{"operator_role": "ops", "operator_id": "ops-user", "reason": "fixed permissions"})
	if resumeResp.Code != http.StatusOK {
		t.Fatalf("ops resume should be ok, got=%d body=%s", resumeResp.Code, resumeResp.Body.String())
	}

	audits := h.ticketService.ListResumeAudits()
	if len(audits) != 1 {
		t.Fatalf("expected 1 resume audit, got=%d", len(audits))
	}
	if audits[0].TicketID != ticketID {
		t.Fatalf("audit ticket mismatch, got=%s want=%s", audits[0].TicketID, ticketID)
	}
	if audits[0].OperatorRole != "ops" {
		t.Fatalf("audit operator role mismatch, got=%s", audits[0].OperatorRole)
	}
}

func postJSONToRouter(t *testing.T, router http.Handler, method, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func extractTicketID(t *testing.T, body []byte) string {
	t.Helper()
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			TicketID string `json:"ticket_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected success envelope, got body=%s", string(body))
	}
	return envelope.Data.TicketID
}
