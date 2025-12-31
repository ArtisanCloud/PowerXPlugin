# Data Model — 008-framework-task-bus

## Entities

### Event

表示一次可发布与可消费的事件。

**Attributes**

- `topic`（string）：形如 `powerx.<domain>.<subdomain>.<action>.v<version>`
- `payload_version`（string）：例如 `v1`
- `occurred_at`（string, RFC3339）
- `meta`（map[string]string）：必须包含：
  - `tenant_uuid`
  - `request_id`
  - `source_plugin`
  - `trace_id`
  - `payload_version`
  - `occurred_at`
- `payload`（object）：按 topic 契约定义（见 `contracts/channel-events.yaml`）

**Validation rules**

- `tenant_uuid` 必填且必须为 UUID 字符串
- `topic` 必须包含 `.v<version>` 后缀
- `payload_version` 必须与 topic 版本一致（例如 `.v1` ↔ `payload_version=v1`）

### Subscription

表示对某一 topic 的订阅与处理策略。

**Attributes**

- `topic`（string）
- `handler`（string）：消费者标识
- `concurrency`（int）：并发度（默认 1）
- `retry_policy`（object）：重试次数/退避策略（由 TaskBus 或本地实现支持）
- `dlq_enabled`（bool）：是否启用死信

### EventBridgeConfig

控制事件桥接与运行策略。

**Attributes**

- `enabled`（bool）：是否启用 TaskBus 模式
- `dual_write_enabled`（bool）：是否双写（迁移期）
- `fallback_policy`（enum）：`auto_fallback_local`（默认）
- `permission_mode`（enum）：`least_privilege`（默认）

### IdempotencyKey

默认去重 key：

`topic + tenant_uuid + trace_id`

说明：

- 当 `trace_id` 缺失时只能“尽力而为”的幂等，需要告警/指标暴露链路缺失。

