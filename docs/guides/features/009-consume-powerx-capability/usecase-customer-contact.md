# 场景：Customer 基础资料与 Contact Runtime

从[Framework 业务模块接入指南](guide.md)进入本文。本文描述插件如何选择客户、创建基础客户资料，以及在客户名下管理联系人。C 端登录、Customer JWT 与 membership 鉴权仍按 [Customer Auth 指南](../../develop/auth/customer.md)接入。

## 1. 选择正确的合同

| 需求 | Framework 入口 | local | delegated |
|---|---|---|---|
| C 端注册、登录、验证、membership | `customerfw.NewRuntime` 的 `Auth()`、`ExternalIdentity()`、`Membership()` | 插件注入 `LocalCustomerStore` 或独立 adapters | Core Auth、External Identity、Membership 客户端；部分操作需要独立的 customer JWT |
| 选择客户、创建基础客户资料 | `customerfw.NewAccountSelectorClient` 的 `ListAccounts`、`CreateBasicAccount` | 插件实现自己的客户存储/服务；Skeleton 有本地示例，Framework **尚无通用 Account Runtime/LocalStore** | 固定的 Core service capability；由凭证确定 tenant |
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

## 3. 客户清单与基础创建

delegated 客户选择使用 `customerfw.NewAccountSelectorClient(invoker)`，只调用两个 typed 方法：

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

local 模式下由插件实现同等业务语义、持久化事务、租户隔离和 UUID 引用；可参照 Skeleton 的[本地基础客户创建处理](../../../../skeleton/backend/go-gin/internal/transport/http/admin/customer/handler.go)。**不要把 delegated selector 注入 local 侧**。当前没有可直接 `NewRuntime` 的通用 Account Store，插件如果需要同一业务 Service 同时支持两种模式，应在插件内定义小接口，并在启动时单选本地实现或 `AccountSelectorClient`。

## 4. 能力、鉴权与错误

| 操作 | service capability |
|---|---|
| 客户清单 | `com.corex.customer.accounts.service_read` |
| 创建基础客户 | `com.corex.customer.accounts.service_manage` |
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
