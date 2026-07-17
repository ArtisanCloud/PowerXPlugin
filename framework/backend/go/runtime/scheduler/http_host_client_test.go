package scheduler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPHostClientUsesLocalProxyAPIKey(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotQuery string
	var gotTenantHeader string
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		APIPrefix:  "/api/v1",
		AuthScheme: "apikey",
		APIKey:     "local-proxy-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotAuth = req.Header.Get("Authorization")
			gotPath = req.URL.Path
			gotQuery = req.URL.RawQuery
			gotTenantHeader = req.Header.Get("X-Tenant-UUID")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"items":[],"pagination":{"total":0}}}`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}

	jobs, err := client.ListJobs(context.Background(), ListJobsInput{TenantUUID: "tenant-001"})
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("ListJobs() len = %d, want 0", len(jobs))
	}
	if gotAuth != "ApiKey local-proxy-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPath != "/api/v1/admin/scheduler/jobs" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("query = %q, want empty tenant query", gotQuery)
	}
	if gotTenantHeader != "" {
		t.Fatalf("X-Tenant-UUID = %q, want empty", gotTenantHeader)
	}
}

func TestHTTPHostClientCreatesHostSchedulerJob(t *testing.T) {
	var gotPath string
	var gotBody string
	var gotTenantHeader string
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		APIPrefix:  "/api/v1",
		AuthScheme: "bearer",
		Token:      "sts-token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotPath = req.URL.Path
			gotTenantHeader = req.Header.Get("X-Tenant-UUID")
			raw, _ := io.ReadAll(req.Body)
			gotBody = string(raw)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"job":{"uuid":"sch-host-001","tenant_uuid":"tenant-001","owner_type":"plugin","owner_id":"com.powerx.plugins.ai-craft","name":"debug","schedule_type":"once","schedule_expr":"2026-05-20T10:00:00Z","status":"active"}}}`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}

	job, err := client.CreateJob(context.Background(), JobSpec{
		TenantUUID:   "tenant-001",
		OwnerType:    OwnerTypePlugin,
		OwnerID:      "com.powerx.plugins.ai-craft",
		Name:         "debug",
		ScheduleType: ScheduleTypeOnce,
		ScheduleExpr: "2026-05-20T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if gotPath != "/api/v1/admin/scheduler/jobs" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"owner_id":"com.powerx.plugins.ai-craft"`) {
		t.Fatalf("body = %s", gotBody)
	}
	if strings.Contains(gotBody, "tenant_uuid") {
		t.Fatalf("body should not include tenant_uuid: %s", gotBody)
	}
	if gotTenantHeader != "" {
		t.Fatalf("X-Tenant-UUID = %q, want empty", gotTenantHeader)
	}
	if job.JobID != "sch-host-001" {
		t.Fatalf("job_id = %q", job.JobID)
	}
}

func TestHTTPHostClientCreatesHostSchedulerJobWithoutTenant(t *testing.T) {
	var gotBody string
	var gotTenantHeader string
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		APIPrefix:  "/api/v1",
		AuthScheme: "apikey",
		APIKey:     "local-proxy-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotTenantHeader = req.Header.Get("X-Tenant-UUID")
			raw, _ := io.ReadAll(req.Body)
			gotBody = string(raw)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"job_id":"sch-host-001","owner_type":"plugin","owner_id":"com.powerx.plugins.ai-craft","name":"debug","schedule_type":"once","schedule_expr":"2026-05-20T10:00:00Z","status":"active"}}`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}

	job, err := NewHostProvider(HostProviderConfig{TenantUUID: "tenant-001"}, client).CreateJob(context.Background(), JobSpec{
		OwnerType:    OwnerTypePlugin,
		OwnerID:      "com.powerx.plugins.ai-craft",
		Name:         "debug",
		ScheduleType: ScheduleTypeOnce,
		ScheduleExpr: "2026-05-20T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if strings.Contains(gotBody, "tenant_uuid") {
		t.Fatalf("body should not include tenant_uuid: %s", gotBody)
	}
	if gotTenantHeader != "" {
		t.Fatalf("X-Tenant-UUID = %q, want empty", gotTenantHeader)
	}
	if job.JobID != "sch-host-001" {
		t.Fatalf("job_id = %q", job.JobID)
	}
}

func TestHTTPHostClientTriggerUsesPowerXActionContract(t *testing.T) {
	var gotBody []byte
	var gotContentType string
	var gotTenantHeader string
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		AuthScheme: "apikey",
		APIKey:     "local-proxy-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotContentType = req.Header.Get("Content-Type")
			gotTenantHeader = req.Header.Get("X-Tenant-UUID")
			if req.Body != nil {
				gotBody, _ = io.ReadAll(req.Body)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{}}`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}

	if err := client.TriggerJob(context.Background(), "sch-host-001", "tenant-001"); err != nil {
		t.Fatalf("TriggerJob() error = %v", err)
	}
	if len(gotBody) != 0 {
		t.Fatalf("body = %s, want empty", string(gotBody))
	}
	if gotContentType != "" {
		t.Fatalf("Content-Type = %q, want empty", gotContentType)
	}
	if gotTenantHeader != "" {
		t.Fatalf("X-Tenant-UUID = %q, want empty", gotTenantHeader)
	}
}

func TestHTTPHostClientAcceptsBareListResponse(t *testing.T) {
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		AuthScheme: "apikey",
		APIKey:     "local-proxy-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"items":[{"job_id":"sch-001","tenant_uuid":"tenant-001","name":"debug","schedule_type":"once","schedule_expr":"2026-05-20T10:00:00Z","status":"active"}],"total":1}`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}

	jobs, err := client.ListJobs(context.Background(), ListJobsInput{TenantUUID: "tenant-001"})
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("ListJobs() len = %d, want 1", len(jobs))
	}
	if jobs[0].JobID != "sch-001" {
		t.Fatalf("job_id = %q", jobs[0].JobID)
	}
}

func TestHTTPHostClientReturnsHostRequestError(t *testing.T) {
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		AuthScheme: "apikey",
		APIKey:     "local-proxy-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`404 page not found`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}

	_, err = client.ListJobs(context.Background(), ListJobsInput{TenantUUID: "tenant-001"})
	if err == nil {
		t.Fatal("ListJobs() expected error")
	}
	var hostErr *HostRequestError
	if !errors.As(err, &hostErr) {
		t.Fatalf("error type = %T, want HostRequestError", err)
	}
	if hostErr.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", hostErr.StatusCode)
	}
	if hostErr.Endpoint != "http://powerx.local/api/v1/admin/scheduler/jobs" {
		t.Fatalf("endpoint = %q", hostErr.Endpoint)
	}
}

func TestHTTPHostClientUsesBearerTokenProvider(t *testing.T) {
	client, err := NewHTTPHostClient(HostProviderConfig{
		BaseURL:    "http://powerx.local",
		AuthScheme: "bearer",
		TokenProvider: func(ctx context.Context) (string, error) {
			return "sts-token", nil
		},
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "Bearer sts-token" {
				t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"items":[],"pagination":{"total":0}}}`)),
				Request:    req,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPHostClient() error = %v", err)
	}
	if _, err := client.ListJobs(context.Background(), ListJobsInput{}); err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
}
