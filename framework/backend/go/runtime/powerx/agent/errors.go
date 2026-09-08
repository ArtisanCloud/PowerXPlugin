package agent

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

const (
	ErrCodeConfigInvalid = "agent.config_invalid"
	ErrCodeAuthInvalid   = "agent.auth_invalid"
	ErrCodeStreamDecode  = "agent.stream_decode_failed"
	ErrCodeTransport     = "agent.transport_failed"
	ErrCodeUnauthorized  = "agent.unauthorized"
	ErrCodeForbidden     = "agent.forbidden"
	ErrCodeNotFound      = "agent.not_found"
	ErrCodeUnavailable   = "agent.upstream_unavailable"
	ErrCodeRateLimited   = "agent.rate_limited"
	ErrCodeDisconnected  = "agent.websocket_disconnected"
)

type Error struct {
	StatusCode int    `json:"-"`
	ReasonCode string `json:"reason_code,omitempty"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	TraceID    string `json:"trace_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func newError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func transportError(resp *http.Response) *Error {
	var raw []byte
	if resp != nil && resp.Body != nil {
		raw, _ = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	}
	return transportErrorPayload(resp, raw)
}

func transportErrorPayload(resp *http.Response, raw []byte) *Error {
	if resp == nil {
		return newError(ErrCodeTransport, "empty agent response")
	}
	code := ErrCodeTransport
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		code = ErrCodeUnauthorized
	case resp.StatusCode == http.StatusForbidden:
		code = ErrCodeForbidden
	case resp.StatusCode == http.StatusNotFound:
		code = ErrCodeNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		code = ErrCodeRateLimited
	case resp.StatusCode >= http.StatusInternalServerError:
		code = ErrCodeUnavailable
	}
	return &Error{
		StatusCode: resp.StatusCode,
		ReasonCode: hostcontract.ParseReasonCode(raw, code),
		Code:       code,
		Message:    fmt.Sprintf("agent host request failed: status=%d", resp.StatusCode),
		TraceID:    strings.TrimSpace(resp.Header.Get("X-Trace-ID")),
		RequestID:  strings.TrimSpace(resp.Header.Get("X-Request-ID")),
	}
}
