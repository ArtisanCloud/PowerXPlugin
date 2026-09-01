package agent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLifecycleClientUsesTenantScopedCoreRoutes(t *testing.T) {
	transport := &lifecycleTransport{}
	client, err := NewClient(PowerXAgentClientConfig{BaseURL: "https://core.example", BearerToken: "service-token"}, WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)

	summary, err := client.GetHealthSummary(context.Background(), "1a6df3ca-2e34-4b5a-9c11-0e9496170f3b")
	require.NoError(t, err)
	require.Equal(t, int32(98), summary.HealthScore)

	history, err := client.ListHealthHistory(context.Background(), "1a6df3ca-2e34-4b5a-9c11-0e9496170f3b", 24, 10)
	require.NoError(t, err)
	require.Len(t, history.Snapshots, 1)

	_, err = client.Freeze(context.Background(), "1a6df3ca-2e34-4b5a-9c11-0e9496170f3b", BridgeControlInput{Reason: "maintenance"})
	require.NoError(t, err)

	require.Equal(t, []string{
		"GET /api/v1/openapi/agents/1a6df3ca-2e34-4b5a-9c11-0e9496170f3b/health/summary",
		"GET /api/v1/openapi/agents/1a6df3ca-2e34-4b5a-9c11-0e9496170f3b/health/history?limit=10&range_hours=24",
		"POST /api/v1/openapi/agents/1a6df3ca-2e34-4b5a-9c11-0e9496170f3b/bridge/freeze",
	}, transport.calls)
	for _, authorization := range transport.authorizations {
		require.Equal(t, "Bearer service-token", authorization)
	}
}

func TestLifecycleClientMapsHostAuthorizationAndAvailabilityErrors(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		status := http.StatusForbidden
		if req.URL.Path == "/unavailable" {
			status = http.StatusServiceUnavailable
		}
		return &http.Response{StatusCode: status, Header: http.Header{"X-Trace-Id": []string{"trace-agent"}}, Body: io.NopCloser(strings.NewReader(`{"error":"rejected"}`)), Request: req}, nil
	})
	client, err := NewClient(PowerXAgentClientConfig{BaseURL: "https://core.example", BearerToken: "service-token"}, WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)
	_, err = client.GetHealthSummary(context.Background(), "agent-1")
	require.Equal(t, ErrCodeForbidden, err.(*Error).Code)
	require.Equal(t, "trace-agent", err.(*Error).TraceID)
	err = client.lifecycleJSON(context.Background(), http.MethodGet, "/unavailable", nil, &HealthSummary{})
	require.Equal(t, ErrCodeUnavailable, err.(*Error).Code)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type lifecycleTransport struct {
	calls          []string
	authorizations []string
}

func (t *lifecycleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls = append(t.calls, req.Method+" "+req.URL.RequestURI())
	t.authorizations = append(t.authorizations, req.Header.Get("Authorization"))
	payload := `{"data":{"agent":{"id":"1a6df3ca-2e34-4b5a-9c11-0e9496170f3b"}}}`
	switch req.URL.Path {
	case "/api/v1/openapi/agents/1a6df3ca-2e34-4b5a-9c11-0e9496170f3b/health/summary":
		payload = `{"data":{"status":"healthy","health_score":98,"metrics":{"success_rate":1}}}`
	case "/api/v1/openapi/agents/1a6df3ca-2e34-4b5a-9c11-0e9496170f3b/health/history":
		payload = `{"data":{"snapshots":[{"status":"healthy","health_score":98,"metrics":{}}]}}`
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
}
