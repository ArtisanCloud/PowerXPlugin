# Data Model: Runtime 日志统一对齐

## Entities

### RuntimeLogRecord

- **trace_id**: string, required
- **task_id**: string, required (`unknown` allowed when missing context)
- **tenant_uuid**: string, required
- **tenant_key**: string, required (derived mirror from `tenant_uuid`)
- **subscriber_id**: string, required (`unknown` allowed when missing context)
- **topic**: string, required
- **status**: enum, required (`queued|processing|succeeded|failed|skipped`)
- **reason**: string, optional (required when `status=skipped` due to missing context)
- **plugin_id**: string, optional
- **component**: string, optional
- **gateway_auth_scheme**: string, optional
- **outbound_token_source**: string, optional
- **timestamp**: string(datetime), required

### FieldContract

- **contract_version**: string
- **required_fields**: string[]
- **status_enum**: string[]
- **compatibility_policy**: object
  - **tenant_field_primary**: string (`tenant_uuid`)
  - **tenant_key_mirror_required**: boolean (`true`)
  - **rollback_window_days**: integer (`7`)

### VerificationSuite

- **scope**: string (`critical_paths_only`)
- **path_groups**: array
  - `task.enqueue`
  - `task.consume`
  - `task.ack`
  - `task.fail`
  - `ws.publish`
  - `ws.dispatch`
- **min_events_per_group**: integer (`20`)
- **required_pass_rate**: number (`1.0`)

## Relationships

- `RuntimeLogRecord` 必须满足 `FieldContract.required_fields`。
- `RuntimeLogRecord.status` 必须属于 `FieldContract.status_enum`。
- `VerificationSuite.path_groups` 定义 `RuntimeLogRecord` 验收抽样边界。

## Validation Rules

- `tenant_uuid` 不能为空；`tenant_key` 必须由 `tenant_uuid` 派生。
- `status=skipped` 时必须带 `reason`，默认值为 `missing_context`。
- `task_id/subscriber_id` 若不可得，必须写 `unknown`，不得省略。
- 关键链路 6 类日志在验收样本中字段完整率必须达到 100%。
