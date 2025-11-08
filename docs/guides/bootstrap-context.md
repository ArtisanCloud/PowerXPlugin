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
- `px-plugin init` / `px-plugin doctor` / `px-plugin import` 等 CLI 命令会在生成项目后立即调用 `bootstrap.Context` 约定的默认实现，请在调试或扩展模板时同步验证 CLI 输出。

## CLI 引导与合规自检

为了让 `bootstrap.Context` 的语义贯穿整个开发流程，CLI 工具在初始化与导入第三方源码时会执行以下步骤：

1. `px-plugin init --template <id>`：读取 `packages/template-registry/index.yaml`，拉取对应模板并写入 `publish.yml` / `reports/sbom.json`。生成后的 Handler 与中间件均依赖 `bootstrap.Context`，方便在不同运行时复用。
2. `px-plugin doctor --fix`：在插件目录生成 `.doctor/report.json`，检测 Node/Go 版本、Feature Flag、`backend/go.mod` 与 `web-admin/node_modules` 状态；必要时自动运行 `go mod tidy` / `npm install`，确保脚手架输出与上下文约定一致。
3. `px-plugin import --source <path>`：根据 `config/compliance/external_source_policy.yaml` 校验第三方源码来源、许可证、包体大小、校验和等信息，并生成 `./.compliance/import-report.json`，供合规审核追踪 `plugin-import-audit` Webhook。

开发者在修改 `bootstrap.Context` 或模板实现后，务必重新运行上述命令并记录报告，以确保新的接口约定与 CLI/文档保持一致。

## 宿主模拟器与沙箱验证

Phase 11 引入的宿主模拟器与沙箱验证同样通过 `bootstrap.Context` 暴露 API：

1. `px-plugin host start --mock` → `POST /internal/dev/hosts/sessions`  
   - Handler：`framework/backend/go/runtime/devapi/handlers/host_simulator.go`  
   - 返回 `sessionId`、`endpoint`，并可通过 `GET /logs`、`POST /attach` 注入断点/变量。
2. `px-plugin dev --watch` / `SessionClient.attachBreakpoints`  
   - CLI 通过 `tools/cli/src/runtime/hotreload/session.ts` 的新方法向宿主推送断点并统计 `dev.hotload.*` 指标。
3. `px-plugin sandbox deploy` → `POST /internal/dev/sandbox/deploy`  
   - Handler：`framework/backend/go/runtime/devapi/handlers/sandbox_validation.go`，负责 orchestration、脱敏数据加载与 `validationId` 输出。
4. `px-plugin debug report` → `POST /internal/dev/debug/report`  
   - Handler：`framework/backend/go/runtime/devapi/handlers/debug_report.go`，调用 `telemetry.NewRecorder` 将诊断结果写入监控/工单。

无论是 CLI 还是宿主 API，最终都依赖 `bootstrap.Context` 统一访问参数、响应和中间件，因此在扩展至 Rust/Java 等语言时仅需保持该接口一致。
