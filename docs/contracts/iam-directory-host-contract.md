# IAM Directory Host Contract

## 边界

- **PowerX Core**：delegated 模式下租户、成员、角色和权限的权威数据源，并负责发布正式 Host API。
- **PowerXPlugin framework**：对插件提供稳定的 IAM DTO、Directory 接口和 local/delegated adapter 装配。
- **业务插件**：只能从 framework `IAMRegistry` 获取 Directory；不得直连 Core 数据库、不得调用 Core 内部 Go service、不得解析 Gateway 原始对象。

本文件冻结 Framework 的语义与数据边界。Core Phase 2 已发布下述 tenant-scoped Host API；业务插件仍只能通过 Framework adapter 使用它们。

## 最小只读能力

| 操作 | 输入 | 成功输出 | 必须失败语义 |
|---|---|---|---|
| `GetMember` | `tenant_uuid`、`member_uuid` | 一个标准 `Member` | `IAM_MEMBER_NOT_FOUND`、`IAM_UPSTREAM_DEPENDENCY`、`IAM_FORBIDDEN` |
| `BatchGetMembers` | `tenant_uuid`、去重后的 `member_uuids[]` | 按请求 UUID 可关联的成员集合 | 同上；不得以空成功结果掩盖上游故障 |
| `BatchResolveMembers` | `tenant_uuid`、去重后的 `member_uuids[]` | 已解析成员与 `missing_member_uuids` | 仅成员缺失/跨租户可进入 missing；鉴权、参数与上游故障必须失败 |
| `ListDepartments` | 凭证 tenant | UUID 部门目录 | `IAM_FORBIDDEN`、`IAM_UPSTREAM_DEPENDENCY` |
| `ListRoles` | 凭证 tenant | UUID 角色目录 | `IAM_FORBIDDEN`、`IAM_UPSTREAM_DEPENDENCY` |
| `ListPermissions` | 凭证 tenant | UUID 权限目录 | `IAM_FORBIDDEN`、`IAM_UPSTREAM_DEPENDENCY` |
| `AuthorizationCheck` | `member_uuid`/`user_uuid`、正式 resource/action | `allowed` 与稳定 `reason_code` | 身份/调用者失败为 401/403；业务拒绝为 `200 allowed:false` |

Core 尚未发布分页 `ListMembers` 或 tenant metadata 读取接口；delegated adapter 对这两项必须明确返回依赖能力未配置，不能返回空集合。

## 标准 Member

```text
member_uuid: UUID，租户成员的唯一跨边界标识
tenant_uuid: UUID，必填
user_uuid: UUID，可选，账号身份，与 member_uuid 不可混用
display_name: 人类可读名称，可为空
status: 稳定状态枚举
```

所有跨服务引用、事件和审计必须使用 `member_uuid`。若无法解析显示名称，返回空值或明确错误；绝不返回 `member_uuid` 作为 `display_name`。

## 模式规则

- local adapter 从插件本地 IAM 权威存储读取，输出相同 DTO。
- delegated adapter 仅调用 Core 已发布 Host API；Core 不可用时返回明确依赖错误。
- delegated 模式不写插件本地组织/IAM 数据，不存在自动切换到 local、缓存伪造结果或轮询 fallback。

## delegated 成员目录的前置授权

成员读取使用 `com.corex.iam.members.read`；部门、角色和权限目录使用 `com.corex.iam.directory.read`；授权判定使用 `com.corex.iam.authorization.check`。所有能力都要求 tenant-scoped STS service actor 或 Gateway API key 的精确 scope 授权。

```json
{
  "allowed_capabilities": [
    "com.corex.iam.members.read",
    "com.corex.iam.directory.read",
    "com.corex.iam.authorization.check"
  ]
}
```

未授予该 capability 的插件服务必须收到 `403 IAM_FORBIDDEN`；不得因为同租户、旧凭证或 HTTP 路由可达而放行。成员 UUID 非法或批量请求包含重复 UUID 时必须返回 `400 IAM_INVALID_ARGUMENT`。Core 的错误 envelope 同时提供 `error_code` 与 `reason_code`，framework 以其 HTTP 状态和稳定错误码映射 Directory 错误。

## 验收

每个 adapter 都必须覆盖：单成员、批量成员、成员不存在、跨租户拒绝、权限拒绝、上游不可用，以及“显示名不等于 UUID”的合同测试。
