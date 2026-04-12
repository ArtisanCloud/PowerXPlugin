package contracts

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode 统一联邦登录错误码，供前后端共享语义。
type ErrorCode string

const (
	ErrorCodeInvalidProvider     ErrorCode = "FEDERATED_INVALID_PROVIDER"
	ErrorCodeProviderRegistered  ErrorCode = "FEDERATED_PROVIDER_ALREADY_REGISTERED"
	ErrorCodeProviderNotFound    ErrorCode = "FEDERATED_PROVIDER_NOT_FOUND"
	ErrorCodeUnauthorized        ErrorCode = "FEDERATED_UNAUTHORIZED"
	ErrorCodeInvalidChallenge    ErrorCode = "FEDERATED_INVALID_CHALLENGE"
	ErrorCodeChallengeExpired    ErrorCode = "FEDERATED_CHALLENGE_EXPIRED"
	ErrorCodeChallengeReplay     ErrorCode = "FEDERATED_CHALLENGE_REPLAY"
	ErrorCodeChallengeTenantMiss ErrorCode = "FEDERATED_CHALLENGE_TENANT_MISMATCH"
	ErrorCodeRiskSignature       ErrorCode = "FEDERATED_RISK_SIGNATURE_INVALID"
	ErrorCodeRiskStateNonce      ErrorCode = "FEDERATED_RISK_STATE_NONCE_MISMATCH"
	ErrorCodeRiskReplay          ErrorCode = "FEDERATED_RISK_REPLAY"
	ErrorCodeRiskTenantBoundary  ErrorCode = "FEDERATED_RISK_TENANT_BOUNDARY"
)

// FederatedError 包装统一错误码与底层错误。
type FederatedError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *FederatedError) Error() string {
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "federated error"
	}
	if e.Code == "" {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %v", msg, e.Cause)
		}
		return msg
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s (%s): %v", msg, e.Code, e.Cause)
	}
	return fmt.Sprintf("%s (%s)", msg, e.Code)
}

func (e *FederatedError) Unwrap() error {
	return e.Cause
}

// NewError 构造标准错误。
func NewError(code ErrorCode, message string) error {
	return &FederatedError{Code: code, Message: message}
}

// WrapError 用于添加底层错误上下文。
func WrapError(code ErrorCode, message string, cause error) error {
	return &FederatedError{Code: code, Message: message, Cause: cause}
}

// HasCode 判断错误链是否包含指定错误码。
func HasCode(err error, code ErrorCode) bool {
	var ferr *FederatedError
	if errors.As(err, &ferr) {
		return ferr.Code == code
	}
	return false
}
