# 框架上下文（`bootstrap.Context`）约定

`bootstrap.Context` 是框架在插件与底层 HTTP 实现之间的抽象层，用于把业务代码与具体 Web 框架解耦。本文档说明该接口的设计目标、方法约定以及跨语言适配指南，便于在不同技术栈上实现一致的行为。

## 设计目标

- **框架无关**：业务 Handler 与中间件只依赖接口，不直接接触 Gin 或其他 Web 框架的 API。
- **可测试**：方便在单测中通过 Stub/Mock 实现上下文，而无需真实的 HTTP Server。
- **可扩展**：允许在不改动业务代码的前提下替换底层 HTTP 适配器，甚至迁移到其他语言。

## 接口能力

当前 Go 版本的接口定义位于 `framework/backend/go/bootstrap/router_port.go`。各语言应遵循相同语义：

| 方法 | 说明 |
| --- | --- |
| `Param(name string) string` | 返回路径参数，例如 `/api/:id` 中的 `id`。|
| `Query(name string) string` | 读取查询参数（URL Query）。|
| `BindJSON(v any) error` | 将请求体反序列化为 JSON 到 `v`。|
| `JSON(code int, v any)` | 设置状态码并写入 JSON 响应。|
| `Status(code int)` | 写入状态码。|
| `Header(name string) string` | 读取请求或响应头。|
| `SetHeader(name, value string)` | 写入或删除响应头。|
| `Method() string` | 返回 HTTP 方法。|
| `Context() context.Context` | 返回请求上下文（用于超时、取消等）。|
| `SetContext(ctx context.Context)` | 替换请求上下文。|
| `HTTPResponseWriter() http.ResponseWriter` | （框架内部使用）返回底层响应写入器。|
| `HTTPRequest() *http.Request` | （框架内部使用）返回底层请求实例。|

> **说明**：`HTTPResponseWriter` / `HTTPRequest` 是为了让适配层安全地访问原生 HTTP 对象，业务侧通常不直接调用。

## 适配器实现思路

1. **定义上下文实现**：在对应语言/框架中创建一个结构体或类，实现上述接口。内部应持有底层框架的 Request/Response 对象，并提供相应的转换逻辑。
2. **实现 Router 适配器**：满足 `bootstrap.Router` 规范（Handle / Group / Use 等），在收到请求时构造上下文实例并调用业务 Handler。
3. **复用框架中间件**：middleware 只依赖 `bootstrap.Context`，因此可以直接复用或按需改写。
4. **保留额外接口**：若需要访问底层请求对象（例如交给 Gin 处理），通过 `HTTPResponseWriter` / `HTTPRequest` 提供有界面的扩展。

## 跨语言迁移指南

虽然当前框架实现是 Go，但同样的接口契约可以在其他语言中复制：

- **PHP / Laravel**：定义一个 `BootstrapContextInterface`，使用 Symfony Request/Response 适配方法。
- **Python / FastAPI**：定义一个协议类或抽象基类，封装 `Request`、`Response`。
- **Rust / Actix**：定义 trait，使用泛型包装 HTTP 请求与响应。

关键要点是保持接口方法及其语义一致，以便框架文档、示例、测试、以及 CLI 模板可以跨语言共享。

## 后续维护建议

- 当接口新增方法时，需同步更新文档、生成器模板以及所有语言的适配实现。
- 模板与示例（如 `tools/cli/internal/templates/.../server/adapter.go.tmpl`）应引用本约定，确保脚手架生成的代码符合规范。
