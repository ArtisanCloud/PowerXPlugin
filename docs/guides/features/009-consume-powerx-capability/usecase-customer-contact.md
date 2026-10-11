# 场景：Customer 基础资料与 Contact Runtime

从[Framework 业务模块接入指南](guide.md)进入本文。本文描述插件如何选择客户、创建/更新基础客户资料，以及在客户名下管理联系人。C 端登录、Customer JWT 与 membership 鉴权仍按 [Customer Auth 指南](../../develop/auth/customer.md)接入。

## 1. 选择正确的合同

| 需求 | Framework 入口 | local | delegated |
|---|---|---|---|
| C 端注册、登录、验证、membership | `customerfw.NewRuntime` 的 `Auth()`、`ExternalIdentity()`、`Membership()` | 插件注入 `LocalCustomerStore` 或独立 adapters | Core Auth、External Identity、Membership 客户端；部分操作需要独立的 customer JWT |
| 选择客户、创建/更新基础客户资料 | `customerfw.NewAccountSelectorClient` 的 `ListAccounts`、`CreateBasicAccount`、`UpdateBasicAccount` | 插件实现自己的客户存储/服务；Skeleton 有本地示例，Framework **尚无通用 Account Runtime/LocalStore** | 固定的 Core service capability；由凭证确定 tenant |
| 客户名下的联系人及渠道身份 | `contactfw.NewRuntime(...).Store()` | 插件实现 `contactfw.LocalStore` | `contactfw.NewCapabilityClient` |

基础客户创建只建立客户资料及当前租户的归属，不创建登录密码或登录身份。联系人是独立的自然人记录，通过 `customer_uuid` 归属客户；`external_subject` 不是 `contact_uuid`。行业标签、跟进、负责人等 SCRM 业务不属于此合同。切换部署模式不会自动搬迁两侧数据，也不保证两侧 UUID 相同。

## 2. 装配与调用 Contact

在插件 bootstrap 中使用**可信启动配置**选定 `provider.ModeLocal` 或 `provider.ModeDelegated`。local 注入真实 `contactfw.LocalStore`，其六个方法是 `Create`、`Get`、`Update`、`ListByCustomer`、`ResolveIdentity`、`BindIdentity`。delegated 使用已配置的服务出站 Gateway invoker：本地联调可以使用已授权 API Key；安装到 PowerX 后使用宿主 STS。业务层只持有 `contactfw.Store`，不读取 mode，也不自行拼接 Gateway 请求。

```go
func BuildContactStore(
    mode provider.Mode,
    local contactfw.LocalStore,
    invoker contactfw.CapabilityGatewayInvoker,
) (contactfw.Store, error) {
    var delegated contactfw.DelegatedContactClient
    if mode == provider.ModeDelegated {
        client, err := contactfw.NewCapabilityClient(contactfw.CapabilityClientConfig{Invoker: invoker})
        if err != nil { return nil, err }
        delegated = client
    }
    runtime, err := contactfw.NewRuntime(mode, local, delegated)
    if err != nil { return nil, err }
    if err := runtime.ValidateRequired(); err != nil { return nil, err }
    return runtime.Store()
}
```

上述片段需导入 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw` 和 `.../runtime/provider`；`invoker` 应由插件已有的可信 Gateway 装配提供。local Store 从已验证的请求上下文取得 tenant/actor，并校验客户、联系人及渠道身份属于同一租户；不得信任请求中的 tenant 覆盖。Skeleton 的参考实现见 [`NewFrameworkContactLocalStore`](../../../../skeleton/backend/go-gin/internal/services/customer/framework_contact_local_store.go) 与[启动装配](../../../../skeleton/backend/go-gin/cmd/plugin/main.go)。`IdentityChannelMigrator` 是单独的 local 运维修复接口，不属于普通 Store，也没有 delegated 版本。

取得 Store 后，可按客户 UUID 分页读取并创建联系人：

```go
page, err := contacts.ListByCustomer(ctx, contactfw.ListByCustomerInput{
    CustomerUUID: customerUUID, Page: 1, PageSize: 20,
})
if err != nil { return err }
_ = page.Items

