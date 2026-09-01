package skills

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientInvokeUsesTenantSkillsContract(t *testing.T) {
	transport := roundTrip(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/api/v1/tenant/skills/invoke", req.URL.Path)
		require.Equal(t, "Bearer service-token", req.Header.Get("Authorization"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"skill_id":"summarize","version":"v1","payload":{"text":"hello"},"context":{"locale":"zh-CN"}}`, string(body))
		return response(http.StatusOK, `{"data":{"trace_id":"trace-001","status":"succeeded","protocol_used":"skill","fallback_used":false,"result":{"summary":"ok"}}}`), nil
	})
	client, err := NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) {
		return "service-token", nil
	}), &http.Client{Transport: transport})
	require.NoError(t, err)

	out, err := client.Invoke(context.Background(), InvokeInput{SkillID: "summarize", Version: "v1", Payload: map[string]any{"text": "hello"}, Context: map[string]any{"locale": "zh-CN"}})
	require.NoError(t, err)
	require.Equal(t, "trace-001", out.TraceID)
	require.Equal(t, "ok", out.Result["summary"])
}

func TestClientInvokeReturnsTypedHTTPError(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return response(http.StatusForbidden, `{"code":"SKILL_FORBIDDEN"}`), nil
	})})
	require.NoError(t, err)

	_, err = client.Invoke(context.Background(), InvokeInput{SkillID: "summarize"})
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	require.Equal(t, http.StatusForbidden, httpErr.StatusCode)
	require.Contains(t, httpErr.Body, "SKILL_FORBIDDEN")
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
