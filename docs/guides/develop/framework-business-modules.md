# Framework 业务对象模块文档入口

插件接入的唯一入口已统一为 **[Framework 业务模块接入指南（local / delegated）](../features/009-consume-powerx-capability/guide.md)**。原有链接保留用于导航，不再维护第二份通用步骤、状态表或历史 Core 请求示例。

## 按目的阅读

| 目的 | 文档 |
|---|---|
| 选择模块、确认版本、实现 local、装配 delegated、验收 | [主指南](../features/009-consume-powerx-capability/guide.md) |
| Framework / Core / 插件的职责和禁止降级规则 | [双模式规范](framework-dual-mode-business-modules.md) |
| 已覆盖操作、测试证据、未验证范围 | [覆盖台账](../../contracts/powerx-core-framework-coverage.md) |
| 开发任务和历史演进 | [009 Tasks](../../../specs/009-consume-powerx-capability/tasks.md) |

## IAM 专题：业务展示与导入

通过启动时绑定的 IAM Registry 获取 DirectoryService，不另建 delegated MemberDirectory，不查 Core 数据库。

- 持久化稳定 member_uuid；查询业务记录后集中解析名称，不持久化 actor_display_name，也不逐行发起 N+1 查询。
- 严格完整读取使用 BatchGetMembers；历史记录允许成员失效时使用 BatchResolveMembers，并只对 missing_member_uuids 显示本地化未知状态。
- Excel 姓名输入使用 BatchResolveMembersByDisplayNames；found 保存成员 UUID，not_found/ambiguous 标注对应行，不随机选人。
- Core 姓名解析响应不包含 tenant_uuid。Go delegated 适配会用同一份 API Key/STS 凭据先读取 /tenant/iam/tenant，核对调用租户，再将已验证的范围补入 found 成员；不会把任意请求参数作为成员租户。每次批量姓名解析增加一次租户读取，复用现有目录读取授权，不缓存租户验证结果。插件继续校验成员所属租户。
- Core 租户目录返回数字 status 和字段 key。GetTenant 与姓名解析的租户核验共用数字状态 DTO 转换：status 保留为统一模型的数字字符串，key 映射为 TenantKey。HTTP 回归使用包含 tenant_uuid、key、name、status、type、plan 的完整响应。
- 401、403、上游失败属于整体调用错误，不转换为姓名不存在、空成功或本地 SQL 查询。
- member_uuid 与 user_uuid 是不同身份引用；不互相替代，不把 UUID 当显示名称。
- Directory/Authz/Context 的完整装配、测试和凭证要求以主指南为准。本页不推断任一部署环境已获授权。

## 变更记录

- 2026-09-08：收敛为主指南导航与 IAM 业务语义专题；移除过时 Media URL、源码路径和独立状态定义。未改变代码合同或提升安装态状态。
