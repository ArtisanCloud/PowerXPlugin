# Async Runtime Scheduler 模式切换开发文档

> 位置约定：开发实施细节放 `docs/plan/develop/*`。本文件用于落地 `standalone local` 与 `delegated proxy` 的统一调度切换实现。

## 1. 目标

1. 调度触发统一走 framework 抽象，业务层不感知运行模式。
2. 根据启动环境自动识别并切换：`standalone local` / `delegated proxy`。
3. 调度链路与 Task/EventBus 复用同一概念：Scheduler 只触发，执行统一 `emit -> event bridge -> taskbus`。

## 2. 范围与非范围

范围：

1. Scheduler 模式识别与 provider 切换规则。
2. Cron Job 接入规范（调用统一 `EventEmitter.Emit`）。
3. 双模式联调、验收、排障基线。

非范围：

1. 不新增独立业务协议。
2. 不让前端决定调度模式。
3. 不在业务代码中新增 `if POWERX_PROXY` 分支。

## 3. 现状（2026-03-23）

1. EventBridge/TaskBus 模式切换能力已具备：
   - `skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go`
   - `skeleton/backend/go-gin/cmd/plugin/main.go`
2. Scheduler 组件已实现（Register/Start/Stop），但主启动流程未接线：
   - `skeleton/backend/go-gin/internal/jobs/integration/scheduler.go`
3. 已有可观测入口：
   - `POST /api/v1/admin/runtime/event-bridge/emit`
   - `GET /api/v1/admin/runtime/metrics`

## 4. 模式识别契约（最终口径）

## 4.1 识别输入

1. `POWERX_PROXY`：主模式信号。
2. `runtime.event_bridge.taskbus_provider`：执行通道（`redis`/`host`）。
3. `gateway.auth_scheme` 与凭证：proxy 出站鉴权。

## 4.2 切换矩阵

1. `POWERX_PROXY!=1`：
   - 模式：`standalone local`
   - 推荐 `taskbus_provider=redis`
2. `POWERX_PROXY=1`：
   - 模式：`delegated proxy`
   - 推荐 `taskbus_provider=host`

## 4.3 禁止事项

1. 业务 service/job 里判断 `POWERX_PROXY`。
2. Cron Job 内直接调用 WS publish 绕过 EventBridge。

## 5. 开发落地步骤

## 5.1 启动层接线（必须）

1. 在 `cmd/plugin/main.go` 初始化 integration scheduler。
2. 使用统一 logger（建议 `component=integration.scheduler`）。
3. 进程启动时 `Start(ctx)`，进程退出时 `Stop(shutdownCtx)`。

## 5.2 Job 注册规范（必须）

1. Job 仅做“触发 + 组装 payload”，执行统一调用 `deps.EventEmitter.Emit(...)`。
2. topic 必须使用 `_topic.*` 命名并在 `skeleton/plugin.yaml` 声明。
3. payload 包含可追踪字段（如 `source=scheduler`, `trace_id`）。

## 5.3 配置规范（必须）

1. `standalone local`：

```yaml
runtime:
  event_bridge:
    mode: "taskbus"
    taskbus_provider: "redis"
```

2. `delegated proxy`：

```yaml
runtime:
  event_bridge:
    mode: "taskbus"
    taskbus_provider: "host"
gateway:
  auth_scheme: "apikey" # 或 bearer（按部署）
```

3. 调度窗口（SLA 示例）：

```yaml
operations:
  sla:
    daily_cron: "0 5 * * *"
    monthly_cron: "0 6 1 * *"
    quarterly_cron: "0 7 1 */3 *"
```

## 6. 最小代码锚点

1. 模式与 provider 选择：`skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go`
2. Emitter 工厂装配：`skeleton/backend/go-gin/cmd/plugin/main.go`
3. Scheduler 引擎：`skeleton/backend/go-gin/internal/jobs/integration/scheduler.go`
4. Runtime 调试入口：`skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/event_bridge_debug_handler.go`

## 7. 验收脚本（开发回归）

## 7.1 Standalone Local

```bash
export POWERX_PROXY=0
# 启动插件后，先订阅 WS
wscat -c "ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN"
```

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/event-bridge/emit \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"_topic.template.update","payload":{"source":"scheduler-smoke","progress":25}}'
```

预期：WS 收到 `ack` + `event`，且 `plugin_event_bridge_emit_total` 增长。

## 7.2 Delegated Proxy

```bash
export POWERX_PROXY=1
export PX_GATEWAY_AUTH_SCHEME=apikey
export PX_GATEWAY_API_KEY=<key>
# 启动插件后，连接插件入口 WS
wscat -c "ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN"
```

联调顺序：先 topic create，再 grant，再 emit/publish。详见：

1. `docs/guides/async_runtime/event_fabric/integration_playbook.md`
2. `docs/guides/async_runtime/websocket/debug_playbook.md`

## 8. Done Definition

1. Scheduler 在主启动流程已接线并可优雅启停。
2. 至少 1 个 Cron Job 通过统一 `emit` 触发事件。
3. `standalone local` 与 `delegated proxy` 两种模式均通过第 7 节验收。
4. 日志能检索 `trace_id/topic/status`，proxy 下可见 `gateway_auth_scheme/outbound_token_source`。

## 9. 常见失败与处理

1. `401`：`gateway.auth_scheme` 与凭证不匹配。
2. `403 topic not allowed`：topic 未创建/ACL 未授权/API Key 快照未轮换。
3. 仅有 `ack` 无 `event`：topic 不一致或未先 grant。
4. 模式错乱：检查 `POWERX_PROXY` 与 `taskbus_provider` 是否按矩阵配对。

## 10. US3 失败闭环实现对齐（2026-03-24）

### 10.1 管理端接口

1. `POST /api/v1/admin/runtime/scheduler/dispatches/{dispatchId}/retry`
2. `POST /api/v1/admin/runtime/scheduler/dispatches/{dispatchId}/pause`
3. `POST /api/v1/admin/runtime/scheduler/tickets/{ticketId}/resume`

### 10.2 状态机与权限口径

1. 重试上限：默认 3 次（配置项 `operations.scheduler.retry_max_attempts`，范围 1-10）。
2. 第 1~(N-1) 次重试返回 `202`；第 N 次超限返回 `409`。
3. 超限后执行 pause，创建恢复工单（`201`），任务进入 `paused`。
4. resume 仅允许 `ops/admin`；非授权角色返回 `403`。
5. 恢复成功后重试窗口重置，下一次 retry 返回 `202`。

### 10.3 审计与可观测字段

1. 恢复操作写入审计记录：`ticket_id / dispatch_id / operator_id / operator_role / recorded_at`。
2. 联调日志需可检索：`trace_id / topic / status / gateway_auth_scheme / outbound_token_source`。

### 10.4 Phase 6 回归命令（已执行）

```bash
mkdir -p tmp/gocache tmp/gomodcache && cd skeleton/backend/go-gin && \
GOCACHE=$PWD/../../tmp/gocache GOMODCACHE=$PWD/../../tmp/gomodcache \
go test ./cmd/plugin ./internal/config ./internal/services/admin/runtime_ops \
  ./internal/transport/http/admin/runtime_ops ./tests/integration \
  -run 'Scheduler|TaskBusProvider|ValidateSchedulerRetryMaxAttemptsRange|DefaultSchedulerConfigValidation' \
  -count=1
```

结果：5/5 包通过（`cmd/plugin`、`internal/config`、`runtime_ops service`、`runtime_ops handler`、`tests/integration`）。
