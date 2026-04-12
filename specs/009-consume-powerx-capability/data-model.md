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
| token | string | Bearer Token（delegated 仅允许 `PX_PLUGIN_TOOL_TOKEN`）；需可轮换。 |
| tenant_uuid | string(UUID) | 绑定租户。 |
| plugin_id | string | 插件标识。 |
| expires_at | timestamp | 过期时间；框架需在接近过期时提示刷新。 |
| scopes | array | 获授权的能力范围。 |

## DelegatedGatewayEnvContract
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| PX_GATEWAY_BASE_URL | string(url) | 宿主注入的 Gateway 基础地址，delegated 必填。 |
| PX_PLUGIN_TOOL_TOKEN | string(jwt) | 宿主注入的短期 Bearer token，delegated 必填。 |
| PX_GATEWAY_AUTH_SCHEME | enum(`bearer`) | delegated 固定值，非 bearer 即配置错误。 |
| tid_claim | string(UUID) | 从 `PX_PLUGIN_TOOL_TOKEN` 解析出的租户标识，缺失即 `GW_TOKEN_INVALID_TID`。 |
| source | enum(`host_injected`) | delegated 模式唯一来源；不接受插件本地推断。 |

## GatewayConfigError
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| code | enum | `GW_CFG_MISSING_BASE_URL`、`GW_CFG_MISSING_TOOL_TOKEN`、`GW_CFG_INVALID_AUTH_SCHEME`、`GW_TOKEN_INVALID_TID`。 |
| message | string | 人类可读错误信息。 |
| details.required | array | 当前模式所需配置字段。 |
| details.present | array | 当前已检测到的配置字段（脱敏）。 |
| details.iam_mode | string | 当前 IAM 模式（`delegated/local`）。 |
| request_id | string | 请求追踪 ID。 |

## InvocationRequest
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| capability_id | string | 目标能力 ID。 |
| action | string | 具体动作（List/Create/Presign 等）。 |
| payload | object | 业务参数。 |
| tenant_uuid | string | 从上下文注入。 |
| request_id | string | 生成的 `X-Request-ID`。 |
| headers | map | 额外上下文：`Authorization`, `tenant_uuid`, `traceparent`。 |

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
