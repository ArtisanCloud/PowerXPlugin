# .NET Customer 管理与 Contact 合同

`CustomerRuntime.cs` 保留 Customer 登录、membership 与登录 Resolve 合同。
`CustomerManagement.cs` 提供独立管理合同，不能用登录 Resolve 替代只读身份查询。

| Interface | 操作 | Host 固定 binding |
| --- | --- | --- |
| ICustomerManagementService | List/Create/Update | core://customer/accounts |
| IContactService | List/Get/Create/Update/ResolveIdentity/BindIdentity | core://customer/contacts |
| ICustomerExternalIdentityManagement | Lookup/List/Bind/CreateAndBind | core://customer/external-identities |

`PowerXCustomerManagementClient` 仅通过 `ICapabilityRegistry` 调用固定的
`com.corex.customer.{aggregate}.service_read/service_manage`，协议 core_internal、方法
INVOKE。请求 DTO 无 tenant、actor、plugin ID、任意 endpoint/header 字段。Scope 的
tenant 必须来自可信本地身份且匹配启动服务租户；不将其作为 outbound payload。
Host 失败不会查询 local store。响应校验 UUID、联系人客户/租户范围与正式分页字段。

Local 实现由消费插件注入持久化 store。客户创建须原子保存客户、membership、主联系人；
company 须显式给自然人主联系人。Contact.roles 仅 primary/legal_representative，
temporary 需要 explicit_temporary；身份绑定需要同租户有效渠道字典 item UUID。
客户 profile 更新保留字段是否出现的区别，空字符串是显式清除；不改 type/主联系人。
HTTP consumer 应拒绝未知字段与 null，不能将 DTO 当作任意 JSON 转发入口。

EF consumer 可在启动时以固定 ProviderMode 调用 `AddPowerXRuntime` 并传入
`ServiceLifetime.Scoped`，使所有操作共享本次请求 DbContext。默认仍为 Singleton。
只有管理诊断页面可以显式选择另一个已装配 adapter；业务运行模式仍由启动配置决定。

Host service grant 与 Admin RBAC 独立；API Key 必须有对应 service 能力授权。
外部身份 grant 必须绑定唯一插件 ID。完整权威合同为 sibling PowerX
`specs/030-customer-contact/contracts/contact-core-internal.md` 与
`docs/contracts/customer-external-identity-management.md`。
