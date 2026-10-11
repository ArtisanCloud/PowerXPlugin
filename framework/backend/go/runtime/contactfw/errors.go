package contactfw

import (
	"errors"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

type ErrorCode string

const (
	CodeInvalidArgument                  ErrorCode = "CONTACT_INVALID_ARGUMENT"
	CodeNotFound                         ErrorCode = "CONTACT_NOT_FOUND"
	CodeCustomerMismatch                 ErrorCode = "CONTACT_CUSTOMER_MISMATCH"
	CodeCustomerMembershipInactive       ErrorCode = "CONTACT_CUSTOMER_MEMBERSHIP_INACTIVE"
	CodeIdentityNotFound                 ErrorCode = "CONTACT_IDENTITY_NOT_FOUND"
	CodeIdentityConflict                 ErrorCode = "CONTACT_IDENTITY_CONFLICT"
	CodeChannelDictionaryInvalid         ErrorCode = "CONTACT_CHANNEL_DICTIONARY_INVALID"
	CodeIdentityChannelMigrationRequired ErrorCode = "CONTACT_IDENTITY_CHANNEL_MIGRATION_REQUIRED"
	CodeDelegateUnavailable              ErrorCode = "CONTACT_DELEGATE_UNAVAILABLE"
	CodeCapabilityForbidden              ErrorCode = "CONTACT_CAPABILITY_FORBIDDEN"
	CodeLocalUnavailable                 ErrorCode = "CONTACT_LOCAL_UNAVAILABLE"
	CodeRuntimeRequired                  ErrorCode = "CONTACT_RUNTIME_REQUIRED"
	CodePrimaryExpectedRequired          ErrorCode = "CONTACT_PRIMARY_EXPECTED_REQUIRED"
	CodePrimaryChanged                   ErrorCode = "CONTACT_PRIMARY_CHANGED"
	CodePrimaryNotActive                 ErrorCode = "CONTACT_PRIMARY_NOT_ACTIVE"
	CodePrimaryOperationRequired         ErrorCode = "CONTACT_PRIMARY_OPERATION_REQUIRED"
	CodePrimaryChangeRequired            ErrorCode = "CONTACT_PRIMARY_CHANGE_REQUIRED"
	CodePrimaryOperationFailed           ErrorCode = "CONTACT_PRIMARY_OPERATION_FAILED"
	CodeRequestInvalid                   ErrorCode = "CONTACT_REQUEST_INVALID"
)

type Error struct {
	Code       ErrorCode
	Cause      error
	StatusCode int
	TraceID    string
	RequestID  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code)
}
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
func NewError(code ErrorCode, cause error) *Error {
	out := &Error{Code: code, Cause: cause}
	var upstream *gateway.InvocationError
	if errors.As(cause, &upstream) && upstream != nil {
		out.StatusCode, out.TraceID, out.RequestID = upstream.StatusCode, upstream.TraceID, upstream.UpstreamRequestID
	}
	return out
}

func HTTPStatus(err error) int {
	var typed *Error
	if errors.As(err, &typed) && typed.StatusCode >= 400 && typed.StatusCode <= 599 {
		return typed.StatusCode
	}
	switch CodeOf(err) {
	case CodeInvalidArgument, CodeRequestInvalid, CodePrimaryExpectedRequired:
		return 400
	case CodeNotFound, CodeIdentityNotFound:
		return 404
	case CodeCapabilityForbidden:
		return 403
	case CodeCustomerMismatch, CodeCustomerMembershipInactive, CodeIdentityConflict, CodeChannelDictionaryInvalid, CodeIdentityChannelMigrationRequired, CodePrimaryChanged, CodePrimaryNotActive, CodePrimaryOperationRequired, CodePrimaryChangeRequired:
		return 409
	case CodePrimaryOperationFailed:
		return 500
	case CodeDelegateUnavailable:
		return 502
	default:
		return 503
	}
}
func CodeOf(err error) ErrorCode {
	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed.Code
	}
	return ""
}
