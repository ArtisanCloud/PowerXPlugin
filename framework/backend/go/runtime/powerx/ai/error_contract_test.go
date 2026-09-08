package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONAndSSEPreserveCoreReason(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 429, 502, 503} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", status, stream), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(status)
					fmt.Fprint(w, `{"error":{"reason_code":"AI_CONTRACT_ERROR"}}`)
				}))
				defer server.Close()
				c, e := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
				if e != nil {
					t.Fatal(e)
				}
				var err error
				if stream {
					err = c.LLMStream(context.Background(), LLMStreamInput{}, func(LLMStreamEvent) error { return nil })
				} else {
					var out any
					err = c.doJSON(context.Background(), "GET", "/test", nil, &out)
				}
				var typed *HTTPError
				if !errors.As(err, &typed) || typed.StatusCode != status || typed.ReasonCode != "AI_CONTRACT_ERROR" {
					t.Fatalf("error=%#v", err)
				}
			})
		}
	}
}
