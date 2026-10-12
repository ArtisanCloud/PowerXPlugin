# Contact 主联系人合同对齐

依据 Core `specs/030-customer-contact/contracts/contact-core-internal.md` 的 2026-10-11 扩展，以及 `docs/contracts/customer-contact-maintenance.openapi.yaml`。

## Framework 接口

Go 新增独立 `contactfw.PrimaryContactManager` 和 `SetPrimaryContactInput` / `PrimaryContactResult`。原六项 `Store` 不扩展；消费方从启动选定的 `Runtime.PrimaryContactManager()` 获取接口。Local 注入 `WithPrimaryContactManagers`，Delegated 的正式客户端绑定同一接口。缺适配器明确失败。

固定 capability 为 `com.corex.customer.contacts.primary_manage`，协议 `core_internal`，`INVOKE core://customer/contacts`，operation 为 `set_primary`。旧 `service_read` / `service_manage` 授权不能设主。请求仅有客户 UUID、目标联系人 UUID、预期旧主引用；tenant、actor、endpoint、headers 和代理参数均不可覆盖。

Go 的预期旧引用为 `*string`：nil 拒绝，非 nil 空字符串明确表示当前无主引用。Core 原始返回为 invocation `data.payload.payload` 下的三个 UUID 字段；客户端核对客户、目标及前一个引用。拒绝错误归属、意外 fallback 或协议变更。

消费方在“更换主联系人”产品动作中调用，旧引用来自当前客户回读，不能用任意空值绕过并发检查：

```go
manager, err := contactRuntime.PrimaryContactManager()
if err != nil {
    return err
}
expected := customer.PrimaryContactUUID
result, err := manager.SetPrimary(ctx, contactfw.SetPrimaryContactInput{
    CustomerUUID: customer.CustomerUUID,
    ContactUUID: selected.ContactUUID,
    ExpectedPrimaryContactUUID: &expected,
})
```

遇到 `CONTACT_PRIMARY_CHANGED` 时重新读取客户和联系人，让操作者再次确认。此接口不是订单导入时选择联系人的前置步骤。

## 业务错误和字段校验

保留 `CONTACT_PRIMARY_EXPECTED_REQUIRED`、`CONTACT_PRIMARY_CHANGED`、`CONTACT_PRIMARY_NOT_ACTIVE`、`CONTACT_PRIMARY_OPERATION_REQUIRED`、`CONTACT_PRIMARY_CHANGE_REQUIRED`、`CONTACT_PRIMARY_OPERATION_FAILED` 和 `CONTACT_REQUEST_INVALID`。从 Core 正式 `error_code` / `details.trace_id` / `request_id` 结构读取；保留实际 HTTP 状态及原始 cause，不把业务冲突归为委托服务不可用。

Customer/Contact 姓名使用 Unicode codepoint 上限 128，邮箱 255、电话 32；拒绝控制字符和非法编码。Go 检查 UTF-8。tag 的 ASCII/byte 规则及 customer external identity subject 的协议 byte 上限保持其独立合同。

普通 Contact 创建不能携带 `primary`。通用编辑无法更换主引用；当前主联系人停用或 temporary 必须先显式更换主引用。客户创建仍原子创建主联系人，不需要追加设主调用。

## Local 实现

Skeleton 的 `FrameworkPrimaryContactManager` 从可信 Context 获取租户，服务 principal 由 bootstrap 配置。通过 Repository 在同一事务内锁定 membership，检查 active 客户/目标和预期旧引用，维护唯一 primary 角色、membership 主引用及审计。普通 Contact 更新获取同一锁。

Local 客户列表、详情和编辑结果从 membership 读取主引用；不读取 account 上的旧缓存。缺少 membership 明确返回 `CUSTOMER_ACCOUNT_REFERENCE_UNAVAILABLE`，已有 Local 数据需按现有客户创建/迁移合同补齐 membership，不做静默兜底。

SQLite 共享现有本地写锁；PostgreSQL 使用持久行锁。审计失败回滚全部变化；没有写入变化时不追加审计。本轮仅实现 Go；C# 由 CRM 团队按 Core 同一合同独立对齐。

## 验证与版本

定向 Go 测试覆盖正式请求/响应、独立授权、旧引用冲突、错误状态/Trace、Unicode 边界、非法覆盖和模式隔离。Skeleton 测试覆盖跨租户/客户、active 约束、普通编辑保护、审计回滚及设主/停用并发。

Go v0.0.24 已发布且不可覆盖；本次准备 v0.0.25。CLI/Skeleton 依赖相应更新。CI 使用 `PX_TEST_FRAMEWORK_SNAPSHOT=1` 把当前 Framework 源码打成不可变 Go module proxy ZIP，生成后端仍以 `GOWORK=off`、无本地 `replace` 编译。不是已发布版本的下载验收。发布 v0.0.25 后，运行不带快照开关的生成后端检查，验证真实发布依赖。

## Core 仍须配合的部分

只读核对发现 Core `internal/transport/http/openapi/capability_registry/tenant_handler.go` 的 `selectInvokeErrorTemplate` 尚未把 `CONTACT_*` 加入业务 HTTP 状态透传分支。当前服务平面可能把 `CONTACT_PRIMARY_CHANGED` 包成 HTTP 502；Framework 保留此实际状态，不能伪称 409。

Core 需保留 Contact 业务错误的原始 HTTP 状态，并补真实 API Key/STS 设主及普通更新的回归：409 冲突、400 非法参数、500 事务失败、业务码、request/Trace 一致。目标实例须加载对应代码和迁移，发布/注册 `primary_manage`，对实际插件凭据授予精确权限并确认 granted。发布能力不会自动授权。API Key 对应 scope 为 `_scope.customer.contacts.primary_manage`、action 为 `manage`、resource type 为 `capability`、pattern 为 `customer_contacts_primary_manage`；STS 按同一 capability 的实例授权执行。

本轮证据仅为本地合同/事务测试与源码核对；不代表正式 Core 凭据已获授权，也不代表 PostgreSQL 实例联测通过。

本次不修改 Core，不执行其迁移、授权、重启或真实设主；不复制 Core CSV/补录 UI，也不修改 AI Craft 的订单联系人选择。
