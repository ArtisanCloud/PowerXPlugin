package customerfw

import (
	"errors"
	"net/http"
)

type ErrorCode string

const (
	CodeCustomerTokenMissing          ErrorCode = "CUSTOMER_TOKEN_MISSING"
	CodeCustomerTokenInvalid          ErrorCode = "CUSTOMER_TOKEN_INVALID"
	CodeCustomerUnauthenticated       ErrorCode = "CUSTOMER_UNAUTHENTICATED"
	CodeCustomerTenantMismatch        ErrorCode = "CUSTOMER_TENANT_MISMATCH"
	CodeCustomerTenantRequired        ErrorCode = "CUSTOMER_TENANT_REQUIRED"
	CodeCustomerMembershipRequired    ErrorCode = "CUSTOMER_MEMBERSHIP_REQUIRED"
	CodeCustomerMembershipDisabled    ErrorCode = "CUSTOMER_MEMBERSHIP_DISABLED"
	CodeCustomerDelegateUnavailable   ErrorCode = "CUSTOMER_DELEGATE_UNAVAILABLE"
	CodeCustomerBootstrapFailed       ErrorCode = "CUSTOMER_BOOTSTRAP_FAILED"
	CodeCustomerContextMissing        ErrorCode = "CUSTOMER_CONTEXT_MISSING"
	CodeCustomerIdentitySourceBlocked ErrorCode = "CUSTOMER_IDENTITY_SOURCE_FORBIDDEN"
)

type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"-"`
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
	return &Error{Code: code, Message: message}
}

func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func CodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var fwerr *Error
	if errors.As(err, &fwerr) && fwerr != nil {
		return fwerr.Code
	}
	switch {
	case errors.Is(err, ErrCustomerContextMissing):
		return CodeCustomerContextMissing
	default:
		return CodeCustomerTokenInvalid
	}
}

func HTTPStatusForCode(code ErrorCode) int {
	switch code {
	case CodeCustomerTokenMissing, CodeCustomerTokenInvalid, CodeCustomerUnauthenticated, CodeCustomerContextMissing:
		return http.StatusUnauthorized
	case CodeCustomerTenantMismatch, CodeCustomerMembershipRequired, CodeCustomerMembershipDisabled, CodeCustomerIdentitySourceBlocked:
		return http.StatusForbidden
	case CodeCustomerTenantRequired, CodeCustomerBootstrapFailed:
		return http.StatusBadRequest
	case CodeCustomerDelegateUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusUnauthorized
	}
}

func RedactSecret(value string) string {
	if value == "" {
		return ""
	}
	return "[redacted]"
}
