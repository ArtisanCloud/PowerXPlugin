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

## HostCredential
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| mode | enum(`sts_service`,`gateway_api_key`) | 安装态使用 STS；Skeleton 开发调试可使用明确配置的 API Key。 |
| token | string | 不可写入调试响应或前端状态的凭证。 |
| tenant_uuid | string(UUID) | 仅由 Core 从凭证推导，不是调用方可传字段。 |
| plugin_id | string | 插件标识。 |
| expires_at | timestamp | 过期时间；框架需在接近过期时提示刷新。 |
| scopes | array | 获授权的能力范围。 |

## DelegatedHostContract
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| PX_GATEWAY_BASE_URL | string(url) | 宿主注入的 Gateway 基础地址，delegated 必填。 |
| sts_token_provider | interface | Framework 注入的短期 service token 提供器，delegated 必填。 |
| tenant_uuid | string(UUID) | Core 根据 STS service actor 推导；不接受插件传入。 |
| source | enum(`host_injected`) | delegated 模式唯一来源；不接受插件本地推断或 API Key fallback。 |

## GatewayConfigError
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| code | enum | `GW_CFG_MISSING_BASE_URL`、`GW_CFG_MISSING_TOOL_TOKEN`、`GW_CFG_INVALID_AUTH_SCHEME`、`GW_TOKEN_INVALID_TID`。 |
| message | string | 人类可读错误信息。 |
| details.required | array | 当前模式所需配置字段。 |
| details.present | array | 当前已检测到的配置字段（脱敏）。 |
| details.provider_mode | string | 当前 IAM 模式（`delegated/local`）。 |
| request_id | string | 请求追踪 ID。 |

## InvocationRequest
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| capability_id | string | 目标能力 ID。 |
| action | string | 具体动作（List/Create/Presign 等）。 |
| payload | object | 业务参数。 |
| request_id | string | 生成的 `X-Request-ID`。 |
| headers | map | 额外上下文：`Authorization`, `traceparent`；不包含 tenant header。 |

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

## MockCapabilityAdapter（历史，不得用于正式 Host Contract）

该历史通用 Gateway 辅助模型不属于当前 Framework/Skeleton Host Contract 实现范围。认证、授权、对象不存在和上游故障必须返回结构化失败，不能用 Mock 或本地数据伪造成功。

## HostContractProbe
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| module | string | 固定模块键，如 `iam`、`knowledge`、`media`。 |
| operation | string | allowlist 中的强类型操作。 |
| provider_mode | enum(`local`,`delegated`) | 当前 Framework provider 模式。 |
| capability_id | string | 本次操作所需 Core capability。 |
| trace_id | string | Core/Framework 返回的 trace；不可用时为空。 |
| reason_code | string | 失败时的稳定机器码；成功时为空。 |
| result | object | 经 Framework DTO 映射的结构化结果，不含 credential。 |
| observed_at | timestamp | probe 完成时间。 |

> 2026-09-03 起，`HostContractProbe` 是新调试台的权威结果模型。`MockCapabilityAdapter` 仅保留为历史通用 capability 开发工具；不得用于 Host Contract Lab 的自动降级。
