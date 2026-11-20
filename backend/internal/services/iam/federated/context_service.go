package federated

import (
	"errors"
)

const (
	FederatedErrAuthUnavailable = "FEDERATED_AUTH_UNAVAILABLE"
	FederatedErrUpstream        = "FEDERATED_UPSTREAM_UNAVAILABLE"
)

// ContextService 统一 standalone/delegated 的上下文与错误语义。
type ContextService struct{}

type ContextError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewContextService() *ContextService { return &ContextService{} }

// NormalizeUnavailableError 保持两种模式对“不可用”语义一致（通用文案 + 同一 code）。
func (s *ContextService) NormalizeUnavailableError(mode string, err error) ContextError {
	_ = mode
	if err == nil {
		err = errors.New("unavailable")
	}
	return ContextError{
		Code:    FederatedErrAuthUnavailable,
		Message: "登录失败，请稍后重试",
	}
}

// NormalizeContext 输出统一结构，并在 delegated 标记宿主权威。
func (s *ContextService) NormalizeContext(mode string, ctx IdentityContext) IdentityContext {
	if mode == "delegated" {
		ctx.PolicySource = "host-authoritative"
		return ctx
	}
	if ctx.PolicySource == "" {
		ctx.PolicySource = "plugin-local"
	}
	return ctx
}
