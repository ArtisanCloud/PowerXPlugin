# PowerXPlugin Framework Scheduler 统一规划

本文定义插件侧与 PowerX 底座之间的统一 Scheduler 规划口径。当前仓库已新增 `framework/backend/go/runtime/scheduler` 初版 facade；现有 EventBridge/TaskBus 继续负责事件投递，Scheduler facade 负责统一“何时触发”的注册与管理契约。

## 1. 目标

1. 在 framework 中提供统一 Scheduler facade，使业务插件不直接感知本地调度或 PowerX 底座调度。
2. 支持插件 owner 维度的计划任务注册、更新、暂停、恢复、手动触发与查询。
3. 支持 `once`、`interval`、`cron` 三类调度。
4. 到点后统一发布调度触发事件，再由插件通过 EventBridge/TaskBus 消费执行。
5. 保持 AI Craft 等业务只依赖 framework 标准，不直接调用底座 scheduler，也不自行维护本地内存 timer。

## 2. 当前结论

1. backend framework 已有初版统一 scheduler facade：
   - 路径：`github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/scheduler`。
   - 已定义 `Scheduler` 接口、`JobSpec`、`Job`、`LocalProvider`、`HostProvider`、`DualProvider`。
   - `LocalProvider` 已能注册 job 并发布 `powerx.runtime.scheduler.triggered.v1`。
   - `HostProvider` 已预留可注入 `HostClient`，等待 PowerX 底座真实 SchedulerService SDK/路由确认后接线。
2. frontend framework 已有初版统一 scheduler client：
   - 路径：`framework/frontend/nuxt/framework-client/scheduler.ts`。
   - 已提供 `createSchedulerClient`、`SchedulerJobSpec`、`SchedulerJob`、`ListSchedulerJobsInput` 等类型。
   - Nuxt admin layer 已提供 `usePowerXScheduler` composable。
3. skeleton 已暴露管理端统一接口：
   - `GET/POST /api/v1/admin/runtime/scheduler/jobs`
   - `GET/PUT /api/v1/admin/runtime/scheduler/jobs/{jobId}`
   - `POST /api/v1/admin/runtime/scheduler/jobs/{jobId}/pause`
   - `POST /api/v1/admin/runtime/scheduler/jobs/{jobId}/resume`
   - `POST /api/v1/admin/runtime/scheduler/jobs/{jobId}/trigger`
4. PowerX 底座已有 scheduler 方向契约需要确认：
   - 目标服务为 `powerx.scheduler.v1.SchedulerService`。
   - 预期方法包括 `CreateJob`、`UpdateJob`、`PauseJob`、`ResumeJob`、`TriggerJob`、`GetJob`、`ListJobs`。
   - 需要确认这些 proto/capability 是否已有真实服务实现与路由注册，而不仅是生成记录。
5. PowerX 当前 `/admin/event-fabric/cron/jobs` 不应作为插件通用 scheduler：
   - 它更像底座内置运维接口。
   - 当前只适合固定系统 job，例如 `event_fabric.retry_dispatch`。
   - 不适合 AI Craft 的付款后提醒、交付前提醒、样品延期跟进等插件业务 schedule。

## 3. 三层职责

### 3.1 PowerX 底座

底座负责提供真实的 SchedulerService 实现，并承担生产环境调度可靠性：

1. 支持插件 owner：
   - `owner_type=plugin`
   - `owner_id=com.powerx.plugins.<plugin>`
2. 支持 schedule 类型：
   - `once`：一次性任务，使用 `run_at`。
   - `interval`：固定间隔任务，使用 `interval_seconds`。
   - `cron`：cron 表达式任务，使用 `cron_expr` + `timezone`。
3. 支持管理操作：
   - create/update/pause/resume/trigger/get/list。
4. 到点后触发事件：
   - 推荐发布 `powerx.runtime.scheduler.triggered.v1`。
   - 插件订阅事件后执行业务。
5. 负责生产环境能力：
   - 租户隔离。
   - 分布式锁或同等防重。
   - 至少一次触发。
   - 重试策略。
   - 审计与指标。

### 3.2 Framework

framework 新增包：

```text
github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/scheduler
```

统一接口建议：

```go
type Scheduler interface {
    CreateJob(ctx context.Context, job JobSpec) (*Job, error)
    UpdateJob(ctx context.Context, job JobSpec) (*Job, error)
    PauseJob(ctx context.Context, jobID string, tenantUUID string) error
    ResumeJob(ctx context.Context, jobID string, tenantUUID string) error
    TriggerJob(ctx context.Context, jobID string, tenantUUID string) error
    GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error)
    ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error)
}
```

