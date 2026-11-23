# 通知原创建凭据只读核对

状态：.NET Framework源码及隔离验证已实现。没有发布新版本或更新日常Core、服务凭据和授权。CRM调用链说明见CRM仓库的 `docs/guides/approval-reminder-reconciliation-acceptance.md`；Core权威schema为 `docs/contracts/notification-member-delivery.openapi.yaml`（1.4.0）。

## 契约与权限

`INotificationCreationReceiptReader.LookupAsync(NotificationScope, NotificationCreationLookupRequest)` 只接受原 `IdempotencyKey` 和预期 `MemberUuid`。Core租户及原服务owner由已验证凭据确定，业务body不能选择其他tenant/plugin、原通知UUID或自由HTTP路径。

正式能力 `com.corex.notifications.creation_receipts.read` 独立于创建和已读：固定 `core://notifications` / `INVOKE` / `operation=lookup_creation_receipt`，或固定服务REST `POST /notifications/creation-receipts/lookup`。显式API-key元数据为 `_scope.notifications.creation_receipts` / `read` / `capability` / `notifications`。拥有创建权限不等于有查询权限；仍要求已发布能力、租户注册及live STS/显式API-key grant。没有Admin权限代替或默认扩权。

返回 `notification_create_receipt.v1`：回显tenant、服务owner、原key、member，区分 `Found`、`NotificationState`、UTC `CheckedAt` 及原创建 `Notification` 快照。SDK验证完整上下文、owner类型、UUID、日期和found/notification状态一致性。原快照不代表当前已读/送达状态。

| 状态 | 含义 |
|---|---|
| `found=false, not_found` | 本服务、租户及原键范围内未查到创建凭据；不证明从未送达 |
| `found=true, present` | 原凭据和当前通知记录均存在 |
| `found=true, deleted` | 原凭据仍在，当前通知已软删除 |
| `found=true, receipt_only` | 原凭据仍在，当前通知记录不可见或已物理移除 |

历史接收人失效不删除原服务的创建凭据查询权，但不恢复接收人或审批权。原成员不匹配为明确拒绝；坏凭据、坏返回、失联、缺权限均为错误，不转换成not_found。

## Local/Host装配

Host由 `PowerXNotificationClient` 实现专用只读接口；只调用固定lookup端点。404/405/501返回 `FRAMEWORK_NOTIFICATION_CREATION_LOOKUP_UNSUPPORTED`，401/403及上游/格式失败保留相应错误。不会调用Publisher或Acknowledger，也不会切到Local。

Local通过 `LocalNotificationCreationReceiptReader` 包装专用 `ILocalNotificationCreationReceiptReader`，同样验证范围和返回值；不要求原有只提供Publisher的store自动承担查询。CRM已提供基于原outbox请求/通知快照的只读适配。新增runtime使用原通知的同一provider和非敏感连接绑定，不按HTTP输入切换模式。

## 验证与限制

通知相关SDK52项以及Framework全量312项通过；两项临时Core HTTP/PostgreSQL合同用例另行通过。覆盖固定路径、unknown/foreign/stale格式拒绝、原body/metadata精度、旧Host明确拒绝和无创建/已读兜底。CRM的临时Core5项包含响应丢失后原结果查询及取消后再次只读核对。

本次只覆盖.NET消费链，未宣称其他语言SDK已提供该接口。没有新Framework或Core业务schema，不修复旧receipt，不保存或恢复旧密钥，不允许在新连接自动重发。改变连接后如何找到原凭据、登记或重新发送属于独立后续契约，不能由该查询结果推导授权。
