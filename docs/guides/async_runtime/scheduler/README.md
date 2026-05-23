# Scheduler / Cron（插件侧）

> 开发实施（模式识别/切换/代码接线）请优先阅读：
> `docs/plan/develop/async_runtime/schedule/scheduler-mode-switch-implementation.md`

## 1. 目标与范围

1. 明确“何时触发”（Scheduler）与“如何执行”（Task/EventBus）的职责边界。
2. 明确 `standalone local`、`local + proxy` 与 `delegated proxy` 的模式识别与切换策略。
3. 确保手动触发与定时触发进入同一事件执行链路，避免双轨实现。
4. 约束后续业务插件（如 AI Craft）通过 framework scheduler facade 接入，不自行实现业务本地 scheduler。

## 2. 核心原则（与 Task/EventBus 对齐）

1. Scheduler 只负责触发，不负责业务执行。
2. 执行统一走 EventBridge/TaskBus 抽象，不在业务代码写 `if proxy/if standalone` 分支。
3. 模式切换由启动环境 + framework 配置决定，不由页面或接口参数临时决定。
4. 手动触发与 Cron 触发必须复用同一个 `emit(topic, payload)` 入口。
5. Scheduler 标准 topic 统一为 `powerx.runtime.scheduler.triggered.v1`（遵循 `powerx.<domain>.<subdomain>.<action>.v<version>`）。
6. 旧规划里的 `scheduler.job.triggered` 仅作为历史草案名称；新实现优先统一到 `powerx.runtime.scheduler.triggered.v1`。

## 2.1 当前实现口径

详细规划见：

```text
docs/plan/014-framework-scheduler.md
```

当前落地判断：

1. backend framework 已提供 `runtime/scheduler` facade，包含 `local` 与 `host` provider。
2. host provider 已接入 PowerX 底座 REST Scheduler：`/api/v1/admin/scheduler/jobs`。
3. skeleton 管理端暴露 `/api/v1/admin/runtime/scheduler/jobs` 系列 API，页面通过该 API 调用，不直连 PowerX。
4. frontend framework 已有 scheduler client/composable，`framework-lab` 提供“本地 Scheduler / 网关 Scheduler”双 tab 联调入口。
5. 插件通用 scheduler 不复用 `/admin/event-fabric/cron/jobs`，该接口偏底座内置运维 job。
6. 业务插件应通过 framework 注册 `once`、`interval`、`cron` job。
7. Scheduler 到点只发布事件，业务执行仍由 EventBridge/TaskBus/WSBus 承接。
8. 标准通知 topic 为 `powerx.runtime.scheduler.triggered.v1`，插件侧已可收到 host Scheduler 到期通知。

## 3. 模式识别与切换（启动期决策）

### 3.1 识别信号

1. 主信号：`IAMMode` + `POWERX_PROXY`
2. 执行提供者：`runtime.event_bridge.taskbus_provider`
3. 网关鉴权：`delegated` 使用 STS/Bearer；`local + proxy` 使用 ApiKey

### 3.2 推荐决策表

1. `POWERX_PROXY!=1`：
   - 运行模式：`standalone local`
   - 推荐 `taskbus_provider=redis`
   - 说明：事件在插件本地闭环，WS 走 `:8078/api/ws`
2. `IAMMode=local` 且 `POWERX_PROXY=1`：
   - 运行模式：`local + proxy`
   - 推荐 `PX_GATEWAY_AUTH_SCHEME=apikey`
   - 说明：本地启动插件，但 Scheduler/WS/能力出站走 PowerX 底座网关
3. `IAMMode=delegated`：
   - 运行模式：`delegated proxy`
   - 推荐 `taskbus_provider=host`
   - 说明：宿主语义模式，出站由 framework gateway 代理到底座

### 3.3 封装边界（必须遵守）

1. 业务层只声明“要触发哪个 `_topic.*`”，不感知 host/redis。
2. provider 选择在启动期完成，参考：
   - `skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go`
   - `skeleton/backend/go-gin/cmd/plugin/main.go`