核心结构建议：

```go
type JobSpec struct {
    TenantUUID     string
    OwnerType      string
    OwnerID        string
    Name           string
    ScheduleType   string // once | interval | cron
    ScheduleExpr   string // RFC3339 for once, seconds for interval, cron expr for cron
    Timezone       string
    Topic          string
    Payload        map[string]any
    IdempotencyKey string
    RetryPolicy    RetryPolicy
}
```

provider：

1. `local`：framework 内置本地实现，用于 standalone/local dev。建议使用 DB due-scan 或可注入 store，避免只依赖内存 timer。
2. `host`：调用 PowerX 底座 `powerx.scheduler.v1.SchedulerService`。
3. `dual`：迁移验证用，短期双写/比对，不作为长期默认。

### 3.3 业务插件

业务插件只依赖 framework scheduler：

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

AI Craft 不判断当前是本地 ticker 还是 PowerX host scheduler。

## 4. 标准触发事件

统一使用：

```text
powerx.runtime.scheduler.triggered.v1
```

说明：

1. 旧文档中的 `scheduler.job.triggered` 作为历史草案名称，不作为新实现推荐 topic。
2. 若底座已有强绑定 topic，需要在底座侧或 framework provider 中映射到 `powerx.runtime.scheduler.triggered.v1`。

payload 最小字段：

```json
{
  "job_id": "...",
  "job_name": "...",
  "owner_type": "plugin",
  "owner_id": "com.powerx.plugins.ai-craft",
  "tenant_uuid": "...",
  "trigger_source": "cron|once|interval|manual|retry",
  "scheduled_at": "...",
  "fired_at": "...",
  "trace_id": "...",
  "idempotency_key": "...",
  "business_action": "...",
  "payload": {}
}
```

## 5. AI Craft 适用场景

AI Craft 后半段业务建议全部走 durable scheduler job：

1. 付款后 50% 进度提醒。
2. 交付前提醒。
3. 样品延期检查。
4. 工厂催办。
5. 超时未确认补偿。

这些任务不应优先实现为 AI Craft 本地 scheduler。正确模型是：

1. 业务创建/更新订单时注册 `once` job。
2. scheduler 到点发布 `powerx.runtime.scheduler.triggered.v1`。
3. AI Craft 消费事件，根据 `business_action` 分发业务。
4. 业务处理使用 `idempotency_key` 防重。

## 6. 最小落地版本

1. 已完成：backend framework 新增 `runtime/scheduler` 包与公共类型。
2. 已完成：实现 `LocalProvider`，用于本地开发与 standalone 触发事件。
3. 已完成：实现 `HostProvider` 接口壳，支持注入底座 SchedulerService client。
4. 已完成：skeleton 当前调度触发 dispatcher 已复用 framework scheduler 的 topic 与 local provider。
5. 已完成：skeleton 管理端暴露 scheduler jobs CRUD/action API。
6. 已完成：frontend framework 新增 scheduler client 与 Nuxt composable。
7. 待完成：确认 PowerX 底座 `powerx.scheduler.v1.SchedulerService` 是否真实可调用。
8. 待完成：将 `HostProvider` 接到底座真实 SchedulerService SDK/路由。
9. 待完成：AI Craft 通过 framework 注册 job，并订阅 `powerx.runtime.scheduler.triggered.v1` 执行业务。

## 7. 底座待确认项

1. `powerx.scheduler.v1.SchedulerService` 的 proto、Go SDK、鉴权方式、服务地址与路由是否已发布。
2. 是否支持 `owner_type=plugin` 与 `owner_id` 查询隔离。
3. `once/interval/cron` 的字段命名与 timezone 规则。
4. 触发事件 topic 是否可统一为 `powerx.runtime.scheduler.triggered.v1`。
5. 调度失败重试、暂停、恢复与人工工单由底座负责到什么边界。
6. 插件侧是否通过 gRPC 直连、gateway proxy，或 capability invoke 调用 SchedulerService。

## 8. 非目标

1. 不把 `/admin/event-fabric/cron/jobs` 扩展为插件通用 scheduler。
2. 不让 AI Craft 先实现独立本地 scheduler 再迁移。
3. 不让业务代码直接判断 `POWERX_PROXY`。
4. 不让 Scheduler 直接承载业务执行；业务执行仍走 EventBridge/TaskBus。
