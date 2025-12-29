# Data Model - PowerX 通用能力插件消费

## CapabilityRegistryEntry
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| capability_id | string | 全局唯一 ID（如 `com.corex.media.assets.manage`），由 Registry 分配；主键。 |
| source | enum(`corex`,`plugin`) | 能力来源；本特性聚焦 `corex`。 |
| description | string | 能力描述及场景。 |
| protocols | array | 可用协议列表（REST、gRPC 等），含路径、方法、版本。 |
| rate_limit | object | 每租户/每插件可配置的 QPS、突发值。 |
| quota | object | 额度配置（每日总调用次数/容量）。 |
| scopes | array | Tool Grant 所需 scope。 |

## ToolGrantToken
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| token | string | Bearer Token，本地保存于 env；需可轮换。 |
| tenant_uuid | string(UUID) | 绑定租户。 |
| plugin_id | string | 插件标识。 |
| expires_at | timestamp | 过期时间；框架需在接近过期时提示刷新。 |
| scopes | array | 获授权的能力范围。 |

## InvocationRequest
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| capability_id | string | 目标能力 ID。 |
| action | string | 具体动作（List/Create/Presign 等）。 |
| payload | object | 业务参数。 |
| tenant_uuid | string | 从上下文注入。 |
| request_id | string | 生成的 `X-Request-ID`。 |
| headers | map | 额外上下文：`Authorization`, `X-Tenant-UUID`, `traceparent`。 |

## InvocationResponseTelemetry
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| capability_id | string | 对应的能力。 |
| tenant_uuid | string | 对应租户。 |
| trace_id | string | Gateway 返回的 trace。 |
| duration_ms | number | 调用耗时。 |
| status | enum(`success`,`rate_limited`,`unauthorized`,`error`) | 结果。 |
| error_code | string | 当 status ≠ success 时的错误码。 |
| timestamp | timestamp | 完成时间。

## MockCapabilityAdapter
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| capability_id | string | Mock 对应能力。 |
| mode | enum(`record`,`replay`,`static`) | Mock 策略。 |
| payload_template | object | 静态响应模板或录制副本。 |
| last_sync | timestamp | 最近一次从真实 Gateway 拉取数据的时间。 |
| enabled | bool | 是否启用（由 `--use-mock=<module>` 控制）。 |
