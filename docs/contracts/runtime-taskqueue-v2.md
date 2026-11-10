# Runtime TaskQueue v2

这是 Queue 的新合同，需与提供 `core://runtime/taskqueue` 的 Core 版本一起发布。源码联调不代表当前发布版 v0.0.28 已具备这些字段或方法。

## 使用要求

- Queue 增加 `Renew`；Message 领取结果增加 `Lease{ReceiptUUID,Fence,ExpiresAt}`。Ack／Nack／Retry 必须使用本次领取的 receipt；Ack 显式提交 `Outcome=completed|failed`。
- `ValidateSubmission` 是 Host／Redis 和持久生产者共同使用的入队校验。消息 ID、租户、subscriber、topic、幂等键及原始信封必需；入队不可携带执行租约或 attempt。
- Host 使用配置的 API prefix 和原有 API Key／Bearer 注入，通过 `/tenant/capabilities/invoke` 分别调用 `com.corex.runtime.taskqueue.service_enqueue` 和 `com.corex.runtime.taskqueue.service_consume`。没有 Admin REST 或 Local 回退。
- Host 必须取得有效的 Core 响应信封、`completed`、`core_internal`、`fallback_used=false`、`taskqueue.v2` 和当前操作确认。空响应、错误信封、非法 JSON、错误协议或缺确认一律失败。HostError 保留 HTTP 状态、上游机器码及 trace／request ID。
- Local Redis 使用独立 v2 Stream、不可变双键索引、独立 receipt、递增 fence、有界重领／重试和 DLQ。Prefix 必须由部署方明确提供，区分环境和服务，不能包含 Redis hash-tag 大括号。旧 List／processing 不会自动迁入。
- `ProcessingExpiry` 现在控制领取租约，默认两分钟。消费者应按剩余租期的三分之一续约，同时对业务结果做数据库 CAS。续约失败必须取消执行；仅队列 fencing 无法保护任意外部副作用。
- `Retry` 结算当前投递，不能用它重新创建同编号消息。需要新业务执行时，由业务方明确创建新的幂等操作，并核对原执行结果。

## 交付边界

Core 的正式 OpenAPI 位于 PowerX `backend/docs/contracts/runtime-taskqueue-v2.openapi.yaml`。SDK 的双 provider 共用入队校验；与 Core 是合同级适配，不是跨仓库复制代码即视为统一。应用应先原子保存业务任务和不可变 outbox，再通过本 Queue 投递。

本批覆盖凭据刷新／前缀、严格响应、真实 Redis 的过期重领和旧 receipt 拒绝；AI Craft 通过独立 modfile 引用本目录回归。发布后才能升级应用正式 go.mod；未执行实际 Host grant 变更或历史队列切换。
