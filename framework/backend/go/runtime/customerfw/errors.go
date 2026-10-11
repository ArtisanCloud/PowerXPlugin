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
	CodeCustomerForbidden             ErrorCode = "CUSTOMER_FORBIDDEN"
	CodeCustomerDelegateUnavailable   ErrorCode = "CUSTOMER_DELEGATE_UNAVAILABLE"
	CodeCustomerBootstrapFailed       ErrorCode = "CUSTOMER_BOOTSTRAP_FAILED"
	CodeCustomerContextMissing        ErrorCode = "CUSTOMER_CONTEXT_MISSING"
	CodeCustomerIdentitySourceBlocked ErrorCode = "CUSTOMER_IDENTITY_SOURCE_FORBIDDEN"
	CodeCustomerInvalidArgument       ErrorCode = "CUSTOMER_INVALID_ARGUMENT"
	CodeCustomerCredentialInvalid     ErrorCode = "CUSTOMER_CREDENTIAL_INVALID"
	CodeCustomerIdentityNotFound      ErrorCode = "CUSTOMER_IDENTITY_NOT_FOUND"
)

type Error struct {
	StatusCode int       `json:"-"`
	ReasonCode string    `json:"reason_code,omitempty"`
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Cause      error     `json:"-"`
}

// HTTPStatus preserves a formal Core error status across wrapping. Local
// errors continue to use the Framework's semantic code mapping.
func HTTPStatus(err error) int {
	var typed *Error
	if errors.As(err, &typed) && typed != nil && typed.StatusCode >= 400 && typed.StatusCode <= 599 {
		return typed.StatusCode
	}
	return HTTPStatusForCode(CodeOf(err))
}

func ReasonOf(err error) string {
	var typed *Error
	if errors.As(err, &typed) && typed != nil && typed.ReasonCode != "" {
		return typed.ReasonCode
	}
	return string(CodeOf(err))
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
	case CodeCustomerTokenMissing, CodeCustomerTokenInvalid, CodeCustomerUnauthenticated, CodeCustomerContextMissing, CodeCustomerCredentialInvalid:
		return http.StatusUnauthorized
	case CodeCustomerTenantMismatch, CodeCustomerMembershipRequired, CodeCustomerMembershipDisabled, CodeCustomerForbidden, CodeCustomerIdentitySourceBlocked:
		return http.StatusForbidden
	case CodeCustomerTenantRequired, CodeCustomerBootstrapFailed, CodeCustomerInvalidArgument:
		return http.StatusBadRequest
	case CodeCustomerIdentityNotFound:
		return http.StatusNotFound
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
