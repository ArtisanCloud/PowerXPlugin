# Standalone / Host 模式判定规则

## 1. 目标

统一插件在本地开发、宿主联调、宿主安装三类场景下的模式判定，避免把 provider 数据源选择、宿主代理链路、前端嵌入状态混成一个开关。

当前唯一有效口径：

- `POWERX_PROVIDER_MODE` 决定业务 provider 数据源：`local` 或 `delegated`
- `POWERX_PROXY` 决定是否接宿主代理/网关链路：`0` 或 `1`
- `NUXT_PUBLIC_INSIDE_POWERX` 只决定前端是否按宿主嵌入上下文运行

`POWERX_PROXY` 不得推导、覆盖或隐式改变 provider mode。

## 2. 后端真实规则

后端先解析 provider mode，再解析 proxy link：

1. provider mode 来源：
   - `config.context.provider_mode`
   - `POWERX_PROVIDER_MODE`
   - 未设置时默认 `local`
2. proxy link 来源：
   - `POWERX_PROXY=1` 表示启用宿主代理/网关链路
   - `POWERX_PROXY=0` 或未设置表示本地链路
3. 两者相互独立：
   - `POWERX_PROVIDER_MODE=delegated` 不等价于 `POWERX_PROXY=1`
   - `POWERX_PROXY=1` 不等价于 `POWERX_PROVIDER_MODE=delegated`
4. 缺少 provider 或 delegated provider 不可用时，业务 API 必须明确返回 503，不返回假空数据。

## 3. 2x2 模式矩阵

| 场景 | POWERX_PROVIDER_MODE | POWERX_PROXY | 数据源 | 链路 | 说明 |
|---|---|---:|---|---|---|
| standalone local | local | 0 | 插件本地 service / DB | 本地 | 默认本地开发 |
| local + proxy debug | local | 1 | 插件本地 service / DB | 宿主 gateway/proxy | 本地数据源联调宿主能力、WS、scheduler |
| standalone delegated | delegated | 0 | framework client / delegated provider | 本地进程直连已配置能力 | 用于本地模拟 delegated provider，不自动启用宿主代理 |
| host delegated | delegated | 1 | framework client / PowerX Core | 宿主 gateway/proxy | 标准宿主安装/委派模式 |

关键约束：

- 只有 `POWERX_PROVIDER_MODE=delegated + POWERX_PROXY=1` 才执行宿主 delegated 环境契约校验。
- `POWERX_PROVIDER_MODE=local + POWERX_PROXY=1` 是合法调试组合，不得被强行当成 delegated。
- `POWERX_PROVIDER_MODE=delegated + POWERX_PROXY=0` 可以启动，但 delegated client/provider 不可用时必须明确失败。

## 4. 前端变量职责

前端不参与后端 provider 决策，只消费后端页面 API 与 `/mode` diagnostics。

1. `runtimeConfig.public.providerMode`
   - 来源：`NUXT_PUBLIC_POWERX_PROVIDER_MODE` 或 `POWERX_PROVIDER_MODE`
   - 未设置时默认 `local`
   - 不从 `insidePowerX` 或 `POWERX_PROXY` 推导
2. `runtimeConfig.public.insidePowerX`
   - 只表示页面是否在宿主插件容器语义中运行
   - 影响 base URL、路由前缀、嵌入布局
3. `runtimeConfig.public.powerxProxy`
   - 只暴露宿主代理链路状态
   - 不表示 provider mode

推荐实践：

1. 宿主安装包显式注入：
   - `POWERX_PROVIDER_MODE=delegated`
   - `POWERX_PROXY=1`
   - `NUXT_PUBLIC_INSIDE_POWERX=1`
   - `NUXT_PUBLIC_POWERX_PROVIDER_MODE=delegated`
2. 本地 standalone 默认：
   - `POWERX_PROVIDER_MODE=local`
   - `POWERX_PROXY=0`
   - `NUXT_PUBLIC_INSIDE_POWERX=0`
