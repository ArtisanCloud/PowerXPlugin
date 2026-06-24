package agent

import "fmt"

const (
	ErrCodeConfigInvalid = "agent.config_invalid"
	ErrCodeAuthInvalid   = "agent.auth_invalid"
	ErrCodeStreamDecode  = "agent.stream_decode_failed"
	ErrCodeTransport     = "agent.transport_failed"
	ErrCodeDisconnected  = "agent.websocket_disconnected"
)

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	TraceID   string `json:"trace_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
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
