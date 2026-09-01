package capability

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientUsesTenantCapabilityRegistryContract(t *testing.T) {
	transport := capabilityRoundTrip(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer service-token", req.Header.Get("Authorization"))
		switch req.URL.Path {
		case "/api/v1/tenant/capabilities":
			require.Equal(t, "page=2&page_size=10&plugin_id=com.example.plugin&protocol=rest", req.URL.RawQuery)
			return capabilityResponse(http.StatusOK, `{"data":{"items":[{"capability_id":"com.example.read","plugin_id":"com.example.plugin","plugin_version":"1.2.3","title":"Read","source":"plugin","categories":["data"],"tool_scope":["tenant"],"policy":{"prefer":"rest","fallback":["skill"]},"protocols":[{"channel":"rest","method":"GET","endpoint":"/v1/items","auth_type":"sts"}],"capabilities_hash":"cap-hash","protocol_hash":"proto-hash","status":"published"}]}}`), nil
		case "/api/v1/tenant/capabilities/resolve":
			require.Equal(t, "endpoint=%2Fv1%2Fitems%2F42&method=GET&source=plugin", req.URL.RawQuery)
			return capabilityResponse(http.StatusOK, `{"data":{"primary_match":{"capability_id":"com.example.read","plugin_id":"com.example.plugin","source":"plugin","protocol":"rest","method":"GET","pattern_endpoint":"/v1/items/:id"}}}`), nil
		case "/api/v1/tenant/invocations":
			require.Equal(t, http.MethodPost, req.Method)
			return capabilityResponse(http.StatusOK, `{"data":{"trace_id":"trace-001","status":"succeeded","protocol_used":"rest","fallback_used":false,"payload":{"id":"42"},"result":{"ok":true}}}`), nil
		case "/api/v1/tenant/invocations/trace-001":
			return capabilityResponse(http.StatusOK, `{"data":{"trace_id":"trace-001","tenant_uuid":"7c50b003-da2f-4514-aa4a-6f095fc4dc6b","capability_id":"com.example.read","protocol_used":"rest","status":"failed","error":{"code":"upstream_failed","message":"retry later"},"latency_ms":12,"event_published":true}}`), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	})
	client, err := NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) {
		return "service-token", nil
	}), &http.Client{Transport: transport})
	require.NoError(t, err)

	items, err := client.List(context.Background(), ListInput{Page: 2, PageSize: 10, PluginID: "com.example.plugin", Protocol: "rest"})
	require.NoError(t, err)
	require.Equal(t, "com.example.read", items[0].CapabilityID)
	require.Equal(t, "1.2.3", items[0].PluginVersion)
	require.Equal(t, "rest", items[0].Policy.Prefer)
	resolved, err := client.Resolve(context.Background(), ResolveInput{Method: "get", Endpoint: "/v1/items/42", Source: "plugin"})
	require.NoError(t, err)
	require.Equal(t, "/v1/items/:id", resolved.PatternEndpoint)
	invoked, err := client.Invoke(context.Background(), InvokeInput{CapabilityID: "com.example.read", Payload: map[string]any{"id": "42"}})
	require.NoError(t, err)
	require.Equal(t, "trace-001", invoked.TraceID)
	require.Equal(t, "42", invoked.Payload["id"])
	trace, err := client.GetInvocation(context.Background(), "trace-001")
	require.NoError(t, err)
	require.Equal(t, "com.example.read", trace.CapabilityID)
	require.Equal(t, "upstream_failed", trace.Error.Code)
}

func TestClientReturnsTypedHTTPError(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: capabilityRoundTrip(func(*http.Request) (*http.Response, error) {
		return capabilityResponse(http.StatusForbidden, `{"reason_code":"CAPABILITY_FORBIDDEN"}`), nil
	})})
	require.NoError(t, err)

	_, err = client.List(context.Background(), ListInput{})
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	require.Equal(t, http.StatusForbidden, httpErr.StatusCode)
	require.Contains(t, httpErr.Body, "CAPABILITY_FORBIDDEN")
}

type capabilityRoundTrip func(*http.Request) (*http.Response, error)

func (f capabilityRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func capabilityResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
