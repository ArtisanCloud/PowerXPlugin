# Standalone / Host 模式判定规则

## 1. 目标

统一插件在本地开发、宿主联调、宿主安装三类场景下的模式判定与鉴权策略，避免 `IAMMode`、`POWERX_PROXY`、`insidePowerX` 混用导致链路偏差。

## 2. 单一判定口径（后端真实规则）

后端以 `IAMMode` + `POWERX_PROXY` 解析“有效运行模式（effective mode）”：

1. 先解析 `IAMMode`（`config.context.iam_mode` 优先，其次环境变量）。
2. 若 `IAMMode=delegated`，直接按宿主链路处理（等效 `effective_proxy=1`）。
3. 若 `IAMMode=local`，再看 `POWERX_PROXY`：
   - `0` => 本地链路
   - `1` => 宿主链路
4. 仅当宿主链路生效时，才解析出站鉴权凭证。

## 3. 三种有效模式（推荐心智模型）

| 模式 | IAMMode | POWERX_PROXY | effective_proxy | 说明 |
|---|---|---|---|---|
| M1 纯本地 | local | 0 | 0 | 本地 IAM + 本地能力链路 |
| M2 本地联调宿主能力 | local | 1 | 1 | 本地启动插件，但 WS/能力走宿主 |
| M3 委派模式 | delegated | 任意（按 1 处理） | 1 | 宿主语义模式（含本地模拟宿主） |

> 约束：`delegated` 不再依赖 `POWERX_PROXY` 是否显式为 `1`，运行时视为宿主链路。

## 4. 前端变量职责（和后端解耦）

前端不参与后端鉴权决策，只做“页面运行上下文”识别：

1. 前端主判定字段：`runtimeConfig.public.insidePowerX`。
2. `insidePowerX` 表示页面是否运行在宿主插件容器语义中（影响路由/baseURL/代理策略）。
3. `NUXT_PUBLIC_INSIDE_POWERX` 仅是前端覆盖入口，不代表后端真实运行模式。
4. `POWERX_PROXY` 是后端链路开关；可在构建时派生到前端，但两者职责不同。

推荐实践：

1. `make dist`（宿主安装包）固定 `insidePowerX=1`。
2. 本地 standalone 默认 `insidePowerX=0`。
3. 本地模拟宿主嵌入时可设 `insidePowerX=1`，但后端是否走宿主链路仍由 `IAMMode/POWERX_PROXY` 决定。

## 5. 宿主链路鉴权规则（effective_proxy=1）

1. 标准宿主链路统一使用 STS access token（Bearer，`aud=powerx:api`）。
2. 不默认透传入站 delegated/user bearer 到宿主 ws-bus 接口。
3. `PX_GATEWAY_API_KEY` 仅用于 standalone 本地联调，不作为宿主发布默认口径。

## 6. WS / Capability Contract v2 对齐

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

## 6.1 Runtime Scheduler Host 调用规则

`local + proxy` 下，Scheduler 调用 PowerX 底座能力的标准口径如下：

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
3. `IAMMode=local + POWERX_PROXY=1` 使用：
   - `PX_GATEWAY_AUTH_SCHEME=apikey`
   - `PX_GATEWAY_API_KEY=<PowerX API Key>`
   - 出站 header：`Authorization: ApiKey <key>`
4. host Scheduler 请求不传 `tenant_uuid`。PowerX 从 ApiKey 鉴权上下文解析租户；如果显式传入 tenant，会导致 `SCHEDULER_TENANT_MISMATCH`。
5. PowerX API Key Profile 必须勾选 `com.corex.scheduler.jobs` 的 REST 权限；权限目录应包含 `admin_scheduler_jobs`、`admin_scheduler_jobs_job_id`、`pause/resume/trigger/runs`。
6. 到期或手动触发后，标准通知 topic 为 `powerx.runtime.scheduler.triggered.v1`，插件通过既有 EventBridge/WSBus 链路接收。

## 7. 代码定位（当前实现）

1. IAM 解析入口：
   - `skeleton/backend/go-gin/internal/bootstrap/iam_resolver.go`
2. 启动期模式日志与契约检查：
   - `skeleton/backend/go-gin/cmd/plugin/main.go`
   - 关键日志：`Runtime mode resolved (2x2)`、`WS/Capability routing`
3. taskbus provider / 宿主链路约束：
   - `skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go`
4. framework ws-bus 宿主客户端（标准路径与调用入口）：
   - `framework/backend/go/runtime/wsbus/host_client.go`
5. skeleton 侧鉴权观测日志（用于排查 token 来源）：
   - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_gateway_auth.go`
6. framework Scheduler host client：
   - `framework/backend/go/runtime/scheduler/http_host_client.go`
7. skeleton Scheduler runtime API：
   - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_job_handler.go`

## 8. Framework 统一封装（当前基线）

为避免业务页面重复判断模式，前后端都已收敛到统一 helper：

1. 后端模式与 host client 解析：
   - `resolveWSBusHostClientConfig(...)`
   - 位置：`skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_gateway_auth.go`
   - 输出：`host_delegated / local_proxy / standalone_local` 对应的 host client 配置（`baseURL/apiPrefix/authScheme/token|apiKey`）
2. 前端模式解析：
   - `resolveFrontendRuntimeMode()`
   - 位置：`skeleton/web-admin/nuxt/app/utils/runtime-mode.ts`
   - 输出：`mode`、`insidePowerX`、`powerxProxy`、`iamMode`、`gatewayAuthScheme`
3. framework WS URL 统一构造：
   - `createPluginWsClient(...).buildURL()`
   - 位置：`framework/frontend/nuxt/framework-client/ws.ts`
   - 规则：优先 `wsBaseURL`，其次 `hostBaseURL`，最终兜底 `window.location.origin + wsPath`（避免因页面漏传参数直接抛错）
4. 业务页面只消费 helper 结果，不再自行拼接 WS/API 地址。

## 9. 最小自检清单

1. 启动日志出现：
   - `Runtime mode resolved (2x2)`
   - `WS/Capability routing`
2. 宿主链路鉴权日志包含：
   - `gateway_auth_scheme`
   - `outbound_token_source`
3. WS 联调必须同时满足：
   - `Grant=ok`
   - `Publish=ok`
   - `ack_ok=true`
   - `event_ok=true`

## 10. Breaking Change（Token 收敛）

从当前版本开始，旧静态 Tool Token 链路已废弃：

1. `IAMMode=delegated` 且 `POWERX_PROXY=1`：
   - 必须使用 STS access token（`aud=powerx:api`）
   - 缺少 `POWERX_STS_CLIENT_ID` / `POWERX_STS_CLIENT_SECRET` / `POWERX_GRPC_UPSTREAM_*` 会被拒绝（fail-fast）
2. `IAMMode=local` 且 `POWERX_PROXY=1`：
   - 本地联调使用 `PX_GATEWAY_AUTH_SCHEME=apikey` + `PX_GATEWAY_API_KEY`
   - 不再读取静态 bearer tool token
3. `POWERX_PROXY=0`：
   - 不依赖宿主 STS 凭证

迁移建议：

1. 宿主部署脚本只注入 `PX_GATEWAY_BASE_URL` 与 STS/gRPC 契约变量
2. 本地联调脚本只注入 ApiKey 契约变量
3. 旧静态 Tool Token 环境变量应直接删除
