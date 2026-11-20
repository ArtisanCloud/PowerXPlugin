# Framework 插件权限查询对齐

本次交付范围是 Go `iam.AuthzService.Authorize` 的 Core 成员插件权限查询、通用服务错误解码以及受控 Gateway SSE 客户端。权威合同为同目录 `plugin-service-authorization.md` 和 OpenAPI，来源 commit、未提交状态与摘要记录在 `plugin-authorization-source.json`。该 Core 合同同时描述服务调用票据；本次不以合同文件同步代表 Framework 已完成目标插件的票据验证客户端。

## 调用参数与结果

插件权限查询显式传入 `TenantUUID`、`MemberUUID`、`UserUUID`、`PluginID`、`PermissionCode`。租户 UUID 仅用于核对返回值，不发送给 Core。插件 ID 与正式业务权限码必须成对提供，不从调用方插件、resource/action 或路由别名推导目标权限。

```go
decision, err := authz.Authorize(ctx, contracts.AuthorizationRequest{
    TenantUUID: tenantUUID,
    MemberUUID: memberUUID,
    UserUUID: userUUID,
    PluginID: targetPluginID,
    PermissionCode: registeredPermissionCode,
})
```

仍使用 Core 正式支持的 `POST /tenant/iam/authorization:check`，以配置的 API 基址及启动时选定的 API Key 或 STS 凭据调用。resource/action 可省略；显式填写时必须与正式登记值及返回值一致。Core 明确保留的 `iam.member/read` 查询继续可用。其他旧插件 resource/action 查询不会自动拼接权限码或猜测插件。

结果保留 Core 的 `PluginID`、`PermissionCode`、成员／用户／租户 UUID、resource/action、reason 与 trace。客户端校验字段完整性和所有关联身份；空结果、主体错配、权限错配或不合法的允许／拒绝状态返回 `IAM_AUTHORIZATION_RESPONSE_INVALID`，不会当成普通拒绝。

`200 allowed:false / IAM_PERMISSION_DENIED` 是正常结果。404 权限未登记、404 成员不存在、400 主体不匹配、403 查询越权和 503 依赖故障保留各自状态及机器码。`delegated.AuthorizationRequestError` 提供 `Status`、`Code`、`ReasonCode`、`TraceID` 和 `RequestID`，不保存敏感正文；`iamerrors.CodeOf/StatusCode` 可直接读取判定与 HTTP 状态。HTTP-200 契约错误的 `Status` 保留 200，`HTTPStatus()` 为 502。

## 运行时与模板

Skeleton delegated IAM 直接使用启动时装配的 Framework 客户端，不再由旧代理用请求字段拼造授权结果。Host Contract Lab 的 `iam / authorization.check` 透传显式插件与权限字段以及响应和错误诊断。纯本地适配器不具备 Core 插件权限登记语义，收到这种查询时明确返回 `IAM_AUTHORIZATION_UNSUPPORTED`；本地既有 resource/action 检查仍由本地权限目录处理。

不自动换凭据、改目标插件、跟随重定向或重试授权请求。`PolicyToken` 不属于 Core 该查询合同，客户端拒绝携带它的请求。服务凭据只在后端使用。

共享样例 `fixtures/plugin-authorization.v1.json` 是 Framework 维护的合同样例，不标为 Core 权威 fixture。文档、Go SDK testdata 和 CLI 导出副本必须一致，新生成的 Go 插件必须包含合同与样例并可独立构建。

## 联调边界

