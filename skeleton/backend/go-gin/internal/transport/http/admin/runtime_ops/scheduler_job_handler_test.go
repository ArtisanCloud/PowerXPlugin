package runtime_ops

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func TestSchedulerJobHandlerCRUDAndActions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	schedulerJobStoreMu.Lock()
	schedulerJobStore = nil
	schedulerJobStoreMu.Unlock()

	emitter := eventbridge.NewLocalEmitter(8)
	deps := &app.Deps{
		EventEmitter: emitter,
		Config: &config.Config{
			Gateway: &config.GatewayConfig{TenantUUID: "tenant-001"},
		},
	}
	handler := NewSchedulerJobHandler(deps)
	r := gin.New()
	group := r.Group("/scheduler")
	group.GET("/jobs", handler.List)
	group.POST("/jobs", handler.Create)
	group.GET("/jobs/:jobId", handler.Get)
	group.POST("/jobs/:jobId/pause", handler.Pause)
	group.POST("/jobs/:jobId/resume", handler.Resume)
	group.POST("/jobs/:jobId/trigger", handler.Trigger)

	createResp := postSchedulerJSON(t, r, "/scheduler/jobs", map[string]any{
		"name":          "sample_progress_50",
		"schedule_type": "once",
		"schedule_expr": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		"payload": map[string]any{
			"business_action": "sample_progress_50",
			"order_id":        "order-001",
		},
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var createBody struct {
		Data struct {
			JobID      string `json:"job_id"`
			TenantUUID string `json:"tenant_uuid"`
			Topic      string `json:"topic"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if createBody.Data.JobID == "" {
		t.Fatal("job_id should not be empty")
	}
	if createBody.Data.TenantUUID != "tenant-001" {
		t.Fatalf("tenant_uuid=%q", createBody.Data.TenantUUID)
	}
	if createBody.Data.Topic != "powerx.runtime.scheduler.triggered.v1" {
		t.Fatalf("topic=%q", createBody.Data.Topic)
	}

	listResp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scheduler/jobs?tenant_uuid=tenant-001", nil)
	req.Header.Set("tenant_uuid", "tenant-other")
	r.ServeHTTP(listResp, req)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	var listBody struct {
		Data struct {
			Items []struct {
				JobID string `json:"job_id"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listBody.Data.Total != 1 || len(listBody.Data.Items) != 1 {
		t.Fatalf("list should use query tenant over conflicting header, body=%s", listResp.Body.String())
	}

	pauseResp := postSchedulerJSON(t, r, "/scheduler/jobs/"+createBody.Data.JobID+"/pause", map[string]any{})
	if pauseResp.Code != http.StatusOK {
		t.Fatalf("pause status=%d body=%s", pauseResp.Code, pauseResp.Body.String())
	}
	resumeResp := postSchedulerJSON(t, r, "/scheduler/jobs/"+createBody.Data.JobID+"/resume", map[string]any{})
	if resumeResp.Code != http.StatusOK {
		t.Fatalf("resume status=%d body=%s", resumeResp.Code, resumeResp.Body.String())
	}
	triggerResp := postSchedulerJSON(t, r, "/scheduler/jobs/"+createBody.Data.JobID+"/trigger", map[string]any{})
	if triggerResp.Code != http.StatusOK {
		t.Fatalf("trigger status=%d body=%s", triggerResp.Code, triggerResp.Body.String())
	}
	triggerAgainResp := postSchedulerJSON(t, r, "/scheduler/jobs/"+createBody.Data.JobID+"/trigger", map[string]any{})
	if triggerAgainResp.Code != http.StatusOK {
		t.Fatalf("trigger again status=%d body=%s", triggerAgainResp.Code, triggerAgainResp.Body.String())
	}
	drained := emitter.Drain()
	if len(drained) != 2 {
		t.Fatalf("manual trigger should emit every time, drained events=%d", len(drained))
	}
	getResp := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/scheduler/jobs/"+createBody.Data.JobID+"?tenant_uuid=tenant-001", nil)
	r.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResp.Code, getResp.Body.String())
	}
	var getBody struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if getBody.Data.Status != "active" {
		t.Fatalf("manual trigger should not complete once job, status=%q", getBody.Data.Status)
	}
}

func TestSchedulerJobHandlerHostModeUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	deps := &app.Deps{
		EventEmitter: eventbridge.NewLocalEmitter(8),
		Config: &config.Config{
			Gateway: &config.GatewayConfig{TenantUUID: "tenant-001"},
		},
	}
	handler := NewSchedulerJobHandler(deps)
	r := gin.New()
	r.POST("/scheduler/jobs", handler.Create)

	resp := postSchedulerJSON(t, r, "/scheduler/jobs", map[string]any{
		"provider_mode": "host",
		"name":          "sample_progress_50",
		"schedule_type": "once",
		"schedule_expr": "2026-05-15T10:30:00Z",
	})
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("host create status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("scheduler host provider is not available")) {
		t.Fatalf("expected host provider unavailable message, body=%s", resp.Body.String())
	}
}

func postSchedulerJSON(t *testing.T, r *gin.Engine, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}
