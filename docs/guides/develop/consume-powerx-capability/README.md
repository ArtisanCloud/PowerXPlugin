# PowerX 能力消费操作手册

> 适用范围：PowerXPlugin 仓库（插件后端 + web-admin + Skeleton），覆盖宿主 Delegated 模式与 Skeleton Standalone 模式。  
> 关联文档：[`docs/plan/009-consume-powerx-capability.md`](../../plan/009-consume-powerx-capability.md)、PowerX 底座指南 [`PowerX/docs/guides/develop/open_capability`](../../../../PowerX/Core/PowerX/docs/guides/develop/open_capability)。

## 1. 统一调用视图

``` 
Plugin Web Admin ──(HTTPS)──> 插件后端 API (/api/v1/integration/capabilities/invoke)
                               │
                               └── Gateway Client ──> PowerX Integration Gateway (/tenant/invocations / gRPC)
```

| 角色 | 默认端口（本地开发） | 说明 |
| --- | --- | --- |
| Web Admin (Vite/Nuxt Dev) | `3131` | 运行 `npm --prefix skeleton/web-admin/nuxt run dev` 后的本地调试端口 |
| 插件后端 Skeleton | `8078` | `go run ./skeleton/backend/go-gin/cmd/plugin` 启动后监听，前端所有 API（含 Capability Lab）都只打到这里 |
| PowerX Integration Gateway | `8077` | PowerX 底座 Dev API 端口，插件后端通过 `PX_GATEWAY_BASE_URL` 调用 `/tenant/invocations`；若宿主端口不同请自行覆盖 |

## 1.1 宿主与插件 Trace 日志统一契约（已落地）

为实现 PowerX 宿主与插件日志在同一日志源聚合检索，统一采用以下字段契约：

- 必填字段：`trace_id`、`request_id`、`plugin_id`、`tenant_uuid`、`path`、`status`、`latency`
- 头部约定：`X-Trace-Id`、`X-Request-ID`（`Request-ID` 作为兼容输入）
- 字段命名：统一使用下划线风格（不要混用 `traceId` / `requestId`）

透传规则（插件侧）：

1. 优先使用宿主透传的 `X-Trace-Id` / `X-Request-ID`
2. 若 `X-Trace-Id` 缺失，则回退为 `request_id`
3. 若 `X-Request-ID` 也缺失，插件生成 `request_id`，并同步设置 `trace_id`
4. 所有 HTTP access 日志必须带 `trace_id + request_id + plugin_id`

代码落点（Skeleton）：

- `skeleton/backend/go-gin/internal/middleware/common.go`
  - `RequestID()`：实现 trace/request 透传优先与回填
  - `RequestLogger()`：统一输出 `trace_id/request_id/plugin_id/tenant_uuid`
- `skeleton/backend/go-gin/internal/transport/http/middleware/request_trace.go`
  - 调试日志字段改为 `trace_id/request_id/plugin_id`

宿主侧职责（PowerX）：

- 在入口与 `/_p` 代理链路生成/透传 `trace_id`、`request_id`
- 代理日志输出统一字段（含 `plugin_id/tenant_uuid/path/status/latency/trace_id`）
- 采集层（Fluent Bit/Vector/OTel Collector）按 `trace_id + plugin_id + tenant_uuid` 聚合检索

排障顺序（推荐）：

1. 先看宿主日志是否有同一 `trace_id` 的入口和 `/_p` 转发记录
2. 再看插件 `HTTP request completed` 是否带同一 `trace_id`
3. 若 trace 断裂，优先检查代理是否透传 `X-Trace-Id/X-Request-ID`

### /tenant/invocations 调用语义

