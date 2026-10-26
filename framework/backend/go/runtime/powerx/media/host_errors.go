package media

import (
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"net/http"
)

func DecodeHostHTTPError(status int, headers http.Header, raw []byte) *HostHTTPError {
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
	trace, _ := details["trace_id"].(string)
	request, _ := details["request_id"].(string)
	fallback := "MEDIA_UPSTREAM_DEPENDENCY"
	switch status {
	case 400:
		fallback = "MEDIA_INVALID_ARGUMENT"
	case 401:
		fallback = "MEDIA_UNAUTHORIZED"
	case 403:
		fallback = "MEDIA_FORBIDDEN"
	case 404:
		fallback = "MEDIA_ASSET_NOT_FOUND"
	}
	reason := hostcontract.ParseReasonCode(raw, fallback)
	return &HostHTTPError{StatusCode: status, ReasonCode: reason, ErrorCode: first(nested.ErrorCode, body.ErrorCode, nested.Code, body.Code, reason), Body: string(raw), Details: details, TraceID: first(body.TraceID, nested.TraceID, trace, headers.Get("X-Trace-ID")), RequestID: first(body.RequestID, nested.RequestID, request, headers.Get("X-Request-ID"))}
}

func derivativeError(status int, code string) *HostHTTPError {
	return &HostHTTPError{StatusCode: status, ReasonCode: code, ErrorCode: code}
}