3. 本地联调宿主链路：
   - `POWERX_PROVIDER_MODE=local`
   - `POWERX_PROXY=1`
   - 使用 ApiKey 或对应 gateway 调试凭证

## 5. 宿主链路鉴权规则

### host delegated

`POWERX_PROVIDER_MODE=delegated + POWERX_PROXY=1`：

1. 使用 STS access token（Bearer，`aud=powerx:api`）。
2. 必须提供：
   - `PX_GATEWAY_BASE_URL`
   - `POWERX_STS_CLIENT_ID`
   - `POWERX_STS_CLIENT_SECRET`
   - `POWERX_GRPC_UPSTREAM_ADDRESS`
   - `POWERX_GRPC_UPSTREAM_TENANT_UUID`
   - `PX_GATEWAY_AUTH_SCHEME=bearer`
3. 缺失时启动 fail-fast 或相关 API 返回明确 503。
4. 不默认透传入站 delegated/user bearer 到宿主 ws-bus 接口。

### local + proxy debug

`POWERX_PROVIDER_MODE=local + POWERX_PROXY=1`：

1. 业务页面数据仍来自插件本地 service / DB。
2. 调 PowerX 底座能力时使用 gateway debug 凭证。
3. 推荐：
   - `PX_GATEWAY_AUTH_SCHEME=apikey`
   - `PX_GATEWAY_API_KEY=<PowerX API Key>`
4. 出站 header：`Authorization: ApiKey <key>`。

## 6. 页面 API 策略

正式业务页面不感知运行模式，路径保持稳定：

- `/api/v1/admin/metadata/*`
- `/api/v1/admin/iam/*`
- `/api/v1/admin/customers/*`
- `/api/v1/admin/ai-settings/*`

后端按 provider mode 自动切换数据源：

- `local`：插件本地 service / DB
- `delegated`：framework client -> gateway/capability -> PowerX Core

每个模块的 `/mode` 或 diagnostics 至少返回：

- `mode`
- `provider`
- `delegated_available`
- `local_available`
- `read_only`

缺 provider 时必须明确返回 503。

## 7. WS / Capability Contract v2

1. 地址契约由宿主注入：
   - `PX_GATEWAY_BASE_URL`（HTTP）
   - `NUXT_PUBLIC_WS_ORIGIN`（WS origin）
   - `NUXT_PUBLIC_WS_PATH`（建议固定 `/api/ws`）
2. 插件只消费契约，不猜端口，不拼 `/_p/.../api/ws`。
3. 标准调试接口仅使用：
   - `POST /api/v1/admin/runtime/ws-bus/grant`
   - `POST /api/v1/admin/runtime/ws-bus/publish`
   - `GET /api/v1/admin/event-fabric/topics`
4. topic 必须字节级一致（grant / subscribe / publish 同值）。
5. 验收以 `sub_sent -> ack_ok -> event_ok` 三段为准。

## 8. Runtime Scheduler Host 调用规则

`local + proxy debug` 下，Scheduler 调 PowerX 底座能力的标准口径如下：

1. 插件前端只调用插件后端 runtime API：
   - `POST /api/v1/admin/runtime/scheduler/jobs`
   - `GET /api/v1/admin/runtime/scheduler/jobs?provider_mode=host`
   - `POST /api/v1/admin/runtime/scheduler/jobs/{job_id}/trigger`
   - `POST /api/v1/admin/runtime/scheduler/jobs/{job_id}/pause`
   - `POST /api/v1/admin/runtime/scheduler/jobs/{job_id}/resume`
