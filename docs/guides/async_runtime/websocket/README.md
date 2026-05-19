# WebSocket 子系统说明（插件侧）

> 对齐 PowerX `async_runtime/websocket` 结构。

## 1. 范围

1. 连接入口：`GET /api/ws`
2. 订阅协议：`subscribe/unsubscribe/ping`
3. 推送包结构：`welcome/ack/error/event`
4. 单连接多 topic 复用

## 2. 核心约束

1. 单页面尽量复用单条 WS 连接
2. topic 命名统一 `_topic.*`
3. 未授权 topic 必须返回 `error`
4. WebSocket 只做实时分发，不替代任务执行

## 3. 鉴权与模式边界

1. Standalone 模式可连接插件 WS：`ws://<plugin-host>/api/ws`。
2. 宿主内嵌模式（`POWERX_PROXY=1`）前端必须连接宿主 WS：`ws://<host>/api/ws`（Contract v2：`NUXT_PUBLIC_WS_ORIGIN + NUXT_PUBLIC_WS_PATH`）。
3. proxy 场景下插件调用宿主 `grant/publish` 的出站鉴权统一使用 STS access token（`aud=powerx:api`）。
   - 宿主模式由 `POWERX_STS_CLIENT_ID` / `POWERX_STS_CLIENT_SECRET` / `POWERX_GRPC_UPSTREAM_*` 注入。
   - `PX_GATEWAY_API_KEY` 仅 standalone 本地联调可选。
4. `grant/publish` 不再默认透传入站 delegated/user bearer（避免 `invalid audience`）。

## 4. 统一联调入口（新增）

1. 插件页面联调统一调用：`POST /api/v1/admin/runtime/ws-bus/test-flow`
2. `test-flow` 后端内部执行：`grant -> publish`，并返回 `flow_mode/echo_ok` 便于快速观测。
3. 前端页面不再直接分别调用 ws-bus 调试端点，而是统一调用 `test-flow`。

## 5. 地址识别责任分层（避免插件各自实现）

1. 前端 framework-client（`framework/frontend/nuxt/framework-client/ws.ts`）负责 WS URL 组装：
   - 输入：`wsBaseURL/hostBaseURL/wsPath/insidePowerX/token`
   - 兜底：当宿主模式缺少显式 baseURL 时，回退到 `window.location.origin + wsPath`，确保不会在本地参数阶段直接失败
2. 插件页面/业务 composable 不应自行拼接 `/_p/.../api/ws` 或猜端口，仅传 runtime 配置值。
3. 后端 framework host client（`framework/backend/go/runtime/wsbus/host_client.go`）负责 HTTP 上游地址与鉴权头的统一发送，不在业务 handler 重复实现。

## 6. test-flow 判定标准（当前版本）

调用 `POST /api/v1/admin/runtime/ws-bus/test-flow` 后，联调页面应联合观察：

1. `flow_mode`
   - `host_strict_ok`：宿主链路严格成功
   - `host_fallback_local_only`：宿主失败并回到本地（仅兼容场景）
   - `local_only`：纯本地模式
2. `host_reachable` / `host_grant_ok` / `host_publish_ok`
3. WS 诊断三段：`sub_sent=true`、`ack_ok=true`、`event_ok=true`
