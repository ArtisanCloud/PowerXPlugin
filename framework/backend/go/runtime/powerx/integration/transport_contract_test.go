package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransportFailureAndEnvelopeContract(t *testing.T) {
	for _, tc := range []struct {
		status   int
		body     string
		expected int
	}{
		{200, "{}", 502}, {200, `{"data":null}`, 502}, {200, "invalid", 502},
		{400, `{"reason_code":"TEST_INVALID_ARGUMENT"}`, 400},
		{401, `{"error_code":"TEST_UNAUTHORIZED"}`, 401},
		{403, `{"error":{"reason_code":"TEST_FORBIDDEN"}}`, 403},
		{404, `{"reason_code":"TEST_NOT_FOUND"}`, 404},
		{502, `{"reason_code":"TEST_UPSTREAM_DEPENDENCY"}`, 502},
		{503, `{"reason_code":"TEST_UPSTREAM_DEPENDENCY"}`, 503},
	} {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.body), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			c := &Client{baseURL: server.URL, http: server.Client(), tokens: TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil })}
			var out map[string]any
			err := c.doJSON(context.Background(), "GET", "/test", nil, &out)
			var typed *HTTPError
			if !errors.As(err, &typed) || typed.StatusCode != tc.expected || typed.ReasonCode == "" {
				t.Fatalf("error=%#v", err)
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	c := &Client{baseURL: server.URL, http: server.Client(), tokens: TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil })}
	server.Close()
	var out map[string]any
	err := c.doJSON(context.Background(), "GET", "/test", nil, &out)
	var typed *HTTPError
	if !errors.As(err, &typed) || typed.StatusCode != 503 {
		t.Fatalf("error=%#v", err)
	}
}