2. 插件后端通过 framework host provider 调 PowerX：
   - `POST /api/v1/admin/scheduler/jobs`
   - `GET /api/v1/admin/scheduler/jobs`
   - `GET /api/v1/admin/scheduler/jobs/{job_id}`
   - `PATCH /api/v1/admin/scheduler/jobs/{job_id}`
   - `POST /api/v1/admin/scheduler/jobs/{job_id}/trigger`
   - `POST /api/v1/admin/scheduler/jobs/{job_id}/pause`
   - `POST /api/v1/admin/scheduler/jobs/{job_id}/resume`
3. `POWERX_PROVIDER_MODE=local + POWERX_PROXY=1` 使用：
   - `PX_GATEWAY_AUTH_SCHEME=apikey`
   - `PX_GATEWAY_API_KEY=<PowerX API Key>`
4. host Scheduler 请求不传 `tenant_uuid`。PowerX 从 ApiKey 鉴权上下文解析租户；如果显式传入 tenant，会导致 `SCHEDULER_TENANT_MISMATCH`。
5. PowerX API Key Profile 必须勾选 `com.corex.scheduler.jobs` 的 REST 权限。
6. 到期或手动触发后，标准通知 topic 为 `powerx.runtime.scheduler.triggered.v1`。

## 9. 代码定位

1. provider mode 解析入口：
   - 当前：`skeleton/backend/go-gin/internal/bootstrap/iam_resolver.go`
   - 目标：迁移为 provider resolver，不继续借 IAM 命名
2. 启动期模式日志与契约检查：
   - `skeleton/backend/go-gin/cmd/plugin/main.go`
   - 关键日志：`Runtime mode resolved (2x2)`、`WS/Capability routing`
3. taskbus provider / 宿主链路约束：
   - `skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go`
4. framework ws-bus 宿主客户端：
   - `framework/backend/go/runtime/wsbus/host_client.go`
5. skeleton 侧鉴权观测日志：
   - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_gateway_auth.go`
6. framework Scheduler host client：
   - `framework/backend/go/runtime/scheduler/http_host_client.go`
7. skeleton Scheduler runtime API：
   - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_job_handler.go`

## 10. 前后端 helper 基线

1. 后端模式与 host client 解析：
   - `resolveWSBusHostClientConfig(...)`
   - 输出：`host_delegated / local_proxy / standalone_local` 对应 host client 配置
2. 前端模式解析：
   - `resolveFrontendRuntimeMode()`
   - 输出：`mode`、`insidePowerX`、`powerxProxy`、`providerMode`、`gatewayAuthScheme`
3. framework WS URL 统一构造：
   - `createPluginWsClient(...).buildURL()`
   - 规则：优先 `wsBaseURL`，其次 `hostBaseURL`，最终兜底 `window.location.origin + wsPath`
4. 业务页面只消费后端 `/mode` 和 helper 结果，不自行拼接 WS/API 地址。

## 11. 最小自检清单

1. 启动日志出现：
   - `Runtime mode resolved (2x2)`
   - `WS/Capability routing`
2. 宿主链路鉴权日志包含：
   - `gateway_auth_scheme`
   - `outbound_token_source`
   - `provider_mode`
3. WS 联调必须同时满足：
   - `Grant=ok`
   - `Publish=ok`
   - `ack_ok=true`
   - `event_ok=true`

## 12. Breaking Change

旧静态 Tool Token 链路已废弃：

1. `POWERX_PROVIDER_MODE=delegated + POWERX_PROXY=1`
   - 必须使用 STS access token
   - 缺少 STS/gRPC 契约变量会被拒绝
2. `POWERX_PROVIDER_MODE=local + POWERX_PROXY=1`
   - 本地联调用 ApiKey
   - 不读取静态 bearer tool token
3. `POWERX_PROXY=0`
   - 不依赖宿主 STS 凭证

迁移建议：

1. 宿主部署脚本显式注入 provider mode、proxy link、STS/gRPC 契约变量。
2. 本地联调脚本显式注入 provider mode、proxy link、ApiKey 契约变量。
3. 旧静态 Tool Token 环境变量直接删除。
