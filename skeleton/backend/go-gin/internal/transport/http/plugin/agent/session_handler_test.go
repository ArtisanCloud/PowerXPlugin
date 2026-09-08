package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	runtimeagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	fwagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const sessionID = "12345678-1234-4234-8234-123456789abc"
const agentID = "12345678-1234-4234-8234-123456789abd"
const messageID = "12345678-1234-4234-8234-123456789abe"
const invocationID = "12345678-1234-4234-8234-123456789abf"

func sessionRouter(t *testing.T, host http.HandlerFunc) *gin.Engine {
	t.Helper()
	core := httptest.NewServer(host)
	t.Cleanup(core.Close)
	client, err := fwagent.NewClientWithTokenProvider(fwagent.PowerXAgentClientConfig{BaseURL: core.URL, Mode: fwagent.ModeDelegated}, fwagent.TokenProviderFunc(func(context.Context) (string, error) { return "server-sts", nil }))
	require.NoError(t, err)
	runtime, err := runtimeagent.NewRuntime(provider.ModeDelegated, nil, client, runtimeagent.WithSessions(nil, client))
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterAPIRoutes(r.Group("/api/v1"), &app.Deps{AgentLifecycle: runtime})
	return r
}
func TestSessionRoutesUseFrameworkContract(t *testing.T) {
	now := time.Now().UTC()
	s := fwagent.ServiceSession{SessionUUID: sessionID, AgentUUID: agentID, Status: "active", Revision: 1, CreatedAt: now, UpdatedAt: now}
	m := fwagent.ServiceMessage{SessionUUID: sessionID, MessageUUID: messageID, Role: "user", Content: "test.input", Sequence: 1, CreatedAt: now}
	i := fwagent.ServiceInvocation{SessionUUID: sessionID, MessageUUID: messageID, InvocationUUID: invocationID, TraceUUID: agentID, Status: "running", CreatedAt: now, DeadlineAt: now.Add(time.Minute)}
	for _, tc := range []struct {
		method, suffix, body string
		status               int
		output               any
	}{
		{"POST", "", `{"agent_uuid":"` + agentID + `","title":"test.title"}`, 201, s},
		{"GET", "?page=1&page_size=20", "", 200, fwagent.ServiceSessionList{Items: []fwagent.ServiceSession{s}, Total: 1, Page: 1, PageSize: 20}},
		{"GET", "/" + sessionID, "", 200, s}, {"PATCH", "/" + sessionID, `{"title":"test.title"}`, 200, s},
		{"POST", "/" + sessionID + "/archive", "", 200, s}, {"DELETE", "/" + sessionID, "", 200, s},
		{"POST", "/" + sessionID + "/messages", `{"role":"user","content":"test.input"}`, 201, m},
		{"GET", "/" + sessionID + "/messages?page=1&page_size=20", "", 200, fwagent.ServiceMessageList{Items: []fwagent.ServiceMessage{m}, Total: 1, Page: 1, PageSize: 20}},
		{"POST", "/" + sessionID + "/invocations", `{"message_uuid":"` + messageID + `"}`, 202, i},
		{"GET", "/" + sessionID + "/invocations/" + invocationID, "", 200, i},
		{"POST", "/" + sessionID + "/invocations/" + invocationID + "/cancel", "", 202, i},
	} {
		t.Run(tc.method+tc.suffix, func(t *testing.T) {
			calls := 0
			r := sessionRouter(t, func(w http.ResponseWriter, req *http.Request) {
				calls++
				require.Equal(t, tc.method, req.Method)
				require.Equal(t, "/api/v1/tenant/agent/sessions"+tc.suffix, req.URL.RequestURI())
				require.Equal(t, "Bearer server-sts", req.Header.Get("Authorization"))
				require.Empty(t, req.Header.Get("X-Tenant-UUID"))
				if strings.HasSuffix(tc.suffix, "/invocations") || tc.method == "POST" && strings.HasSuffix(tc.suffix, "/messages") {
					require.Equal(t, "request-key", req.Header.Get("Idempotency-Key"))
				}
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(gin.H{"code": tc.status, "data": tc.output})
			})
			req := httptest.NewRequest(tc.method, "/api/v1/plugin/agent/sessions"+tc.suffix, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer plugin-admin")
			req.Header.Set("Idempotency-Key", "request-key")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Equal(t, 1, calls)
		})
	}
}
func TestSessionRoutesRejectOldInputsAndPreserveErrors(t *testing.T) {
	r := sessionRouter(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected Core request") })
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/sessions", `{"agent_id":"1"}`, 400},
		{"POST", "/sessions", `{"agent_uuid":"` + agentID + `","tenant_uuid":"` + agentID + `"}`, 400},
		{"POST", "/sessions", `{"agent_uuid":"` + agentID + `","title":null}`, 400},
		{"POST", "/sessions", `{"agent_uuid":"` + agentID + `","title":"a","title":"b"}`, 400},
		{"GET", "/sessions?tenant_uuid=" + agentID, "", 400},
		{"GET", "/sessions/12", "", 400},
		{"GET", "/stream/sse?q=test", "", 404},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(tc.method, "/api/v1/plugin/agent"+tc.path, strings.NewReader(tc.body)))
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
	}
	for _, status := range []int{400, 401, 403, 404, 409, 429, 502, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := sessionRouter(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"reason_code":"AGENT_SESSION_FORBIDDEN"}`)
			})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/plugin/agent/sessions/"+sessionID, nil))
			require.Equal(t, status, rec.Code)
			require.Contains(t, rec.Body.String(), "AGENT_SESSION_FORBIDDEN")
		})
	}
}

func TestSessionRoutesDoNotUseGatewayWhenRuntimeMissing(t *testing.T) {
	r := gin.New()
	RegisterAPIRoutes(r.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/plugin/agent/sessions", nil))
	require.Equal(t, 503, rec.Code)
	require.Contains(t, rec.Body.String(), "AGENT_SESSION_UNAVAILABLE")
}
func TestSessionRouteStreamsWithoutExecuting(t *testing.T) {
	now := time.Now().UTC()
	run := fwagent.ServiceInvocation{SessionUUID: sessionID, MessageUUID: messageID, InvocationUUID: invocationID, TraceUUID: agentID, Status: "succeeded", CreatedAt: now, DeadlineAt: now.Add(time.Minute), FinishedAt: &now}
	calls := 0
	r := sessionRouter(t, func(w http.ResponseWriter, req *http.Request) {
		calls++
		require.Equal(t, "GET", req.Method)
		require.True(t, strings.HasSuffix(req.URL.Path, "/events"))
		w.Header().Set("Content-Type", "text/event-stream")
		raw, _ := json.Marshal(run)
		fmt.Fprintf(w, "event: final\ndata: %s\n\nevent: end\ndata: {\"status\":\"succeeded\"}\n\n", raw)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/plugin/agent/sessions/"+sessionID+"/invocations/"+invocationID+"/events", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), "event:final")
	require.Contains(t, rec.Body.String(), "event:end")
	require.Equal(t, 1, calls)
}
