package notifications

import (
	"encoding/json"
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

// DecodeHTTPError retains diagnostics without translating conflicts into a
// generic upstream failure. Body is diagnostic data, not a display message.
func DecodeHTTPError(status int, headers http.Header, raw []byte) *HTTPError {
	type diagnostic struct {
		ErrorCode string         `json:"error_code"`
		Code      string         `json:"code"`
		TraceID   string         `json:"trace_id"`
		RequestID string         `json:"request_id"`
		Details   map[string]any `json:"details"`
	}
	var body struct {
		diagnostic
		Error json.RawMessage `json:"error"`
	}
	_ = json.Unmarshal(raw, &body)
	var nested diagnostic
	_ = json.Unmarshal(body.Error, &nested)
	first := func(values ...string) string {
		for _, v := range values {
			if v != "" {
				return v
			}
		}
		return ""
	}
	details := body.Details
	if details == nil {
		details = nested.Details
	}
	detailTrace, _ := details["trace_id"].(string)
	detailRequest, _ := details["request_id"].(string)
	reason := hostcontract.ParseReasonCode(raw, "NOTIFICATION_UPSTREAM_DEPENDENCY")
	return &HTTPError{StatusCode: status, ReasonCode: reason, ErrorCode: first(nested.ErrorCode, body.ErrorCode, nested.Code, body.Code, reason), TraceID: first(body.TraceID, nested.TraceID, detailTrace, headers.Get("X-Trace-ID")), RequestID: first(body.RequestID, nested.RequestID, detailRequest, headers.Get("X-Request-ID")), Details: details, Body: string(raw)}
}
