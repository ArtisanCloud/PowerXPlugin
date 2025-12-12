# Data Model — 插件能力注册与暴露治理闭环

## CapabilityRecord
- **Fields**: `id` (string, global unique), `name`, `description`, `version`, `sensitivity`, `input_schema_path`, `output_schema_path`, `protocols` (map), `status` (`draft|under_review|approved|exposure_pending|exposed|deprecated`), `audited_at`, `tenant_scope`, `tags`.
- **Relationships**: 1→N `ReviewTask`, 1→N `ExposurePackage`, 1→N `LifecyclePlan`, 1→1 `CapabilityCatalog` entry.
- **Rules**: ID 为 `reverse-domain`；版本遵循 SemVer；敏感度驱动审核流；Schema 路径必须存在于 `contracts/schema/**`。

## ReviewTask
- **Fields**: `task_id`, `capability_id`, `role` (`security|compliance|ops`), `assignees`, `sla_due_at`, `status`, `comments`, `attachments`, `risk_score`。
- **Relationships**: N→1 `CapabilityRecord`；可引用 `CapabilityCatalog` 中的敏感标签。
- **Rules**: 高敏任务需双人复核；SLA 超时触发升级事件。

## ExposurePackage
- **Fields**: `capability_id`, `channels` (list of channel configs), `auth_strategy`, `rate_limit`, `quota`, `tenants` (list), `docs_bundle_path`, `sdk_bundle_path`, `status`。
- **Relationships**: N→1 `CapabilityRecord`; 1→N `TenantSubscription`。
- **Rules**: 每个通道必须指定协议资产（OpenAPI path、Proto service、Workflow template 等）；状态只允许单向推进（pending→syncing→active→disabled）。

## TenantSubscription
- **Fields**: `tenant_uuid`, `capability_id`, `quota`, `usage`, `status`, `notifications`, `last_notified_at`。
- **Relationships**: N→1 `ExposurePackage`。
- **Rules**: `tenant_uuid` 必须从宿主 IAM 拉取；额度变更需要审计。

## LifecyclePlan
- **Fields**: `capability_id`, `plan_id`, `change_type` (`upgrade|deprecation|rollback`), `diff_summary`, `grace_period`, `notification_channels`, `dual_run_until`, `rollback_plan`, `status`。
- **Relationships**: N→1 `CapabilityRecord`; N→N `TenantSubscription`（受影响租户列表）。
- **Rules**: 通知覆盖率需=100%；当抽样监控异常时允许暂停或回滚。

## CapabilityCatalog
- **Fields**: `plugin_id`, `manifest_version`, `imports` (list), `entries` (array of embedded capability descriptors), `checksum`, `generated_at`。
- **Relationships**: 1→N `CapabilityRecord`（通过 `entries[*].id]`），1→1 `CapabilitiesManager`。
- **Rules**: `checksum` 用于检测目录变化；`imports` 中的文件必须存在且唯一。

## CapabilitiesManager (runtime object)
- **Fields**: `version`, `supported_protocols`, `exported_assets` (paths), `sync_status`, `last_sync_at`, `errors`。
- **Relationships**: 1→1 `CapabilityCatalog`; 1→N `IntegrationSync`（PowerX `capability_registry`、Workflow Builder、Agent Hub）。
- **Rules**: `sync_status` 允许 `idle|syncing|failed|blocked`; 若失败需触发回滚；必须暴露 API `ListCapabilities`, `ExportProtocols`, `RegisterWithHost`。