- `/tenant/invocations` 是“能力调度器”而非 HTTP 代理，**`action` 只是能力语义标签**；要让 Gateway 正确路由，必须在 `payload` 中给出完整的协议描述。
- 对 REST 通道，payload 至少包含 `method`、`endpoint`、可选的 `query`/`headers`/`body`。未传 `method` 或 `endpoint` 会被插件后端直接阻止（Skeleton/宿主均如此）。
- gRPC 或其他协议同理，需要声明 `preferred_protocol`（`grpc`、`workflow`、`agent` 等）以及 Gateway 文档要求的字段。

```jsonc
{
  "capabilityId": "com.corex.media.assets.manage",
  "action": "Create",
  "preferredProtocol": "rest",
  "payload": {
    "method": "POST",
    "endpoint": "/api/v1/media/assets",
    "headers": {
      "Content-Type": "application/json"
    },
    "query": {
      "page": 1,
      "page_size": 20
    },
    "body": {
      "assetName": "demo.pdf",
      "uploadMethod": "presign_upload"
    }
  }
}
```

- Gateway 返回体由两部分组成：顶层的 `traceId/status/errors`，以及 **真实业务响应的 JSON**（Skeleton 与宿主都会把它透传到前端）。因此 UI/日志需要把整个 body 记录下来，而非仅查看 trace 元数据。
- 如果 Gateway 找不到匹配 adapter，会返回 404/502 并附带错误数组；这通常是因为 `method`/`endpoint`/`preferred_protocol` 与能力注册不一致。

- **前端** 永远只访问插件后端（宿主反代 `/_p/<plugin-id>/api/v1`，本地 Skeleton 为 `http://127.0.0.1:8078/api/v1`）。
- **后端** 读取统一环境变量：
  - `PX_GATEWAY_BASE_URL`
  - 宿主 delegated：`POWERX_STS_CLIENT_ID` / `POWERX_STS_CLIENT_SECRET` / `POWERX_GRPC_UPSTREAM_ADDRESS` / `POWERX_GRPC_UPSTREAM_TENANT_UUID`
  - Standalone：`PX_GATEWAY_AUTH_SCHEME=apikey` / `PX_GATEWAY_API_KEY`
  - 租户由显式上下文或 STS 凭证解析，不再从静态 bearer token 推导
  - `PX_GATEWAY_CONTRACT_VERSION`（可选，配合 `dist/capability-contracts.json` 校验契约版本）
- **Gateway Client** 负责注入 `Authorization`、`X-Request-ID`，并输出 TraceId、限流事件等观测数据；`tenant_uuid` 并非所有模式都强制透传（proxy 场景通常由底座按凭证解析）。

> **环境加载与模式说明**：
>
> - Go Gin / FastAPI 后端会自动读取 `skeleton/backend/.env`（示例见 `skeleton/backend/.env.example`），并覆盖 `config.yaml` 中同名配置。
> - 宿主模式要求 `POWERX_PROXY=1`，并提供 `PX_GATEWAY_BASE_URL + POWERX_STS_CLIENT_ID + POWERX_STS_CLIENT_SECRET + POWERX_GRPC_UPSTREAM_ADDRESS + POWERX_GRPC_UPSTREAM_TENANT_UUID`；否则会返回 503。
> - 若 GoLand Run Config 中仍有旧环境变量（如 `POWERX_PROXY=0`/`IAM_MODE=local`），会覆盖 `.env` 的值，请先清理。

### Gateway API 前缀规范（含 WS-Bus）

- 统一新增 `PX_GATEWAY_API_PREFIX`，默认值为 `/api/v1`（对应 `config.yaml` 的 `gateway.api_prefix`）。
- 绝大多数 Gateway 能力调用（如 `/tenant/invocations`）跟随该前缀拼接请求地址。
- **WS-Bus 在不同环境可能不一致**：有的网关走 `/api/v1/...`，有的只走 `/api/...`。  
  因此不要写死路径，统一通过 `PX_GATEWAY_API_PREFIX` 控制：
  - 若你的网关是 `/api/v1`：`PX_GATEWAY_API_PREFIX=/api/v1`
  - 若你的网关是 `/api`：`PX_GATEWAY_API_PREFIX=/api`

