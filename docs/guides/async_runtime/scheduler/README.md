# Scheduler / Cron（插件侧）

> 开发实施（模式识别/切换/代码接线）请优先阅读：
> `docs/plan/develop/async_runtime/schedule/scheduler-mode-switch-implementation.md`

## 1. 目标与范围

1. 明确“何时触发”（Scheduler）与“如何执行”（Task/EventBus）的职责边界。
2. 明确 `standalone local` 与 `delegated proxy` 的模式识别与切换策略。
3. 确保手动触发与定时触发进入同一事件执行链路，避免双轨实现。

## 2. 核心原则（与 Task/EventBus 对齐）

1. Scheduler 只负责触发，不负责业务执行。
2. 执行统一走 EventBridge/TaskBus 抽象，不在业务代码写 `if proxy/if standalone` 分支。
3. 模式切换由启动环境 + framework 配置决定，不由页面或接口参数临时决定。
4. 手动触发与 Cron 触发必须复用同一个 `emit(topic, payload)` 入口。
5. Scheduler 标准 topic 统一为 `powerx.runtime.scheduler.triggered.v1`（遵循 `powerx.<domain>.<subdomain>.<action>.v<version>`）。

## 3. 模式识别与切换（启动期决策）

### 3.1 识别信号

1. 主信号：`POWERX_PROXY`
2. 执行提供者：`runtime.event_bridge.taskbus_provider`
3. 网关鉴权：`gateway.auth_scheme` + 对应凭证（`tool_token` 或 `api_key`）

### 3.2 推荐决策表

1. `POWERX_PROXY!=1`：
   - 运行模式：`standalone local`
   - 推荐 `taskbus_provider=redis`
   - 说明：事件在插件本地闭环，WS 走 `:8078/api/ws`
2. `POWERX_PROXY=1`：
   - 运行模式：`delegated proxy`
   - 推荐 `taskbus_provider=host`
   - 说明：插件入口仍是 `:8078`，出站由 framework gateway 代理到底座

### 3.3 封装边界（必须遵守）

1. 业务层只声明“要触发哪个 `_topic.*`”，不感知 host/redis。
2. provider 选择在启动期完成，参考：
   - `skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go`
   - `skeleton/backend/go-gin/cmd/plugin/main.go`
3. Scheduler 组件只注册 Job 与调度频率，参考：
   - `skeleton/backend/go-gin/internal/jobs/integration/scheduler.go`

## 4. 配置建议（最小可用）

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
  auth_scheme: "apikey" # 或 bearer（取决于部署策略）
```

3. SLA 调度窗口配置（业务 Cron 示例）：

```yaml
operations:
  sla:
    timezone: "UTC"
    daily_cron: "0 5 * * *"
    monthly_cron: "0 6 1 * *"
    quarterly_cron: "0 7 1 */3 *"
```

## 5. 联调方法（先验证“同链路”）

说明：当前仓库已提供 Scheduler 组件与 EventBridge 主链路。业务 Cron 任务应注册到 Scheduler，并在 Job 内调用同一个 `emit` 入口。

### Step 1：准备 WS 订阅

1. `standalone local`：`ws://127.0.0.1:8078/api/ws`
2. `delegated proxy`：同样连插件 `:8078/api/ws`，按 `websocket/debug_playbook.md` 完成 topic/create/grant

### Step 2：模拟“调度触发事件”

先用 runtime 调试入口替代 Cron 触发，验证执行链路一致性：

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/event-bridge/emit \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"powerx.runtime.scheduler.triggered.v1","payload":{"source":"manual","trigger_source":"manual","job_name":"runtime.scheduler.trigger","business_action":"reconcile","status":"queued","trace_id":"manual-trace-001"}}'
```

再执行一次 Cron 触发（由 scheduler 自动发出同 topic 事件），payload 语义保持一致：

- `business_action=reconcile`
- `status=queued`
- `trace_id` 必填，且应与事件 meta 中 `trace_id` 一致

### Step 3：验收

1. WS 先收到 `ack`，后收到 `event`
2. 手动触发与 Cron 触发必须同 topic（`powerx.runtime.scheduler.triggered.v1`）且语义字段一致（`business_action/status`）
3. 每次触发均可检索 `trace_id`，并可完成“触发 -> 分发”链路关联
4. 指标可见：

```bash
curl -sS http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_(emit_total|latency_ms)'
```

5. 日志可检索 `trace_id/topic/status`

## 6. Cron 接入约束（实施时）

1. Cron Job 内禁止直接 publish WS，必须先 `emit` 到 EventBridge。
2. Cron Job 与手工触发写入相同 topic，保持消费者与告警规则复用。
3. `delegated proxy` 模式下，租户解析由底座按凭证处理，插件任务不拼接租户参数。

## 7. 常见问题

1. 有调度日志但无 WS 事件：检查 topic ACL / grant / subscribe 是否完整。
2. proxy 下 `403 topic not allowed`：检查 topic 是否已创建、profile 权限是否覆盖、API Key 是否已轮换。
3. `401`：`gateway.auth_scheme` 与实际凭证不匹配（Bearer/ApiKey 混用）。
4. 两种模式行为不一致：优先检查 `POWERX_PROXY` 与 `taskbus_provider` 是否按决策表配对。
