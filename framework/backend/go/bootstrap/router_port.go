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