### 鉴权规范（framework 统一策略）

为避免各插件实现不一致，统一由 framework 执行模式分流与凭证策略：

| 运行形态 | 是否手动指定 auth scheme | 推荐凭证 | 说明 |
| --- | --- | --- | --- |
| Delegated（宿主/代理） | 否（由 runtime 决策） | STS access token（Bearer, `aud=powerx:api`） | 必须走 STS；缺失 STS 契约变量即 fail-fast。 |
| Standalone Local | 否（由 runtime 决策） | `PX_GATEWAY_API_KEY`（ApiKey） | 本地联调用 ApiKey，不使用宿主 STS 凭证。 |

实现判定规则（framework 目标）：

1. 先根据 runtime 模式判定 `delegated` 或 `standalone local`；
2. `delegated` 强制 STS Exchange 后发送 `Authorization: Bearer <sts_access_token>`；
3. `standalone local` 使用 `Authorization: ApiKey <api_key>`；
4. 模式与凭证不匹配时，启动期直接报错并给出诊断字段（mode/required_credential/base_url_configured）。

建议：

- 插件业务 Handler 不要再手写 `bearer/apikey` 分支；
- 统一复用 framework Host Capability Client + `RequireCapabilityGateway` Guard；
- 保持“一种模式一套凭证”，避免混用造成线上不可预测行为。

## 2. 前提条件

| 项目 | 说明 |
| --- | --- |
| Manifest | `skeleton/plugin.yaml` 中声明 `capabilities.required`（需要调用的 `source=corex` 能力）与 `capabilities.provides`（插件自身提供的能力）。提交前运行 `node scripts/capabilities/validate-capabilities.mjs --manifest ./skeleton/plugin.yaml`。 |
| 底座能力参考 | 查阅 PowerX 底座文档 `PowerX/docs/guides/develop/open_capability`，了解 Media/Event/Workflow/Knowledge 模块的 `capability_id`、REST/gRPC 协议、示例命令。 |
| 凭证 | 宿主/Delegated：注入 `PX_GATEWAY_BASE_URL + POWERX_STS_CLIENT_ID + POWERX_STS_CLIENT_SECRET + POWERX_GRPC_UPSTREAM_ADDRESS + POWERX_GRPC_UPSTREAM_TENANT_UUID`。Skeleton Local：使用 `PX_GATEWAY_BASE_URL + PX_GATEWAY_AUTH_SCHEME=apikey + PX_GATEWAY_API_KEY`。 |
| CLI & 工具 | `scripts/capabilities/run-from-package.mjs`（手动触发能力调用）、`px-plugin capabilities quota --manifest ...`（为租户配置配额/限流样例）。 |

## 3. 宿主（Delegated）模式

1. **能力申领**：在 Admin 控制台或 `px-plugin capabilities apply --manifest ./skeleton/plugin.yaml`（CLI 提供后）申领目标 `capability_id`。
2. **配置注入**：
   - `framework/backend/go/bootstrap/app.go` 会自动读取宿主注入的 `PX_GATEWAY_*` 环境变量。
   - Nuxt Admin 通过 `runtimeConfig.public.powerx` 获取插件后端 API 前缀，无需任何 PowerX 凭证。
3. **后端封装**：
   - `framework/backend/go/internal/services/capability_invoker` 作为唯一入口，提供 `Invoke(ctx, capabilityId, action, payload)`。
   - `framework/backend/go/router/router.go` 暴露 `/api/v1/integration/capabilities/invoke`，接入前端/CLI。
   - HTTP handler 统一调用 `transport/http/middleware.RequireCapabilityGateway`，确保网关不可用时输出一致的 503 结构。
