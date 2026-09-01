package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientUsesTypedTenantIntegrationRoutes(t *testing.T) {
	transport := &integrationTransport{}
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: transport})
	require.NoError(t, err)

	routes, err := client.ListRoutes(context.Background(), ListRoutesInput{CapabilityID: "com.example.run", Channel: "http"})
	require.NoError(t, err)
	require.Equal(t, "d4c268d4-8f16-42a1-b1fa-ed0a0f3cd8cb", routes[0].RouteUUID)

	route, err := client.GetRoute(context.Background(), "example-run")
	require.NoError(t, err)
	require.Equal(t, "com.example.run", route.CapabilityID)

	result, err := client.InvokeRoute(context.Background(), "example-run", InvokeRouteInput{Payload: map[string]any{"input": "value"}})
	require.NoError(t, err)
	require.Equal(t, "trace-001", result.TraceID)

	require.Equal(t, []string{
		"GET /api/v1/tenant/integration/routes?capability_id=com.example.run&channel=http",
		"GET /api/v1/tenant/integration/routes/example-run",
		"POST /api/v1/tenant/integration/routes/example-run/invoke",
	}, transport.calls)
}

type integrationTransport struct{ calls []string }

func (t *integrationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls = append(t.calls, req.Method+" "+req.URL.RequestURI())
	payload := `{"data":{}}`
	switch req.URL.Path {
	case "/api/v1/tenant/integration/routes":
		payload = `{"data":{"items":[{"route_id":"d4c268d4-8f16-42a1-b1fa-ed0a0f3cd8cb","route_slug":"example-run","capability_id":"com.example.run","channels":["http"]}]}}`
	case "/api/v1/tenant/integration/routes/example-run":
		payload = `{"data":{"route_id":"d4c268d4-8f16-42a1-b1fa-ed0a0f3cd8cb","route_slug":"example-run","capability_id":"com.example.run","rate_limit":{"limit":10,"burst":5,"window_seconds":60,"scope":"tenant"}}}`
	case "/api/v1/tenant/integration/routes/example-run/invoke":
		payload = `{"data":{"result":{"ok":true},"routed_capability_id":"com.example.run","trace_id":"trace-001"}}`
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
}
