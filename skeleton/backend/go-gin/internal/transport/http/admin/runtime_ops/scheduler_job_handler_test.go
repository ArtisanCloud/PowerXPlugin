package runtime_ops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	fwscheduler "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/scheduler"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
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

func TestSchedulerJobHandlerHostModeDoesNotSendRequestTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSchedulerJobHandler(&app.Deps{
		EventEmitter: eventbridge.NewLocalEmitter(8),
		Config: &config.Config{
			Gateway: &config.GatewayConfig{TenantUUID: "tenant-from-gateway-config"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/scheduler/jobs?provider_mode=host", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	if got := handler.tenantUUID(c); got != "" {
		t.Fatalf("host tenant = %q, want empty", got)
	}
}

func TestSchedulerJobHandlerHostModeIgnoresGlobalTenantHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSchedulerJobHandler(&app.Deps{
		EventEmitter: eventbridge.NewLocalEmitter(8),
	})
	req := httptest.NewRequest(http.MethodGet, "/scheduler/jobs?provider_mode=host", nil)
	req.Header.Set("tenant_uuid", "tenant-from-global-client")
	req.Header.Set("X-Tenant-UUID", "tenant-from-global-client")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	if got := handler.tenantUUID(c); got != "" {
		t.Fatalf("host tenant from global header = %q, want empty", got)
	}
}

func TestSchedulerJobHandlerHostCreateDoesNotForwardTenantUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &captureSchedulerHostClient{}
	handler := &SchedulerJobHandler{
		localScheduler: fwscheduler.NewLocalProvider(fwscheduler.LocalProviderConfig{}),
		hostScheduler:  fwscheduler.NewHostProvider(fwscheduler.HostProviderConfig{}, fake),
		defaultTenant:  "tenant-from-gateway-config",
	}
	r := gin.New()
	r.POST("/scheduler/jobs", handler.Create)

	resp := postSchedulerJSON(t, r, "/scheduler/jobs?provider_mode=host", map[string]any{
		"provider_mode": "host",
		"tenant_uuid":   "tenant-from-browser",
		"owner_type":    "plugin",
		"owner_id":      "com.powerx.plugins.base",
		"name":          "host-create",
		"schedule_type": "once",
		"schedule_expr": "2026-05-15T10:30:00Z",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("host create status=%d body=%s", resp.Code, resp.Body.String())
	}
	if fake.created.TenantUUID != "" {
		t.Fatalf("forwarded tenant_uuid = %q, want empty", fake.created.TenantUUID)
	}
}

func TestSchedulerJobHandlerLocalProxyUsesConfiguredAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("POWERX_PROXY", "1")

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/scheduler/jobs" {
			t.Fatalf("unexpected scheduler path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"job_id":"job-1","owner_type":"plugin","owner_id":"com.powerx.plugins.base","name":"host-create","schedule_type":"once","schedule_expr":"2026-05-15T10:30:00Z","status":"active"}}`))
	}))
	defer server.Close()

	handler := NewSchedulerJobHandler(&app.Deps{
		EventEmitter: eventbridge.NewLocalEmitter(8),
		IAMMode:      iamservice.IAMModeDelegated,
		Config: &config.Config{
			Gateway: &config.GatewayConfig{
				BaseURL:    server.URL,
				APIPrefix:  "/api/v1",
				AuthScheme: "apikey",
				APIKey:     "gateway-key",
				ToolToken:  "sts-token-should-not-be-used",
			},
		},
	})
	r := gin.New()
	r.POST("/scheduler/jobs", handler.Create)

	resp := postSchedulerJSON(t, r, "/scheduler/jobs?provider_mode=host", map[string]any{
		"provider_mode": "host",
		"owner_type":    "plugin",
		"owner_id":      "com.powerx.plugins.base",
		"name":          "host-create",
		"schedule_type": "once",
		"schedule_expr": "2026-05-15T10:30:00Z",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("host create status=%d body=%s", resp.Code, resp.Body.String())
	}
	if gotAuth != "ApiKey gateway-key" {
		t.Fatalf("Authorization=%q, want ApiKey gateway-key", gotAuth)
	}
}

type captureSchedulerHostClient struct {
	created fwscheduler.JobSpec
}

func (c *captureSchedulerHostClient) CreateJob(_ context.Context, job fwscheduler.JobSpec) (*fwscheduler.Job, error) {
	c.created = job
	return &fwscheduler.Job{JobSpec: job, Status: fwscheduler.StatusActive}, nil
}

func (c *captureSchedulerHostClient) UpdateJob(_ context.Context, job fwscheduler.JobSpec) (*fwscheduler.Job, error) {
	return &fwscheduler.Job{JobSpec: job, Status: fwscheduler.StatusActive}, nil
}

func (c *captureSchedulerHostClient) PauseJob(context.Context, string, string) error { return nil }
func (c *captureSchedulerHostClient) ResumeJob(context.Context, string, string) error {
	return nil
}
func (c *captureSchedulerHostClient) TriggerJob(context.Context, string, string) error {
	return nil
}
func (c *captureSchedulerHostClient) GetJob(context.Context, string, string) (*fwscheduler.Job, error) {
	return nil, fwscheduler.ErrJobNotFound
}
func (c *captureSchedulerHostClient) ListJobs(context.Context, fwscheduler.ListJobsInput) ([]*fwscheduler.Job, error) {
	return nil, nil
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
