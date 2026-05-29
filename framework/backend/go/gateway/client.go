package gateway

import (
	internal "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/integration/gateway"
)

// Config 是 Gateway Client 的公开配置。
type Config = internal.Config

// TokenProvider 返回短期 Bearer token。
type TokenProvider = internal.TokenProvider

// InvokeRequest 描述一次能力调用请求。
type InvokeRequest = internal.InvokeRequest

// Response 封装 Gateway 返回值。
type Response = internal.Response

// GatewayError 映射 Gateway 的错误条目。
type GatewayError = internal.GatewayError

// ContractStatus 描述契约摘要与期望版本的状态。
type ContractStatus = internal.ContractStatus

// InvocationError 为标准化错误类型。
type InvocationError = internal.InvocationError

// Client 暴露 Gateway 调用能力。
type Client = internal.Client

// NewClient 构造官方 Gateway Client。
func NewClient(cfg Config) (*Client, error) {
	return internal.NewClient(cfg)
}