created, err := contacts.Create(ctx, contactfw.CreateContactInput{
    CustomerUUID: customerUUID,
    DisplayName: "Example Contact",
    GivenName: "Example",
    FamilyName: "Contact",
    Status: contactfw.StatusActive,
    Roles: []contactfw.Role{contactfw.RolePrimary},
    Tags: []string{"test.local"},
    CreationIntent: contactfw.CreationIntentExplicitCreate,
})
if err != nil { return err }
_ = created.ContactUUID
```

示例中的显示名是测试输入，产品界面文案须使用插件的 locale。创建必填有效 `customer_uuid`、非空 `display_name`、`status` 和 `creation_intent`；`temporary` 只能配 `explicit_temporary`，`active/inactive` 配 `explicit_create`。角色限 `primary`、`legal_representative`；标签最多 20 个，单项最多 64 字节，总计最多 1024 字节，格式为小写字母/数字起头，后续可含 `.`、`_`、`-`。列表默认每页 20，最大 100。

`Get`/`Update` 均传 `customer_uuid` 与 `contact_uuid`。`ResolveIdentity` 传 `customer_uuid`、`channel_dictionary_item_uuid`、`external_subject`；`BindIdentity` 再传 `contact_uuid`。渠道使用字典项 UUID，不能用历史 channel code 或自由文本。所有对象引用须为 UUID。联系人响应包含 `contact_uuid`、`tenant_uuid`、`customer_uuid`、姓名、状态、角色、标签及时间；身份响应另含 `identity_uuid`、渠道字典项 UUID、外部 subject 与验证状态。`tenant_uuid` 是服务端返回的诊断信息，不是业务可选租户。

## 3. 客户清单、基础创建与更新

delegated 客户选择使用 `customerfw.NewAccountSelectorClient(invoker)`，使用以下 typed 方法：

```go
selector, err := customerfw.NewAccountSelectorClient(invoker)
if err != nil { return err }
page, err := selector.ListAccounts(ctx, customerfw.ListAccountsRequest{
    Query: "Example", Page: 1, PageSize: 20,
})
if err != nil { return err }
_ = page.Items // 每项以 CustomerUUID 作内部引用，以 DisplayName 等业务字段显示

account, err := selector.CreateBasicAccount(ctx, customerfw.CreateBasicAccountRequest{
    DisplayName: "Example Customer", PrimaryEmail: "customer@example.com",
    Status: "active", RequestID: requestID,
})
if err != nil { return err }
_ = account.CustomerUUID
```

需导入 `.../runtime/customerfw`。`ListAccountsRequest` 支持 `Query`、`Status`、`Page`、`PageSize`、`RequestID`；其 `TenantUUID` 字段**不会**由 service selector 发送给 Core，租户以出站凭证为准。创建输入支持 `status`、`primary_email`、`primary_phone`、`display_name`、`nickname`、`given_name`、`family_name`、`avatar_url`、`locale`、`timezone`，以及只用于请求追踪的 `RequestID`；不接受 tenant、密码或任意管理操作。具体必填/唯一性仍由 Core 校验。客户 UUID 来自返回结果，再用于联系人操作。

更新使用同一 service_manage 能力的 `operation=update`，示例：

```go
name, emptyPhone := "Updated Customer", ""
updated, err := selector.UpdateBasicAccount(ctx, customerfw.UpdateBasicAccountRequest{
    CustomerUUID: account.CustomerUUID,
    DisplayName: &name,
    PrimaryPhone: &emptyPhone,
    RequestID: requestID,
})
if err != nil { return err }
_ = updated
```

所有资料字段均为 `*string`：nil 不变、显式空字符串清空可选资料；display_name/status 不可清空。不接受类型、主要联系人、租户覆盖、null 或未知字段。响应按新对象解码，清空后省略的字段不能合并回旧值。Core 当前编辑还包含历史主要联系人修复，边界与待验收项见 [更新合同对齐](../../../contracts/customer-service-update-gap.md)。

local 模式下由插件实现同等业务语义、持久化事务、租户隔离和 UUID 引用；可参照 Skeleton 的[本地基础客户创建处理](../../../../skeleton/backend/go-gin/internal/transport/http/admin/customer/handler.go)。**不要把 delegated selector 注入 local 侧**。当前没有可直接 `NewRuntime` 的通用 Account Store，插件如果需要同一业务 Service 同时支持两种模式，应在插件内定义小接口，并在启动时单选本地实现或 `AccountSelectorClient`。

## 4. 能力、鉴权与错误

| 操作 | service capability |
|---|---|
| 客户清单 | `com.corex.customer.accounts.service_read` |
| 创建/更新基础客户 | `com.corex.customer.accounts.service_manage` |
| 联系人列表、读取、解析身份 | `com.corex.customer.contacts.service_read` |
| 创建/更新联系人、绑定身份 | `com.corex.customer.contacts.service_manage` |

这些客户端固定使用 `core_internal`、`INVOKE` 和 Core customer/accounts 或 customer/contacts 端点；插件业务不传裸 endpoint、tenant 或任意 payload。需要哪项就声明哪项：只读插件不申请 manage。实际插件 `capabilities.required` 需逐项写完整 ID，并按[delegated 装配与安装验收](usecase-delegated-runtime.md)加入 `com.corex.capabilities.grant_status.read`、完成发布、租户注册和 grant。**不要以 `*.admin_manage` 代替 service capability**；admin 管理 REST 与插件服务主体权限不同。

例如需要客户选择、创建客户及联系人读写的插件，在自己的 manifest 中声明：

```yaml
capabilities:
  required:
    - com.corex.capabilities.grant_status.read
    - com.corex.customer.accounts.service_read
    - com.corex.customer.accounts.service_manage
    - com.corex.customer.contacts.service_read
    - com.corex.customer.contacts.service_manage
