package agent

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTransportErrorPreservesReasonAndTrace(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 429, 502, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			response := &http.Response{StatusCode: status, Header: http.Header{"X-Trace-Id": []string{"trace"}, "X-Request-Id": []string{"request"}}, Body: io.NopCloser(strings.NewReader(`{"reason_code":"AGENT_CONTRACT_ERROR"}`))}
			out := transportError(response)
			if out.StatusCode != status || out.ReasonCode != "AGENT_CONTRACT_ERROR" || out.TraceID != "trace" || out.RequestID != "request" {
				t.Fatalf("error=%#v", out)
			}
		})
	}
}
