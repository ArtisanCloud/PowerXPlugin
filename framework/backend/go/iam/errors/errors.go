package iamerrors

import "errors"

const (
	CodeBaselineInvalidArgument   = "IAM_BASELINE_INVALID_ARGUMENT"
	CodeBaselineResponseInvalid   = "IAM_BASELINE_RESPONSE_INVALID"
	CodeBaselineUnavailable       = "IAM_BASELINE_DEPENDENCY_UNAVAILABLE"
	CodeIdentityMigrationRequired = "IAM_IDENTITY_MIGRATION_REQUIRED"
	CodeIdentityResponseInvalid   = "IAM_CONTEXT_RESPONSE_INVALID"
	CodeContextInvalidArgument    = "IAM_CONTEXT_INVALID_ARGUMENT"
	CodeProofRequired             = "IAM_CONTEXT_PROOF_REQUIRED"
	CodeProofInvalid              = "IAM_CONTEXT_PROOF_INVALID"
	CodeProofExpired              = "IAM_CONTEXT_PROOF_EXPIRED"
	CodeUserSessionInvalid        = "IAM_CONTEXT_USER_SESSION_INVALID"
	CodeUserSessionExpired        = "IAM_CONTEXT_USER_SESSION_EXPIRED"
	CodeServiceCredentialInvalid  = "IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID"
	CodeServiceCredentialExpired  = "IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED"
	CodeContextUnavailable        = "IAM_CONTEXT_UNAVAILABLE"
	CodeContextRequestForbidden   = "IAM_CONTEXT_REQUEST_FORBIDDEN"
	CodeModeInvalid               = "PROVIDER_MODE_INVALID"
	CodeInvalidArgument           = "IAM_INVALID_ARGUMENT"
	CodeModeConflict              = "PROVIDER_MODE_CONFLICT"
	CodeAdapterNotBound           = "IAM_ADAPTER_NOT_BOUND"
	CodeAdapterAlreadyBind        = "IAM_ADAPTER_ALREADY_BOUND"
	CodeUnauthorized              = "IAM_UNAUTHORIZED"
	CodeForbidden                 = "IAM_FORBIDDEN"
	CodeUpstreamDependency        = "IAM_UPSTREAM_DEPENDENCY"
	CodeMemberNotFound            = "IAM_MEMBER_NOT_FOUND"
)

// Error 表示 IAM 域内的统一错误对象。
type Error struct {
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Code
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New 生成标准 IAM 错误。
func New(code, message string) error {
	return &Error{Code: code, Message: message}
}

// Wrap 在统一错误码下包装底层错误。
func Wrap(code, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// CodeOf 返回错误码；非 IAM 错误返回空串。
func CodeOf(err error) string {
	if err == nil {
		return ""
	}
	var coded interface{ IAMCode() string }
	if errors.As(err, &coded) {
		return coded.IAMCode()
	}
	var target *Error
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

// IsCode 判断错误是否匹配给定错误码。
func IsCode(err error, code string) bool {
	return CodeOf(err) == code
}

// StatusCode 提供统一 HTTP 状态映射。
func StatusCode(err error) int {
	var status interface{ HTTPStatus() int }
	if errors.As(err, &status) {
		return status.HTTPStatus()
	}
	switch CodeOf(err) {
	case CodeBaselineInvalidArgument, "IAM_BASELINE_REQUIREMENTS_INVALID":
		return 400
	case CodeBaselineResponseInvalid:
		return 502
	case CodeBaselineUnavailable:
		return 503
	case "IAM_CONTEXT_WS_HANDLE_REQUIRED", "IAM_CONTEXT_WS_HANDLE_INVALID", "IAM_CONTEXT_WS_HANDLE_EXPIRED":
		return 401
	case "IAM_WS_IDENTITY_MISMATCH":
		return 403
	case "IAM_WS_CREDENTIAL_EXPIRED", "IAM_WS_IDENTITY_REVOKED":
		return 401
	case "IAM_WS_TOPIC_FORBIDDEN":
		return 403
	case CodeProofRequired, CodeProofInvalid, CodeProofExpired, CodeUserSessionInvalid, CodeUserSessionExpired, CodeServiceCredentialInvalid, CodeServiceCredentialExpired:
		return 401
	case CodeContextRequestForbidden, "IAM_CONTEXT_SERVICE_FORBIDDEN", "IAM_CONTEXT_USER_FORBIDDEN", "IAM_CONTEXT_MEMBER_REQUIRED", "IAM_CONTEXT_TENANT_FORBIDDEN", "IAM_CONTEXT_PLUGIN_FORBIDDEN":
		return 403
	case "IAM_CONTEXT_POLICY_CHANGED", CodeIdentityMigrationRequired:
		return 409
	case CodeContextInvalidArgument:
		return 400
	case CodeContextUnavailable:
		return 503
	case CodeIdentityResponseInvalid:
		return 502
	case CodeUnauthorized:
		return 401
	case CodeForbidden:
		return 403
	case CodeMemberNotFound:
		return 404
	case CodeUpstreamDependency, CodeAdapterNotBound:
		return 424
	case CodeModeConflict, CodeAdapterAlreadyBind:
		return 409
	case CodeModeInvalid, CodeInvalidArgument:
		return 400
	default:
		return 500
	}
}