```

该示例是能力清单片段，不会自动给现有 API Key 或已安装实例加 grant；实际插件按所用操作删减。

API Key 联调时还需该 Key 的精确 capability/scope grant；`403 registry.capability_forbidden` 应核对能力发布、租户启用、API Key/安装实例授权及实际使用的出站凭证。Contact 返回的稳定错误可用 `contactfw.CodeOf(err)` 识别，如 `CONTACT_INVALID_ARGUMENT`、`CONTACT_CUSTOMER_MISMATCH`、`CONTACT_IDENTITY_CONFLICT`、`CONTACT_CAPABILITY_FORBIDDEN`、`CONTACT_DELEGATE_UNAVAILABLE`。保留 HTTP 状态、`reason_code`、trace 供诊断；用户可见消息由插件 locale 映射，不直接展示原始 `err.Error()`。失败后不切换到本地表。

## 5. 联调与验收

Skeleton 管理端 `/admin/templates/framework-lab` 的“客户 / 联系人 Runtime”页可分别探测 Local Adapter 和 Delegated / PowerX Host：客户表、联系人表及新增弹窗调用 Skeleton 的调试接口。`framework_debug_route` 只用于该 Lab 的显式探测，**不是生产请求可用的模式开关**。业务路由继续使用启动选定的 Runtime。Skeleton 当前示例清单未必声明上述四项 service capability；检查实际插件 manifest，不以页面按钮代替安装授权。

在各自 `go.mod` 所在目录做源码合同检查：

```bash
cd framework/backend/go
go test ./runtime/contactfw ./runtime/customerfw -count=1
cd ../../../skeleton/backend/go-gin
go test ./internal/services/customer ./internal/transport/http/admin/customer -count=1
```

源码测试验证 DTO/Factory/客户端行为；local 还需真实数据库与租户隔离测试。API Key 成功只证明该开发凭证和当前 Core 环境可调用。插件安装后的 STS、实例 grant、撤权和跨租户拒绝需另做真实验收，当前不据 API Key 结果宣称已通过。


## 客户外部身份管理（2026-09-27）

这组操作管理 Customer 的外部身份，独立于 ContactIdentity，也独立于登录 Resolve。

| 方法 | 语义 | Core capability |
| --- | --- | --- |
| Lookup | 未关联返回 found=false，零业务写入 | com.corex.customer.external_identities.service_read |
| ListByCustomer | 当前插件命名空间内分页列表 | 同上 |
| Bind | 指定已有客户；同归属幂等、其他归属冲突 | com.corex.customer.external_identities.service_manage |
| CreateAndBind | 事务创建客户、membership、主要联系人、身份与审计；重复调用返回原关联 | 同上 |

Framework 接口为 `customerfw.ExternalIdentityStore`，从 `Runtime.Identities()` 取得。启动时通过 `RuntimeAdapters.Identities` 单选注入；Delegated 使用 `NewExternalIdentityClient(invoker)`，固定调用 `INVOKE core://customer/external-identities`。Skeleton 本地实现为 `customer.NewExternalIdentityStore(db, pluginID)`（repository 包）。无可用 adapter 明确失败，不回退。

