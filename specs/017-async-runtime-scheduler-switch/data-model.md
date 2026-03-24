# Data Model: Async Runtime Scheduler 模式切换

## Entities

### RuntimeModeProfile

- **mode_signal**: string, required（`POWERX_PROXY` 解析结果）
- **execution_mode**: enum, required（`standalone_local` | `delegated_proxy`）
- **taskbus_provider**: enum, required（`redis` | `host`）
- **gateway_auth_scheme**: enum, optional（`bearer` | `apikey`）
- **is_valid**: boolean, required
- **validation_error**: string, optional（冲突时必填）

### SchedulerTrigger

- **trigger_id**: string, required
- **trigger_source**: enum, required（`manual` | `cron`）
- **topic**: string, required（`_topic.*`）
- **payload**: object, required
- **trace_id**: string, required
- **created_at**: datetime, required

### DispatchExecution

- **dispatch_id**: string, required
- **trigger_id**: string, required
- **execution_path**: enum, required（`local_path` | `proxy_path`）
- **status**: enum, required（`queued` | `processing` | `succeeded` | `failed` | `paused`）
- **result_reason**: string, optional
- **finished_at**: datetime, optional

### RetryPolicyRecord

- **dispatch_id**: string, required
- **max_attempts**: integer, required
- **current_attempt**: integer, required
- **last_error_code**: string, optional
- **last_error_message**: string, optional
- **exhausted**: boolean, required

### RecoveryTicket

- **ticket_id**: string, required
- **dispatch_id**: string, required
- **ticket_status**: enum, required（`open` | `processing` | `resolved`）
- **paused_job_id**: string, required
- **resume_role_required**: enum, required（`ops_admin_only`）
- **resolved_by**: string, optional
- **resolved_at**: datetime, optional

### ValidationEvidence

- **evidence_id**: string, required
- **mode**: enum, required（`standalone_local` | `delegated_proxy`）
- **scenario_type**: enum, required（`manual_trigger` | `cron_trigger` | `permission_failure` | `config_conflict`）
- **passed**: boolean, required
- **recorded_at**: datetime, required
- **notes**: string, optional

## Relationships

- `RuntimeModeProfile` 决定 `DispatchExecution.execution_path`。
- `SchedulerTrigger` 触发 `DispatchExecution`。
- `DispatchExecution` 失败时可产生一个 `RetryPolicyRecord`。
- `RetryPolicyRecord.exhausted=true` 时必须生成 `RecoveryTicket` 并令任务进入 `paused`。
- `ValidationEvidence` 对应每次模式验收场景，验证状态语义与结果一致性。

## Validation Rules

- `RuntimeModeProfile.is_valid=false` 时，系统必须 fail-fast，不创建调度触发。
- `trigger_source` 为 `manual` 与 `cron` 时，`topic/payload` 语义必须一致。
- `delegated_proxy` 权限失败必须进入有限重试流程，禁止直接丢弃。
- 重试超限必须创建工单并暂停任务，恢复角色限制为运维/管理员。
- “结果语义一致”判定规则：状态流转与业务结果一致；执行耗时不纳入不一致判定。
