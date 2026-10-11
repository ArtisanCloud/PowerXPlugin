# Local Scheduler execution

`AddPowerXScheduler(ProviderMode.Local, ...)` assembles the persistent store, one
local EventBridge, `LocalScheduler`, and the background runner. Inject the same
`IEventEmitter` when constructing a scheduler with an application-owned store.

`TriggerJobAsync` dispatches the standard Scheduler event immediately and waits
for its consumers and the completed snapshot write before returning. It does
not wait for the background scan. Missing jobs return false; inactive jobs fail
with `SCHEDULER_JOB_NOT_ACTIVE`; dispatch failure returns
`SCHEDULER_UPSTREAM_DEPENDENCY`. Cancellation propagates.

The manual due time is saved before dispatch. Failure leaves the occurrence due
with its original idempotency key for retry. Consumers must persist their own
idempotency receipts alongside business changes: a post-dispatch storage failure
can lead to redelivery. An already-due manual retry does not reset the key.

Once jobs complete on successful manual execution, consuming their scheduled
occurrence. Interval and cron jobs retain active status with a new next run time.
The background runner scans scheduled jobs once a minute and uses the same
dispatch/completion path. Execution gates are scoped by tenant and job in one
`LocalScheduler` instance, with revision checks after acquiring the gate. This
prevents concurrent manual and scheduled dispatch in that runtime; it is not a
distributed lease across application instances.

`ILocalSchedulerDiagnostics.TriggerJobPreviewAsync` dispatches a separate manual
preview immediately, retaining the job status and scheduled occurrence. Active
and completed jobs can be previewed repeatedly; paused jobs fail explicitly.
Each preview has its own idempotency key/trace; `Job.LastExecutionId` records the
last successful execution. Preview failures do not consume the schedule or
create retry jobs. Applications must restrict this operation to diagnostic jobs.

`ILocalSchedulerDiagnostics.DeleteJobAsync` takes the confirmed job UUID and
revision, uses the execution gate, and delegates to the store's atomic deletion.
CRM implements this as soft deletion; the explicit in-memory test store removes
the entry. Missing jobs return false; stale UUID/revision fails with a conflict.
Applications own deletion authorization, sample classification, preview and
confirmation, and any recovery workflow. Due scans reload snapshots and cannot
dispatch a deleted job.

Event dispatch success is separate from notification receipt. WS delivery may
have no connected subscribers; local WS notifications have no offline replay.
Delegated Scheduler selection and Host failure behavior remain unchanged.

## Core HTTP management

### 服务态读取与清理（2026-10-08 Core 合同）

`PowerXSchedulerClient` 实现独立可选 `ISchedulerCleanup`，不改变 `IScheduler` 必选方法。
`DeleteJobAsync(SchedulerDeleteRequest)` 提交确认的 job UUID 和 positive uint64 revision，
返回 `powerx.scheduler.job-deletion/v1` typed receipt，保留 already_deleted/history/trace。
不自动刷新版本、重试删除或回退 Local；409、403、Core reason_code 与 request/trace 原样保留。

新读取和清理在 API Key Host simulation 下使用 `/tenant/scheduler/jobs` REST；
Bearer/STS 使用 `/tenant/invocations`，固定 `core_internal`、`INVOKE`、
`core://scheduler/jobs` 和 typed `list_jobs/get_job/list_runs/delete_job`。
不携带浏览器 tenant、bearer、自由 headers/query/raw proxy，不回退旧 Admin 读取。
service_read/service_delete 及精确 plugin owner grant 均由 Core 实时授权。
应用负责样例筛选、完整分页预览、准确确认集合、管理员校验及批量部分失败反馈。

服务 DTO 使用 job_uuid/payload/revision，映射到 Framework Job；该模型的 revision 仍为
long，超范围读取明确失败；删除请求和回执版本使用 ulong。`GetCleanupJobAsync` 可显式
读取墓碑，`ListRunsAsync` 保留删除后的授权历史读取。仅 Get 的真实 HTTP 404 返回 null。

既有创建/修改/暂停/恢复/触发仍走 Core 保留的 Admin 写入口。新合同只定义服务态
读取和删除，不在 Framework 构造不存在的 typed 创建/触发操作。

Existing write responses use Core's `code/data` envelopes and map the authoritative
`uuid` to both Framework UUID and job ID. Core's `payload_json` becomes the typed
Framework payload. Lists follow Core pagination, and job actions use UUID paths.
Only `GetJobAsync` treats HTTP 404 as a missing object; an action returning 404
fails explicitly. Caller tenant parameters and administrator bearer tokens are
not forwarded; the configured service credential determines Core scope.

`TriggerWithRunAsync` and `ListRunsAsync` expose typed Core run records. A
`triggered` run records scheduler triggering, not downstream business consumption
or notification receipt. Consumers must keep these acceptance boundaries separate.
Applications may register an independent Host diagnostics client alongside the
local business runtime without replacing its startup-selected scheduler.