4. **前端调用**：
   - 使用 `framework/frontend/nuxt/framework-admin/layer/app/plugins/powerx-capability.client.ts` 或 `usePowerXCapability` composable，调用插件后端 API。
   - 支持在 UI 中展示 Gateway TraceId、Mock 提示、警告（如契约版本过期 -> 服务端会设置 `X-PowerX-Contract-Status` header）。
5. **观测**：
   - `framework/backend/go/observability` 记录 `capabilityId`、`tenantUUID`、`traceId`。
   - `docs/operations/observability.md` 记录指标与排障步骤。

## 4. Skeleton（Standalone）模式

1. **环境准备**：
   - 根据 `.env.local` 模板手动写入 `PX_GATEWAY_BASE_URL/PX_GATEWAY_AUTH_SCHEME=apikey/PX_GATEWAY_API_KEY`。
   - `skeleton/backend/.env.example` 默认把 `PX_GATEWAY_BASE_URL` 指向 `http://127.0.0.1:8077`。若暂时没有 ApiKey，可把 `PX_USE_MOCK=media` 等写入以验证前后端链路。
   - 可选：`PX_USE_MOCK=<module>` 用于 Dev Gateway 不可达时的 Mock。
2. **后端配置**：
   - `skeleton/backend/go-gin/internal/integrations/gateway` 包装框架 Gateway Client，支持 Mock/离线提示。
   - `skeleton/backend/go-gin/cmd/server/main.go` 在启动时检测 Token 过期、输出提示。
3. **前端**：
   - `skeleton/web-admin/nuxt` 复用和宿主一致的 `powerx-capability` 插件，只不过 `runtimeConfig.public.powerx.apiBase` 默认为本地 `http://127.0.0.1:8078/api/v1`。
4. **调试**：
   - `npm --prefix skeleton/web-admin/nuxt run dev -- --use-mock=media` 可在前端快速切 Mock。
   - `scripts/capabilities/run-from-package.mjs --mode skeleton --manifest ./skeleton/plugin.yaml --cap <capabilityId>` 自动读取 `.env.local` 并打印请求/响应日志。（当前版本 `--use-mock` 选项存在 “mock is not defined” Bug，需后续修复。）
5. **观测**：
   - Skeleton 后端同样会记录 `traceId`/限流事件，便于复现宿主场景。

## 5. 能力发现 & Registry 对齐

1. 在 PowerX 底座 Web Admin 中浏览“开放能力”页面，或使用 `curl` 调用 `/admin/platform-capabilities`。
2. 通过 `PowerX/docs/guides/develop/open_capability/<module>.md` 查阅具体能力（`capability_id`、REST/gRPC 接口、示例）。
3. 将需要的能力写入 `plugin.yaml`：
   ```yaml
   capabilities:
     required:
       - com.corex.media.assets.manage
       - com.corex.eventfabric.publish
   ```
4. 若需更新配额，使用：
   ```bash
   px-plugin capabilities quota \
     --manifest ./skeleton/plugin.yaml \
     --capability-id com.powerx.plugins.base.template.create \
     --tenant sandbox-tenant \
     --base-url "$PX_DEV_API_BASEURL" \
     --token "$PX_DEV_API_TOKEN" \
     --qps 50 --burst 100 --limits 5000
   ```
   CLI 会生成 Postman/HTTP 示例写入 `dist/capabilities/<id>/samples/`。

## 6. Skeleton 能力调试页（Capability Lab）

Skeleton web-admin 已内置 `/powerx/capability-lab` 页面（侧边导航“能力”分组下的“开放能力调试”入口），仅 `IsRoot`/系统管理员可打开。该页面与正式业务共用同一后端 `/api/v1/integration/capabilities/invoke`，方便开发者在本地验证 Gateway 链路、Mock 及契约告警。能力下拉会请求插件后端 `/api/v1/admin/capabilities?source=corex`，后端再通过 Gateway 调用 PowerX `/tenant/capabilities` 获取最新的 `source=corex` 底座能力，因此模块/Action/协议信息与底座文档保持实时一致；若未携带 `source` 参数，则继续返回本地 manifest 数据以兼容其他管理页面。页面直接依据实时的 PowerX Capability Catalog 生成「模块 → 能力 → Action」三级联动，并在选择 REST Action 时**自动填充包含 `preferredProtocol/method/endpoint/query/body` 的模板**，提示你补齐 `/tenant/invocations` 所需字段；gRPC/Workflow/其他协议同样会根据 catalog 返回值自动补齐 service/method/字段，无需再借助本地 mock。

