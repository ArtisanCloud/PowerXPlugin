package contactfw

import "errors"

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
)

type Error struct {
	Code  ErrorCode
	Cause error
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
func NewError(code ErrorCode, cause error) *Error { return &Error{Code: code, Cause: cause} }
func CodeOf(err error) ErrorCode {
	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed.Code
	}
	return ""
}
