package publisher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRequestTracePersistsAcrossReloadAndLostResponse(t *testing.T) {
	f := &reviewFixture{lost: true}
	e, input := reviewEngine(t, f)
	s, err := e.StartSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	s, err = e.WaitSubmission(s.SubmissionID)
	if err != nil || s.Stage != "succeeded" {
		t.Fatal(s.Stage, err)
	}
	page, err := e.RequestTraces("submissions", s.SubmissionID, 1, 100)
	if err != nil || page.Total < 10 {
		t.Fatal(page.Total, err)
	}
	posts := 0
	ids := map[string]bool{}
	for _, trace := range page.Items {
		if trace.OperationID != s.OperationID || ids[trace.ClientRequestID] || trace.FinishedAt == nil {
			t.Fatal("trace_binding")
		}
		ids[trace.ClientRequestID] = true
		if trace.Method == http.MethodPost {
			posts++
			if trace.HTTPStatus != 0 || trace.ServerRequestID != "" {
				t.Fatal("lost_response_invented")
			}
		} else if !canonicalUUID(trace.ServerRequestID) {
			t.Fatal("server_request_id_missing")
		}
	}
	if posts != 1 {
		t.Fatal("duplicate_submission", posts)
	}

	server, err := Start(context.Background(), e, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	rawBootstrap, err := e.state.read("tool-session.json", MaxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap Bootstrap
	if json.Unmarshal(rawBootstrap, &bootstrap) != nil {
		t.Fatal("bootstrap")
	}
	session := exchange(t, server, bootstrap)
	route := "/v1/submissions/" + s.SubmissionID + "/requests?page=2&size=2"
	local := decodeData[RequestTracePage](t, request(t, server.Info().Endpoint, http.MethodGet, route, session.Token, ""), http.StatusOK)
	if len(local.Items) != 2 || local.Total != page.Total {
		t.Fatal("trace_api_pagination")
	}
	bad := request(t, server.Info().Endpoint, http.MethodGet, route+"&size=3", session.Token, "")
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatal("duplicate_query_accepted")
	}
	reloaded, err := Open(e.root)
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()
	second, err := reloaded.RequestTraces("submissions", s.SubmissionID, 2, 2)
	if err != nil || second.Total != page.Total || len(second.Items) != 2 || second.Items[0].ClientRequestID != page.Items[2].ClientRequestID {
		t.Fatal("trace_reload_pagination", err)
	}
	if _, err = e.RequestTraces("submissions", uuid.NewString(), 1, 20); AsError(err).Code != NotFound {
		t.Fatal("unknown_task", err)
	}
	d, err := e.requestTraceDir("submissions", s.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	raw, _ := json.Marshal(map[string]any{"schema": RequestTraceSchema, "token": "forbidden_field"})
	if err = d.write(page.Items[0].ClientRequestID+".json", raw, true); err != nil {
		t.Fatal(err)
	}
	if _, err = e.RequestTraces("submissions", s.SubmissionID, 1, 20); AsError(err).Code != InvalidRecord {
		t.Fatal("corrupt_trace_accepted", err)
	}
}

func TestMarketErrorPreservesOnlyCanonicalServerRequestID(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(fmtInt(map[bool]int{true: 1, false: 0}[valid]), func(t *testing.T) {
			id := uuid.NewString()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("X-Request-ID", id)
				if !valid {
					w.Header().Set("X-Request-ID", "invalid-private-value")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(403)
				_, _ = w.Write([]byte(`{"code":403,"error":"publisher_forbidden","message":"private upstream prose","retryable":false}`))
			}))
			defer server.Close()
			_, err := marketRequest(context.Background(), Market{MarketInput: MarketInput{APIRoot: server.URL}}, "pxpub_"+strings.Repeat("A", 43), "/publisher/me", "MeResponse", nil, nil)
			result := AsError(err)
			if valid && (result.Code != MarketRejected || result.Params["request_id"] != id) {
				t.Fatal("request_id_not_preserved", result)
			}
			raw, _ := json.Marshal(result)
			if strings.Contains(string(raw), "private") || strings.Contains(string(raw), "pxpub_") || !valid && result.Params["request_id"] != "" {
				t.Fatal("untrusted_diagnostic_leak")
			}
		})
	}
}