**使用步骤**

1. 启动 `skeleton/backend/go-gin` 与 `skeleton/web-admin/nuxt`，使用 Root 账户登录后点击“开放能力调试”。
2. 在“调用配置”卡片填写 `capabilityId`、`action` 和 JSON `payload`。选择 REST Action 后，Payload 区域会展示模板（含 `method`、`endpoint`、`headers`、`query`、`body`），并允许你一键插入；gRPC/Other Action 会提示协议类型与必填字段。可选参数包括：
   - **Mock 模块**：在输入框填写模块名（如 `media`），页面会透传 `X-PX-Use-Mock: <module>` 到后端/Gateway；响应 `warnings` 中会标记“通过 X-PX-Use-Mock 请求 Mock 模块”，方便确认 Mock 是否生效。
   - **Request ID / API Base**：用于调试自定义 `X-Request-ID` 或手动切换代理地址。
3. “请求预览”实时展示最终 URL、Headers、Body，可一键复制到 curl / `.http` 文件。
4. 调用完成后，“调用结果”显示状态、TraceId、耗时与 JSON 响应；若契约版本过期、Mock 被启用或 Gateway 返回提示，都会出现在 `warnings` 中。“最近记录”会缓存最近 5 条请求，便于回放。
5. Gateway 不可达时，可在页面中指定 Mock 模块，或结合 `.env.local` 中的 `PX_USE_MOCK`，以验证前后端封装是否正常。

> 页面不会直接暴露任何 `PX_*` 凭证，所有请求均通过插件后端代理；调试头主要包含 `X-PX-Use-Mock` / `X-Request-ID`，`tenant_uuid` 不作为该页面固定输入项。

## 7. 常见问题 & 排障

| 问题 | 原因/排查 |
| --- | --- |
| `gateway: base URL is required` | 未注入 `PX_GATEWAY_BASE_URL`；检查宿主部署或 Skeleton `.env.local`。 |
| `Authorization` 相关 401 | 凭证缺失/过期，或运行模式与凭证策略不匹配；delegated 检查 STS client 配置，local 检查 `PX_GATEWAY_API_KEY`。 |
| `mock is not defined`（`run-from-package`） | 当前 CLI Bug，临时改用 `node scripts/capabilities/validate-capabilities.mjs` + Go 测试。 |
| 契约版本警告 | `dist/capability-contracts.json` 与 `PX_GATEWAY_CONTRACT_VERSION` 不一致。运行 `npm --prefix scripts/capabilities run digest` 更新摘要并检查 `docs/plan/009...` 的契约升级流程。 |

## 8. Agent Client 边界

Gateway Client 用于插件消费 PowerX 平台能力；Agent Client 用于 PowerX Agent Runtime 会话和事件流。智能任务调试应使用 `runtime/powerx/agent` 的 invoke、SSE 或 WS typed event 解码，不要在业务层重复实现 Agent stream 协议。delegated 模式下 Agent Client 只接受 bearer/STS，不接受 `PX_TOOL_TOKEN` 或 `PX_GATEWAY_API_KEY` 作为 delegated 凭证。

---

通过上述手册，开发者可在宿主与 Skeleton 两种模式下快速配置凭证、调用 PowerX 能力、调试 Mock/实链路，并结合 PowerX 底座文档验证接口定义。后续若 CLI 增加 `capabilities plan/login` 等命令，可在本文对应章节补充示例。
