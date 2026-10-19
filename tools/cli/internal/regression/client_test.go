package regression

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientUUIDDedupAndSSE(t *testing.T) {
	run, tenant, event, request, profile, caseID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	requests := 0
	detail := Detail{Run: Run{UUID: run, TenantUUID: tenant, Phase: "running", Verdict: "pending", CleanupState: "pending", Revision: 1}, LastEventUUID: &event, CaseExecutions: json.RawMessage(`[]`)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture.token" {
			t.Error("REGRESSION_INVALID_REQUEST")
		}
		switch r.URL.Path {
		case "/admin/regression/runs":
			var body map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&body)
			var id string
			_ = json.Unmarshal(body["client_request_uuid"], &id)
			if id != request {
				t.Error(id)
			}
			requests++
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": detail})
		case "/admin/regression/runs/" + run + "/events":
			if r.URL.Query().Get("after_event_uuid") != event {
				t.Error(event)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			final := detail.Run
			final.Phase = "completed"
			final.Verdict = "blocked"
			final.Revision = 2
			final.CleanupState = "not_required"
			_, _ = fmt.Fprint(w, "event: powerx.regression.run.updated.v1\ndata: ")
			_ = json.NewEncoder(w).Encode(map[string]any{"protocol": "sse", "event_type": "powerx.regression.run.updated.v1", "plugin_id": "com.powerx.fixture", "tenant_uuid": tenant, "trace_id": "fixture.trace", "request_id": "fixture.request", "payload": map[string]any{"uuid": uuid.NewString(), "tenant_uuid": tenant, "run_uuid": run, "revision": 2, "sequence": 2, "meta": map[string]any{"tenant_uuid": tenant, "source_plugin": "com.powerx.fixture", "trace_id": "fixture.trace", "request_id": "fixture.request", "occurred_at": time.Now().UTC(), "payload_version": "v1"}, "payload": map[string]any{"run": final}}})
			_, _ = fmt.Fprint(w, "\n")
			detail.Run = final
		case "/admin/regression/runs/" + run:
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": detail})
		default:
			t.Error(r.URL.Path)
		}
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, Token: "fixture.token", PluginID: "com.powerx.fixture", HTTP: server.Client()}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := client.Create(ctx, request, profile, []string{caseID}, json.RawMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	if requests != 2 {
		t.Fatal(requests)
	}
	if _, err := client.Create(ctx, request, "1", []string{caseID}, json.RawMessage(`{}`)); err == nil {
		t.Fatal("REGRESSION_INVALID_REQUEST")
	}
	got, err := client.Wait(ctx, detail)
	coded, ok := err.(*Error)
	if !ok || coded.ExitCode() != 2 || got.Run.Verdict != "blocked" {
		t.Fatal(got, err)
	}
}
func TestClientStreamDisconnectDoesNotCancelOrPoll(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" {
			t.Error(r.Method)
		}
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer server.Close()
	c := &Client{BaseURL: server.URL, Token: "fixture.token", PluginID: "com.powerx.fixture"}
	d := Detail{Run: Run{UUID: uuid.NewString(), TenantUUID: uuid.NewString(), Phase: "running"}}
	_, err := c.Wait(context.Background(), d)
	coded, ok := err.(*Error)
	if !ok || coded.ExitCode() != 3 || coded.RunUUID != d.Run.UUID || calls != 1 {
		t.Fatal(err, calls)
	}
}

func TestClientConnectionFailureKeepsRecoverableRunUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	c := &Client{BaseURL: server.URL, Token: "fixture.token", PluginID: "com.powerx.fixture"}
	d := Detail{Run: Run{UUID: uuid.NewString(), TenantUUID: uuid.NewString(), Phase: "running"}}
	got, err := c.Wait(context.Background(), d)
	coded, ok := err.(*Error)
	if !ok || coded.RunUUID != d.Run.UUID || got.Run.UUID != d.Run.UUID || coded.ExitCode() != 3 {
		t.Fatal(got, err)
	}
}

func TestClientRejectsInvalidEventTimeAndTenantSnapshot(t *testing.T) {
	for _, mode := range []string{"occurred_at", "tenant_uuid", "revision"} {
		t.Run(mode, func(t *testing.T) {
			run, tenant, event := uuid.NewString(), uuid.NewString(), uuid.NewString()
			body := map[string]any{"uuid": event, "tenant_uuid": tenant, "run_uuid": run, "sequence": 2, "revision": 2, "meta": map[string]any{"tenant_uuid": tenant, "source_plugin": "com.powerx.fixture", "request_id": "fixture.request", "trace_id": "fixture.trace", "occurred_at": time.Now().UTC().Format(time.RFC3339Nano), "payload_version": "v1"}, "payload": map[string]any{"run": map[string]any{"uuid": run, "tenant_uuid": tenant, "revision": 2, "phase": "completed", "verdict": "passed", "cleanup_state": "succeeded"}}}
			if mode == "occurred_at" {
				body["meta"].(map[string]any)["occurred_at"] = "fixture.invalidZ"
			}
			if mode == "tenant_uuid" {
				body["payload"].(map[string]any)["run"].(map[string]any)["tenant_uuid"] = uuid.NewString()
			}
			if mode == "revision" {
				body["revision"] = 0
				body["sequence"] = 0
				body["payload"].(map[string]any)["run"].(map[string]any)["revision"] = 0
			}
			envelope := map[string]any{"protocol": "sse", "event_type": "powerx.regression.run.updated.v1", "plugin_id": "com.powerx.fixture", "tenant_uuid": tenant, "request_id": "fixture.request", "trace_id": "fixture.trace", "payload": body}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				b, _ := json.Marshal(envelope)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
			}))
			defer server.Close()
			c := &Client{BaseURL: server.URL, Token: "fixture.token", PluginID: "com.powerx.fixture"}
			_, err := c.Wait(context.Background(), Detail{Run: Run{UUID: run, TenantUUID: tenant, Phase: "running", Revision: 1}})
			coded, ok := err.(*Error)
			if !ok || coded.Code != "REGRESSION_EVENT_META_INVALID" || coded.RunUUID != run {
				t.Fatal(err)
			}
		})
	}
}
