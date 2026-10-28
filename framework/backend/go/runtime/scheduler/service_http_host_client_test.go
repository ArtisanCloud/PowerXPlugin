package scheduler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestServiceSchedulerReadPaginatesAndUsesTrustedIdentity(t *testing.T) {
	calls := 0
	client, err := NewServiceHTTPHostClient(HostProviderConfig{BaseURL: "http://core.test", Token: "sts-test", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/api/v1/tenant/scheduler/jobs" || r.URL.Query().Get("page") != fmt.Sprint(calls) || r.Header.Get("Authorization") != "Bearer sts-test" || r.URL.Query().Has("tenant_uuid") || r.Header.Get("X-Tenant-UUID") != "" {
			t.Fatalf("unexpected service request: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":200,"data":{"items":[{"job_uuid":"11111111-1111-4111-8111-111111111111","owner_type":"plugin","owner_id":"test.plugin","name":"job","status":"paused"}],"pagination":{"total":2}}}`)), Header: make(http.Header)}, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := client.ListJobs(context.Background(), ListJobsInput{OwnerType: OwnerTypePlugin, OwnerID: "test.plugin", TenantUUID: "untrusted", Limit: 100})
	if err != nil || len(jobs) != 2 || calls != 2 || jobs[0].JobID == "" || jobs[0].Status != StatusPaused {
		t.Fatalf("jobs=%v calls=%d err=%v", jobs, calls, err)
	}
}

func TestServiceSchedulerDenialNeverRetriesAdmin(t *testing.T) {
	calls := 0
	client, err := NewServiceHTTPHostClient(HostProviderConfig{BaseURL: "http://core.test", Token: "sts-test", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader(`{"error":"denied"}`)), Header: make(http.Header)}, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListJobs(context.Background(), ListJobsInput{OwnerType: OwnerTypePlugin, OwnerID: "test.plugin"}); err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
