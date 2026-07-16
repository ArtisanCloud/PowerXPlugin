package knowledge

import (
	"errors"
	"net/http"
)

type ErrorCode string

const (
	CodeProviderUnavailable   ErrorCode = "KNOWLEDGE_PROVIDER_UNAVAILABLE"
	CodeUnauthorized          ErrorCode = "KNOWLEDGE_UNAUTHORIZED"
	CodeForbidden             ErrorCode = "KNOWLEDGE_FORBIDDEN"
	CodeConflict              ErrorCode = "KNOWLEDGE_CONFLICT"
	CodeNotFound              ErrorCode = "KNOWLEDGE_NOT_FOUND"
	CodeRateLimited           ErrorCode = "KNOWLEDGE_RATE_LIMITED"
	CodeUnsupportedCapability ErrorCode = "KNOWLEDGE_UNSUPPORTED_CAPABILITY"
	CodeTenantRequired        ErrorCode = "KNOWLEDGE_TENANT_REQUIRED"
	CodeTenantMismatch        ErrorCode = "KNOWLEDGE_TENANT_MISMATCH"
	CodeInvalidDocument       ErrorCode = "KNOWLEDGE_INVALID_DOCUMENT"
	CodeIndexFailed           ErrorCode = "KNOWLEDGE_INDEX_FAILED"
	CodeRedactionRequired     ErrorCode = "KNOWLEDGE_REDACTION_REQUIRED"
)

type Error struct {
	Code        ErrorCode      `json:"code"`
	Message     string         `json:"message"`
	Provider    string         `json:"provider,omitempty"`
	Operation   string         `json:"operation,omitempty"`
	Retryable   bool           `json:"retryable,omitempty"`
	TraceID     string         `json:"trace_id,omitempty"`
	SafeDetails map[string]any `json:"safe_details,omitempty"`
	Cause       error          `json:"-"`
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

func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: RedactString(message)}
}

func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: RedactString(message), Cause: cause}
}

func CodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var fwerr *Error
	if errors.As(err, &fwerr) && fwerr != nil {
		return fwerr.Code
	}
	return CodeProviderUnavailable
}

func HTTPStatusForCode(code ErrorCode) int {
	switch code {
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden, CodeTenantMismatch, CodeRedactionRequired:
		return http.StatusForbidden
	case CodeConflict:
		return http.StatusConflict
	case CodeNotFound:
		return http.StatusNotFound
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeProviderUnavailable:
		return http.StatusServiceUnavailable
	case CodeUnsupportedCapability, CodeTenantRequired, CodeInvalidDocument, CodeIndexFailed:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func Unsupported(operation string) *Error {
	return &Error{
		Code:      CodeUnsupportedCapability,
		Message:   "knowledge operation is unsupported",
		Operation: operation,
	}
}
