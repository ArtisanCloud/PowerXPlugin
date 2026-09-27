# Framework Customer Identity/Auth

2026-09-08 migration: `NewDelegatedCustomerAuthClient` and its legacy
`/customer/auth/*` transport have been removed. Construct `NewDelegatedCoreAuthClient`
with a service token provider, inject it into `NewRuntime`, and consume `Auth()`
or `AuthClient()`. Only the explicitly supported Core channel is available;
local authentication remains plugin-supplied. Missing adapters never fall back.

`customerfw` provides generic C-end external customer identity and authorization primitives for plugin mini-app routes.

The `CustomerRuntime` identity/auth boundary is intentionally limited to:

- validating customer tokens
- attaching a normalized `CustomerContext`
- resolving tenant context for customer requests
- enforcing customer-to-tenant membership
- defining bootstrap and delegated auth client contracts
- providing diagnostics fields that avoid raw token or secret values

The separate `AccountSelectorClient` in this package lists and creates basic customer account records through service-scoped Core capabilities; it is not part of `CustomerRuntime`, and this package does not provide a generic local Account Store. Neither contract models SCRM or industry concepts such as customer profiles, tags, owners, follow-ups, timelines, players, guardians, learners, patients, fans, benefits, training plans, or reports. Those remain plugin domain models and should be exposed through plugin capabilities such as SCRM when other plugins need them.

It does carry generic PowerX Core customer display attributes through `CustomerContext.Profile`: `display_name`, `nickname`, `given_name`, `family_name`, `avatar_url`, `locale`, and `timezone`. Those fields are the base customer identity shape, not an SCRM or industry domain model.

## 双模式边界

`customerfw` 的调用方不得自行选择本地表或 Core。目标架构由 Framework Runtime Factory 根据可信 provider mode 选择 adapter：local 模式由插件注入 `LocalCustomerStore` contract 实现，delegated 模式由 Framework Core adapter 调用正式 Host Contract。两种模式都必须输出相同的 `CustomerContext`；缺少 local adapter 或 Core capability 时明确失败，不能 fallback。

当前外部身份解析的 Core capability 只返回 customer 与 membership UUID、显示名。已发布的 `com.corex.customer.memberships.delegated_read` 则通过双凭证合同返回当前 customer 在当前 tenant 的完整 membership（roles/scopes）。详细规范见 [Framework 双模式业务模块规范](../../../../../docs/guides/develop/framework-dual-mode-business-modules.md)。

`Runtime` 按操作解析 adapter：`Auth()`、`ExternalIdentity()` 与 `Membership()` 分别返回所选模式的 contract 或明确的 `FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE`。因此缺少 delegated membership contract 不会使登录 adapter 被 local fallback 替代；只有实际要求 membership 的路由会被拒绝。`ExternalIdentityResolver` 不接受或传递 `tenant_uuid`，Core 必须从 STS service credential 推导租户范围。

Delegated membership adapter 使用 `Authorization: Bearer <plugin STS>` 和 `X-PowerX-Customer-Authorization: Bearer <customer JWT>` 调用 Core；三个 UUID 仅来自 Core 回包，不能由插件作为请求参数传递。缺少该 capability/grant 或 customer credential 必须明确失败，绝不能读取插件本地 `customer_*` 表。`NewUnavailableDelegatedMembershipResolver()` 只保留给测试或显式未配置的启动失败路径。

Boundary rule:

- Framework IAM identifies back-office members/employees.
- `customerfw` identifies C-end external customers and their tenant membership.
- SCRM plugins model customer business data, tags, lifecycle, follow-up records, and member-customer business relationships.
- Other plugins call SCRM capabilities for customer profiles, tags, owners, follow-ups, lifecycle, leads, or timelines.

Typical usage:

```go
protected := group.Group(
	"",
	customerfw.Authenticate(validator, customerfw.RequireTenant()),
	customerfw.RequireMembership(resolver),
	tenantfw.EnsureTenant(),
)
```

Handlers should read only the framework context:

```go
cc := customerfw.MustContextFromGin(c)
tenantUUID := cc.TenantUUID
customerUUID := cc.CustomerUUID
```

Testing helpers:

```go
validator := customerfw.NewMockCustomerValidator(&customerfw.CustomerContext{
	TenantUUID:    "tenant-a",
	CustomerUUID:  "customer-a",
	Authenticated: true,
})
token := customerfw.TestToken("customer-a", "tenant-a")
ctx := customerfw.WithCustomerContext(context.Background(), &customerfw.CustomerContext{
	TenantUUID:    "tenant-a",
	CustomerUUID:  "customer-a",
	Authenticated: true,
})
```

Developer-facing docs:

- `docs/guides/develop/auth/customer.md`
- `docs/guides/features/009-consume-powerx-capability/usecase-customer-contact.md` describes the separate account selector and Contact runtime; this Auth runtime is not an Account Store.
- `docs/contracts/customer-auth.openapi.yaml`

### 客户默认类型与主联系人（2026-09-26）

`CreateBasicAccount` 未指定或空 `Type` 默认 `person`；明确 `company` 保留且创建须提供自然人主联系人。Delegated 由 Core 在客户写入或已验证身份解析事务中补存历史空类型、复用唯一有效 Contact 或为个人创建 Contact，并返回稳定 UUID；Framework 不在插件本地补写。外部身份结果接受 person/company，不得将 company 改成 person。失效引用、多有效联系人歧义须失败。客户/联系人查询保持只读。Local 持久化适配器应由插件在自己的客户写入事务落实相同规则。

### 客户外部身份管理（2026-09-27）

`NewExternalIdentityClient(invoker)` 提供 `Lookup`、`ListByCustomer`、`Bind`、`CreateAndBind`，实现 `ExternalIdentityStore`。读写分别调用 Core 的 `customer.external_identities.service_read/service_manage`，固定 `core://customer/external-identities`，不转用 Admin 接口、不回退本地。Lookup found=false 明确表示当前范围未关联，不会自动调用 Resolve。Local adapter 应实现同接口和相同事务、幂等与只读语义。

API Key 必须有精确 grant 和 Core 管理员绑定的唯一 plugin_id；STS 必须有实例 grant。provider_subject 包含渠道实例；Shopify 与登录一致使用店铺域名和完整 GID。`NormalizeBasicAccountLabels` 允许仅邮箱输入成为展示标签，不填充真实姓、名。Core 完整交接见其 `docs/contracts/customer-external-identity-management.md`。
