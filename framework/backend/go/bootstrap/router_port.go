package bootstrap

import "context"

// Router 对外暴露最小路由能力，以避免绑定具体 HTTP 实现。
type Router interface {
	Group(rel string) Router
	Handle(method, path string, h Handler)
	Use(mw ...Middleware)
}

// Handler 抽象 HTTP 处理器。
type Handler func(Context)

// Middleware 为 Handler 提供装饰器能力。
type Middleware func(Handler) Handler

// Context 为 Handler 提供统一的请求访问能力。
type Context interface {
	Param(name string) string
	Query(name string) string
	BindJSON(v any) error
	JSON(code int, v any)
	Status(code int)
	Header(name string) string
	SetHeader(name, value string)
	Method() string
	Context() context.Context
	SetContext(ctx context.Context)
}

// CapabilityInvoker is the framework-level local capability execution hook.
// Plugins register one when they own capabilities that must execute in-process.
type CapabilityInvoker interface {
	CanInvokeCapability(capabilityID string) bool
	InvokeCapability(ctx context.Context, params CapabilityInvokeParams) (*CapabilityInvokeResult, error)
}

// CapabilityInvokeParams describes a framework capability invocation request.
type CapabilityInvokeParams struct {
	CapabilityID      string
	Action            string
	PreferredProtocol string
	Payload           map[string]any
	Metadata          map[string]any
	Headers           map[string]string
	RequestID         string
	TenantUUID        string
}

// CapabilityInvokeResult is the normalized local capability response.
type CapabilityInvokeResult struct {
	TraceID  string
	Status   string
	Data     any
	Raw      []byte
	Metadata map[string]any
}