3. Scheduler 组件只注册 Job 与调度频率，参考：
   - `skeleton/backend/go-gin/internal/jobs/integration/scheduler.go`
4. 后续 `runtime/scheduler` facade 落地后，业务代码应依赖 framework 接口，不再直接依赖 skeleton 本地 scheduler。

### 3.4 Host Scheduler 标准调用规则

1. 插件页面只调用插件后端：

```text
POST /api/v1/admin/runtime/scheduler/jobs
GET  /api/v1/admin/runtime/scheduler/jobs?provider_mode=host
POST /api/v1/admin/runtime/scheduler/jobs/{job_id}/trigger
POST /api/v1/admin/runtime/scheduler/jobs/{job_id}/pause
POST /api/v1/admin/runtime/scheduler/jobs/{job_id}/resume
```

2. 插件后端通过 framework host client 调 PowerX 底座：

```text
POST /api/v1/admin/scheduler/jobs
GET  /api/v1/admin/scheduler/jobs
GET  /api/v1/admin/scheduler/jobs/{job_id}
PATCH /api/v1/admin/scheduler/jobs/{job_id}
POST /api/v1/admin/scheduler/jobs/{job_id}/trigger
POST /api/v1/admin/scheduler/jobs/{job_id}/pause
POST /api/v1/admin/scheduler/jobs/{job_id}/resume
```

3. host Scheduler 请求不传 `tenant_uuid`。租户由 PowerX 根据 ApiKey/Bearer 鉴权上下文解析；若插件显式传入 `tenant_uuid`，framework host client 也会在出站前清空。
4. `owner_type/owner_id` 表示调度任务归属，测试按钮默认使用当前 skeleton 插件 ID。PowerX 对 owner 的最终授权以底座实现为准。
5. PowerX API Key Profile 必须包含 `com.corex.scheduler.jobs` 对应 REST 权限。权限目录中应能看到并勾选 `admin_scheduler_jobs`、`admin_scheduler_jobs_job_id`、`pause/resume/trigger/runs` 等资源。

## 4. 配置建议（最小可用）

1. `standalone local`：

```yaml
runtime:
  event_bridge:
    mode: "taskbus"
    taskbus_provider: "redis"
```

2. `local + proxy`：

```yaml
context:
  iam_mode: "local"
gateway:
  base_url: "http://127.0.0.1:8077"
  api_prefix: "/api/v1"
  auth_scheme: "apikey"
  api_key: "<PowerX API Key>"
runtime:
  event_bridge:
    mode: "taskbus"
    taskbus_provider: "host"
```

环境变量等价写法：

```bash
POWERX_PROXY=1
PX_GATEWAY_BASE_URL=http://127.0.0.1:8077
PX_GATEWAY_AUTH_SCHEME=apikey
PX_GATEWAY_API_KEY=<PowerX API Key>
```

3. `delegated proxy`：

```yaml
runtime:
  event_bridge:
    mode: "taskbus"
    taskbus_provider: "host"
gateway:
  auth_scheme: "bearer"
```

4. SLA 调度窗口配置（业务 Cron 示例）：

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

### Step 1：准备 PowerX API Key 权限

1. 在 PowerX 后台进入“设置 / API Key 管理”。
2. 选择当前 Profile，勾选 `Runtime Scheduler` 相关 REST 权限：
   - `admin_scheduler_jobs · create/list`
   - `admin_scheduler_jobs_job_id · read/update`
   - `admin_scheduler_jobs_job_id_trigger · create`
   - `admin_scheduler_jobs_job_id_pause · create`
   - `admin_scheduler_jobs_job_id_resume · create`
   - `admin_scheduler_jobs_job_id_runs · list`
3. 保存权限，确认日志出现类似 `profile permissions synced ... snapshot_permissions=...`。
4. 使用该 Profile 下的 API Key 配置插件 `PX_GATEWAY_API_KEY`。

### Step 2：准备 WS 订阅

