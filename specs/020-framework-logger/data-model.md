# Data Model: Framework 统一日志适配

## 1. Log Policy

- **Purpose**: 描述 PowerX 下发并由 framework 消费的日志策略。  
- **Key Fields**:
  - `policy_version`
  - `mode` (`host|standalone`)
  - `sinks[]` (`stdout|file|loki`)
  - `format` (`json|text`)
  - `level` (`debug|info|warn|error`)
  - `authorized_extra_sinks[]`
  - `retry` (`enabled`, `max_attempts`, `backoff_ms`)
  - `updated_at`
- **Rules**:
  - 宿主模式默认必须包含 `stdout`。
  - 若启用 `file/loki`，必须出现在授权列表。
  - 未识别配置应回退到安全默认值并产生日志告警。

## 2. Log Event

- **Purpose**: 插件业务侧输出的标准化日志事件。  
- **Key Fields**:
  - `timestamp_utc`
  - `biz_date`
  - `biz_tz`
  - `level`
  - `message`
  - `plugin_id`
  - `tenant_uuid`
  - `component`
  - `trace_id`
  - `event`
  - `fields` (扩展字段对象)
- **Rules**:
  - 固定标签字段仅允许 `plugin_id, tenant_uuid, component, level`。
  - 高基数字段必须放在 `fields`，不得升级为标签。
  - 缺失 `trace_id` 时需自动补齐并标记来源。

## 3. Sink Route Outcome

- **Purpose**: 描述单条日志在每个 sink 的路由执行结果。  
- **Key Fields**:
  - `trace_id`
  - `sink`
  - `status` (`success|failed|retrying|dropped`)
  - `attempt`
  - `error_code`
  - `error_message`
  - `occurred_at`
- **Rules**:
  - 任一 sink 失败不得阻断其他 sink。
  - 失败 sink 必须留下可观测结果，并按策略重试。

## 4. Legacy Logging Violation

- **Purpose**: 记录直写日志违规点与治理状态。  
- **Key Fields**:
  - `violation_id`
  - `module_path`
  - `symbol`
  - `violation_type` (`direct_logrus|direct_zap|direct_file`)
  - `status` (`detected|warned|blocked|resolved`)
  - `deadline_version`
  - `detected_at`
  - `updated_at`
- **Rules**:
  - 截止版本前允许 `warned`，截止版本后未修复项必须进入 `blocked`。
  - 新增代码出现违规时直接进入 `blocked`（不享受历史豁免）。

## Relationships

- Log Policy 1:N Sink Route Outcome（按策略驱动路由结果）
- Log Event 1:N Sink Route Outcome（每条日志在多个 sink 有独立结果）
- Legacy Logging Violation N:1 Log Policy（治理策略决定违规处理阶段）