- Go SDK、Skeleton、Host Contract 调试入口和 CLI 模板在本轮范围内；Nuxt 无需新增媒体或鉴权调用协议。此处不宣称 .NET 已对齐。
- SCRM 仍需显式填写插件与正式权限码；Worker 必须先获得真实关联的 user UUID，不能仅传 member UUID。当前 SCRM 本地 `replace` 指向此工作区，部署时仍需发布并固定匹配 SDK 版本。
- Core 已修复服务票据转发的业务错误和流式授权边界；Framework 消费 Core 返回的机器码与安全恢复字段，不解析错误字符串或补造恢复信息。
- 查询能力必须已发布，具备当前租户 registration 及精确实时 grant；本次改代码不会调整现有 Key 或 STS 授权。
- 本地源码及受控 HTTP 测试不代表安装态或 SCRM 真实消息发送通过。双方部署后需覆盖允许、撤权、未登记权限、主体错配和依赖失败，最后验收具体消息链路。

## 服务错误与版本诊断

`gateway.InvocationError` 与 `capability.HTTPError` 提供一致的 `StatusCode`、`ErrorCode`、`ReasonCode`、`TraceID`、`RequestID` 和类型化 `Details`。其中 `Details` 为 `hostcontract.InvocationFailureDetails`，保留 Core 白名单中的恢复字段，以及 `ExpectedProviderVersion`／`ActualProviderVersion`。恢复布尔值和整数使用指针，缺失值不会变成 `false` 或 `0`。

Core trace 从响应头、顶层 `trace_id` 或 `details.trace_id` 读取；插件 trace 和 request ID 单独保存在 `Details.UpstreamTraceID`／`UpstreamRequestID`。Gateway 原有 `UpstreamRequestID` 仍表示 Core 响应的 request ID，新代码优先使用 `RequestID` 明确表达这一含义。

`retryable`、`next_action_code`、重试间隔、操作 UUID 和版本信息只提供给业务调用方。客户端不自动重放业务请求，也不从上游 message、原始正文或自由文本推测机器码。`Body` 保留为后端诊断字段且不参与 JSON 序列化，`Error()` 只包含机器诊断；Host Contract 调试响应只暴露筛选后的字段。普通 Gateway 调用不再记录请求正文。

共享样例 `fixtures/plugin-service-errors-stream.v1.json` 由 Framework 维护，覆盖幂等冲突、可重试 503、运行版本不符、运行不可用和流式拒绝。Core 对 SCRM `error.code` 的归一化在 Core 完成；Framework 消费归一化后的正式错误 envelope。

## 受控流式入口

通用 Gateway `Client.InvokeStream`、Capability `Client.InvokeStream` 采用同一传输；`capability.Runtime.Streamer()` 只解析启动时选定的适配器。未实现流式的本地适配器明确返回不可用，不选择 Core 或其他协议兜底。

`StreamInput` 仅接受能力 ID、环境、业务 body 和 request／trace 标识。只发送 `preferred_protocol=rest`、`payload.body` 和可选 `payload.query.env`；env 仅允许 dev/staging/prod，地址、方法和身份头由 Core 正式绑定及后端凭据决定。API 前缀按配置拼接一次，客户端拒绝重定向和非 SSE 的成功响应。

`ai.Client.LLMStream` 及 Framework AI Runtime 已使用固定能力 `com.corex.ai.llm.stream` 经 `/tenant/invocations/stream` 发起调用；`LLMStreamInput.Environment` 对应可选环境。LLM 必须收到结束事件才能成功完成。已有会话的 `LLMSessionStream` 是独立的会话读取接口，不属于本次 POST 流式绑定。

流式支持 SSE event、id、多行 data、心跳和 Core trace；取消、超时或回调失败会关闭上游响应体，不重连、不轮询、不换凭据。Gateway／Capability 的 `StreamTimeout` 默认五分钟，独立于普通请求超时；AI 流式使用 AI 客户端的 Timeout。调用方可通过更短的 context deadline 终止流式。

Core 当前仅支持声明的 Core REST `.stream` 绑定。插件票据流式调用的 `CAPABILITY_SERVICE_STREAM_UNSUPPORTED` 保留为明确失败，不回退到旧代理或用户身份。服务票据、实时 grant、registration、模型授权和版本锁仍由 Core 验证；受控 SDK 测试不代表这些安装态条件已满足。