1. `standalone local`：`ws://127.0.0.1:8078/api/ws`
2. `local + proxy` / `delegated proxy`：同样连插件 `:8078/api/ws`，由插件后端完成 host 链路调用；按 `websocket/debug_playbook.md` 完成 topic/create/grant。
3. 标准 topic：`powerx.runtime.scheduler.triggered.v1`。

### Step 3：页面创建 Host Scheduler Job

1. 打开 framework lab 页面。
2. 选择“网关 Scheduler”。
3. 点击“创建 Scheduler 样例”。
4. 预期：
   - 插件请求 `POST /api/v1/admin/runtime/scheduler/jobs`
   - 插件转发到 PowerX `POST /api/v1/admin/scheduler/jobs`
   - PowerX 返回创建成功，列表里出现新 job
   - 到期后插件收到 `powerx.runtime.scheduler.triggered.v1` 通知

### Step 4：模拟“调度触发事件”

也可以先用 runtime 调试入口替代 Cron 触发，验证执行链路一致性：

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

### Step 5：验收

1. WS 先收到 `ack`，后收到 `event`
2. 手动触发与 Cron 触发必须同 topic（`powerx.runtime.scheduler.triggered.v1`）且语义字段一致（`business_action/status`）
3. 每次触发均可检索 `trace_id`，并可完成“触发 -> 分发”链路关联
4. 指标可见：

```bash
curl -sS http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_(emit_total|latency_ms)'
```

5. 日志可检索 `trace_id/topic/status`
6. PowerX `logs/info.log` 可检索 `POST /api/v1/admin/scheduler/jobs` 的 `status=200`，失败时可按 `request_id` 定位。

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
5. API Key 权限页看不到 Scheduler：检查 PowerX `backend/config/platform_capabilities/*.yaml` 是否已为 `com.corex.scheduler.jobs` 注册 REST protocols，并重新 seed/刷新权限目录。
6. `SCHEDULER_TENANT_MISMATCH`：host 调用不应传 `tenant_uuid`，检查插件是否已使用 framework host client，且 PowerX API key 解析出的租户是否正确。
7. `SCHEDULER_PLUGIN_OWNER_MISMATCH`：说明请求已进入 PowerX Scheduler 服务内部 owner 校验，先确认 PowerX 已按 ApiKey REST 权限链路适配 owner 授权。
8. 底座 SchedulerService 只存在 proto/capability 但不可调用：先补底座服务实现与 framework HostProvider，再接入业务插件。

## 9. 当前代码实现映射

| 能力 | 代码位置 | 说明 |
|---|---|---|
| framework host client | `framework/backend/go/runtime/scheduler/http_host_client.go` | 调 PowerX `/api/v1/admin/scheduler/jobs`，支持 `ApiKey`/`Bearer`，host 出站清空 `tenant_uuid` |
| provider 封装 | `framework/backend/go/runtime/scheduler/host_provider.go` | host provider 不 fallback tenant，动作接口不透传 tenant |
| DTO/校验 | `framework/backend/go/runtime/scheduler/types.go` | host create/update 不要求 `tenant_uuid`，local 仍要求租户 |
| skeleton runtime API | `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_job_handler.go` | `/api/v1/admin/runtime/scheduler/jobs` 系列入口，选择 local/host provider |
| 前端联调页 | `skeleton/web-admin/nuxt/app/pages/templates/framework-lab.vue` | “本地 Scheduler / 网关 Scheduler”创建、列表、触发、暂停、恢复 |
| 前端 API client | `skeleton/web-admin/nuxt/app/composables/api/useScheduler.ts` | 封装 scheduler runtime API |
| 事件声明 | `skeleton/plugin.d/events.yaml` | 声明 `powerx.runtime.scheduler.triggered.v1` |
| 回归测试 | `framework/backend/go/runtime/scheduler/http_host_client_test.go`、`skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_job_handler_test.go` | 覆盖 host 路径、鉴权、tenant 不透传 |

## 10. 变更记录

| 日期 | 变更 |
|---|---|
| 2026-05-23 | 对齐 PowerX Runtime Scheduler REST host 链路、ApiKey 权限前置条件、host 不传 `tenant_uuid`、到期通知验收口径 |
