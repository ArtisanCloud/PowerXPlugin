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
2. 宿主内嵌模式（`POWERX_PROXY=1`）前端必须连接宿主 WS：`ws://<host>/api/ws`（Contract v2：`PX_WS_BASE_URL + NUXT_PUBLIC_WS_URL`）。
3. proxy 场景下插件调用宿主 `grant/publish` 的出站鉴权优先使用宿主契约凭证：
   - `PX_GATEWAY_AUTH_SCHEME=bearer`：使用 `PX_PLUGIN_TOOL_TOKEN`
   - `PX_GATEWAY_AUTH_SCHEME=apikey`：使用 `PX_GATEWAY_API_KEY`
4. `grant/publish` 不再默认透传入站 delegated/user bearer（避免 `invalid audience`）。

## 4. 统一联调入口（新增）

1. 插件页面联调统一调用：`POST /api/v1/admin/runtime/ws-bus/test-flow`
2. `test-flow` 后端内部执行：`grant -> publish`，并返回 `flow_mode/echo_ok` 便于快速观测。
3. 前端页面不再直接分别调用 `internal/ws-bus/grant` 与 `internal/ws-bus/publish`。
