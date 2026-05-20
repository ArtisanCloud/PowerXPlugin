# Scheduler / Cron（插件侧）

> 开发实施（模式识别/切换/代码接线）请优先阅读：
> `docs/plan/develop/async_runtime/schedule/scheduler-mode-switch-implementation.md`

## 1. 目标与范围

1. 明确“何时触发”（Scheduler）与“如何执行”（Task/EventBus）的职责边界。
2. 明确 `standalone local` 与 `delegated proxy` 的模式识别与切换策略。
3. 确保手动触发与定时触发进入同一事件执行链路，避免双轨实现。
4. 约束后续业务插件（如 AI Craft）通过 framework scheduler facade 接入，不自行实现业务本地 scheduler。

## 2. 核心原则（与 Task/EventBus 对齐）

1. Scheduler 只负责触发，不负责业务执行。
2. 执行统一走 EventBridge/TaskBus 抽象，不在业务代码写 `if proxy/if standalone` 分支。
3. 模式切换由启动环境 + framework 配置决定，不由页面或接口参数临时决定。
4. 手动触发与 Cron 触发必须复用同一个 `emit(topic, payload)` 入口。
5. Scheduler 标准 topic 统一为 `powerx.runtime.scheduler.triggered.v1`（遵循 `powerx.<domain>.<subdomain>.<action>.v<version>`）。
6. 旧规划里的 `scheduler.job.triggered` 仅作为历史草案名称；新实现优先统一到 `powerx.runtime.scheduler.triggered.v1`。

## 2.1 统一规划口径

详细规划见：

```text
docs/plan/014-framework-scheduler.md
```

当前落地判断：

1. backend framework 已有初版 `runtime/scheduler` facade。
2. frontend framework 已有初版 scheduler client/composable，页面应通过统一 client 调用。
3. skeleton 管理端已暴露 `/api/v1/admin/runtime/scheduler/jobs` 系列 API。
4. PowerX 底座仍需要确认 `powerx.scheduler.v1.SchedulerService` 是否已经真实可调用。
5. 插件通用 scheduler 不复用 `/admin/event-fabric/cron/jobs`，该接口偏底座内置运维 job。
6. 业务插件应通过 framework 注册 `once`、`interval`、`cron` job。
7. Scheduler 到点只发布事件，业务执行仍由 EventBridge/TaskBus 承接。

## 3. 模式识别与切换（启动期决策）

### 3.1 识别信号

1. 主信号：`POWERX_PROXY`
2. 执行提供者：`runtime.event_bridge.taskbus_provider`
3. 网关鉴权：宿主 delegated 使用 STS token provider；standalone 使用 `api_key`

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
4. 后续 `runtime/scheduler` facade 落地后，业务代码应依赖 framework 接口，不再直接依赖 skeleton 本地 scheduler。

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

## 7. 业务 Schedule 接入建议

### 7.1 周期扫描类

适用于 SLA 聚合、secret rotation 扫描、webhook retry due 扫描、license renewal 检查。

建议使用 `cron` 或 `interval` job。任务触发后扫描数据库 due records，再进入业务处理。

### 7.2 Durable Due Job 类

适用于 AI Craft 这类业务节点：

1. 付款后 50% 进度提醒。
2. 交付前提醒。
3. 样品延期检查。
4. 工厂催办。
5. 超时未确认补偿。

建议注册 `once` job，并在 payload 中携带 `business_action`、业务主键和 `idempotency_key`。不要为每个订单维护本地内存 timer。

示例：

```go
scheduler.CreateJob(ctx, scheduler.JobSpec{
    TenantUUID:   tenantUUID,
    OwnerType:    "plugin",
    OwnerID:      "com.powerx.plugins.ai-craft",
    Name:         "sample_progress_50",
    ScheduleType: "once",
    ScheduleExpr: eta50.Format(time.RFC3339),
    Topic:        "powerx.runtime.scheduler.triggered.v1",
    Payload: map[string]any{
        "business_action": "sample_progress_50",
        "design_task_id":  designTaskID,
        "order_id":        orderID,
    },
})
```

### 7.3 Provider 选择

1. `local`：本地开发/standalone，建议 DB due-scan 或可注入 store。
2. `host`：生产优先，调用 PowerX 底座 `powerx.scheduler.v1.SchedulerService`。
3. `dual`：仅用于迁移比对，不作为长期默认。

### 7.4 页面调用约束

页面不直接拼底座 scheduler 地址，也不直接调用 `/admin/event-fabric/cron/jobs`。统一使用 frontend framework scheduler client：

```ts
const scheduler = usePowerXScheduler({
  pluginId: "com.powerx.plugins.ai-craft",
  tenantUuid
})

await scheduler.createJob({
  name: "sample_progress_50",
  schedule_type: "once",
  schedule_expr: eta50,
  payload: {
    business_action: "sample_progress_50",
    order_id: orderId
  }
})
```

## 8. 常见问题

1. 有调度日志但无 WS 事件：检查 topic ACL / grant / subscribe 是否完整。
2. proxy 下 `403 topic not allowed`：检查 topic 是否已创建、profile 权限是否覆盖、API Key 是否已轮换。
3. `401`：`gateway.auth_scheme` 与实际凭证不匹配（Bearer/ApiKey 混用）。
4. 两种模式行为不一致：优先检查 `POWERX_PROXY` 与 `taskbus_provider` 是否按决策表配对。
5. 底座 SchedulerService 只存在 proto/capability 但不可调用：先补底座服务实现与 framework HostProvider，再接入业务插件。