身份唯一键为 `(provider, provider_subject)`；provider 来自可信插件配置/凭证，禁止 payload 覆盖。subject 使用 `channel:instance:entity:id`。Shopify 调用 `customerfw.ShopifyExternalIdentitySubject(domain, fullCustomerGID)`，不得将店铺 ID 替换域名、裁掉 GID 或按邮箱合并。登录 Resolve 与管理操作使用同一格式及同一身份锁，但 Resolve 仍保留登录用 resolve-or-create 语义。

Local 写入使用 PostgreSQL advisory transaction lock（SQLite 示例进程内串行化，数据库唯一约束继续保护身份）。查询只读；管理操作不为其他租户新建 membership，也不修复历史主要联系人。写入和审计同事务，重复写入不重复记录变更审计。跨进程 SQLite 写竞争可显式返回数据库忙，不宣称跨进程 SQLite 幂等重试已验收。

`customerfw.HTTPStatus` / `ReasonOf` 保留 Core 403/404/409/424 错误。Local 对应错误为 `CUSTOMER_ACCOUNT_NOT_FOUND`、`CUSTOMER_EXTERNAL_IDENTITY_CONFLICT`、`CUSTOMER_ACCOUNT_INVALID_ARGUMENT`、`CUSTOMER_PRIMARY_CONTACT_REQUIRED`。

### 调试与迁移

`/admin/templates/framework-lab` → 客户/联系人 → 客户外部身份。先选择 Local 或 Delegated，再加载并选中客户。客户表下方提供“联系人／外部身份”页签，选中客户或切换页签即自动加载相应记录，无需手动加载；“添加外部身份”仅为当前客户追加绑定，弹窗不含客户创建表单；可依次选择 Shopify、微信等平台，绑定后自动刷新当前客户身份。“新建客户并绑定”使用独立入口，完成后定位并选中接口返回的客户（包含幂等返回已有客户的情况）。两种操作均可先查询身份关联。弹窗沿用 Core 的 UModal、身份列表与显式操作布局；Core 目前没有这四项 Customer 身份管理的完整 UI，不能把 Contact 渠道绑定当作同一业务。

本地身份表新增稳定 `identity_uuid`。现有环境须执行 Skeleton 常规数据库迁移，先添加可空列并回填 UUID，再由 AutoMigrate 约束非空和唯一。回填包括已软删除记录且重跑保持 UUID。不可将 numeric id 当作身份 UUID。当前任务不自动执行迁移、不启动/重启后台。

需要为安装实例明确授权上述两项 capability。API Key 的精确 grant 必须带唯一可信 plugin_id；`capability-seed` 不会自动授予插件权限。本次不修改现有 Key 或安装实例 grant。

### 验证边界

本地聚焦测试覆盖零新增查询、重复绑定、冲突不改原归属、跨租户、八请求并发幂等、审计失败完整回滚、UUID 迁移重跑稳定。HTTP 测试覆盖模式隔离、可信租户上下文、错误状态透传及禁止字段。Core 的真实 API Key/STS 证据见 Core 仓库 `docs/contracts/customer-live-acceptance-2026-09-27.md`。

本插件新管理页面、本地 PostgreSQL 运行链路及安装生命周期仍需在用户迁移并运行最新后台后验收；源码测试、SQLite 测试和页面构建不代替这几项。
