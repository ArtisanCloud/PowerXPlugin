package metadata

import (
	"errors"
	"net/http"
)

type ErrorCode string

const (
	CodeClientUnavailable ErrorCode = "METADATA_CLIENT_UNAVAILABLE"
	CodeInvalidRequest    ErrorCode = "METADATA_INVALID_REQUEST"
	CodeUnauthorized      ErrorCode = "METADATA_UNAUTHORIZED"
	CodeForbidden         ErrorCode = "METADATA_FORBIDDEN"
	CodeNotFound          ErrorCode = "METADATA_NOT_FOUND"
	CodeConflict          ErrorCode = "METADATA_CONFLICT"
	CodeDecodeFailed      ErrorCode = "METADATA_DECODE_FAILED"
	CodeGatewayFailed     ErrorCode = "METADATA_GATEWAY_FAILED"
)

type Error struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Operation string         `json:"operation,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	Cause     error          `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func CodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var fwerr *Error
	if errors.As(err, &fwerr) && fwerr != nil {
		return fwerr.Code
	}
	return CodeGatewayFailed
}

func HTTPStatusForCode(code ErrorCode) int {
	switch code {
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeInvalidRequest, CodeDecodeFailed:
		return http.StatusBadRequest
	case CodeClientUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadGateway
	}
}

func invalid(operation, message string) *Error {
	return &Error{Code: CodeInvalidRequest, Message: message, Operation: operation}
}
