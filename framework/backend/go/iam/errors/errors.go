package iamerrors

import "errors"

const (
	CodeModeInvalid        = "PROVIDER_MODE_INVALID"
	CodeInvalidArgument    = "IAM_INVALID_ARGUMENT"
	CodeModeConflict       = "PROVIDER_MODE_CONFLICT"
	CodeAdapterNotBound    = "IAM_ADAPTER_NOT_BOUND"
	CodeAdapterAlreadyBind = "IAM_ADAPTER_ALREADY_BOUND"
	CodeUnauthorized       = "IAM_UNAUTHORIZED"
	CodeForbidden          = "IAM_FORBIDDEN"
	CodeUpstreamDependency = "IAM_UPSTREAM_DEPENDENCY"
	CodeMemberNotFound     = "IAM_MEMBER_NOT_FOUND"
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
	switch CodeOf(err) {
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
