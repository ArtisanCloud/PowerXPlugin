# Task 子系统说明（插件侧）

> 平台对齐入口：`docs/guides/async_runtime/README.md`

## 1. 这份文档解决什么问题

1. 如何确认插件 Task 主链路打通
2. 如何在 standalone / proxy 下做最短手工验证
3. 如何判断“任务执行”与“实时推送”是否都正常

## 2. 前置条件

1. `skeleton/plugin.yaml` 已声明目标 `_topic.*`
2. 后端默认强鉴权（必须携带有效凭证）
3. 推荐配置：
   - standalone：`runtime.event_bridge.taskbus_provider=redis`
   - standalone + proxy：`runtime.event_bridge.taskbus_provider=host`
4. proxy 场景需先准备底座 API Key（profile `permission_ids` 已覆盖）

## 3. 手工验证（最短路径）

### Step 1：触发任务事件（插件）

`POST /api/v1/admin/runtime/event-bridge/emit`

预期：

1. 返回 `ok=true`
2. 返回 `trace_id`

### Step 2：看运行指标（插件）

`GET /api/v1/admin/runtime/metrics`

预期：

1. `plugin_event_bridge_emit_total` 增长
2. `plugin_event_bridge_latency_ms` 有值

### Step 3：验证实时可见（WS）

1. 先订阅目标 topic
2. 再触发 `emit` 或 `ws-bus/publish`
3. 预期先 `ack` 后 `event`

## 4. 模式差异

1. standalone：订阅插件 `ws://127.0.0.1:8078/api/ws`
2. standalone + proxy：
   - 业务入口仍调用插件 `:8078`
   - 插件出站到底座按配置走 ApiKey
   - 底座订阅与 ACL 验证按 PowerX 规则执行

## 5. 最小验收标准

1. Task 触发成功（`ok=true`）
2. 指标增长可见
3. WS 实时推送可见
4. 页面不依赖轮询驱动任务执行

## 6. 参考文档

1. `docs/guides/async_runtime/task/mechanism.md`
2. `docs/guides/async_runtime/event_fabric/integration_playbook.md`
3. `docs/guides/async_runtime/websocket/debug_playbook.md`
