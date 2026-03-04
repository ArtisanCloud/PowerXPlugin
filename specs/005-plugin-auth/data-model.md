# Data Model – Plugin Auth Integration

## Overview
两种模式共享同一组实体；Delegated 模式只消费 DTO，不持久化；Local 模式在插件数据库中创建实际表。

## Entities

### Tenant
- **Fields**: `id (uint64)`, `key (string, unique, lowercase)`, `name`, `status (enum: active/disabled)`, `plan`, `created_at`, `updated_at`.
- **Constraints**: `key` 全局唯一；状态默认为 `active`。
- **Relationships**: `Tenant` 拥有多个 `Member`（User in tenant）和 `Department`。

### User
- **Fields**: `id (uint64)`, `email`, `phone`, `display_name`, `avatar_url`, `status (enum)`, `created_at`, `updated_at`.
- **Constraints**: `email` or `phone` or `username` 至少一个；`status` 默认为 `active`。
- **Relationships**: 与 `Tenant` 通过 `Member` 关联；可属于多个 `Role` via `MemberRole`。

### Member (TenantUser)
- **Fields**: `id`, `tenant_uuid`, `user_id`, `username (lowercase unique per tenant)`, `display_name`, `avatar_url`, `status`, `meta JSONB`.
- **Constraints**: `(tenant_uuid, username)` 唯一；`status` 继承 User 状态。
- **Relationships**: 多对多连接 `Role`（表 `member_roles`）。

### Role
- **Fields**: `id`, `tenant_uuid`, `code (string)`, `name`, `description`, `created_at`, `updated_at`.
- **Constraints**: `(tenant_uuid, code)` 唯一；`code` 仅小写字母+冒号。
- **Relationships**: 多对多 `Permission`（`role_permissions`）与 `Member`。

### Permission
- **Fields**: `id`, `resource`, `action`, `description`.
- **Constraints**: `(resource, action)` 唯一。
- **Relationships**: 关联 `Role`。

### Department
- **Fields**: `id`, `tenant_uuid`, `name`, `code`, `parent_id`, `description`, `created_at`, `updated_at`.
- **Constraints**: `(tenant_uuid, code)` 唯一；`parent_id` 可为空形成树。
- **Relationships**: 自引用形成组织结构；`Member` 可关联 Department。

### AuthTokens (volatile)
- **Fields**: `token_type`, `access_token`, `refresh_token`, `expires_in (int seconds)`, `scope`, `expires_at (unix ms)`.
- **Constraints**: `access_token` 必须存在；`expires_at` = `issued_at + expires_in`；`refresh_token` 仅在登录/刷新时下发。
- **Usage**: 存于 localStorage + cookie；后端仅透传。

### TenantContext (derived)
- **Fields**: `tenant_uuid`, `user_id`, `roles[]`, `permissions[]`, `policy_version`, `issued_at`.
- **Constraints**: 来自 JWT 或 Signed Context；需在 request middleware 校验；`roles` 用于 RBAC。

### IAMModeSetting
- **Fields**: `mode (enum: delegated|local)`, `source (enum: config/env/derived)`, `powerx_proxy`, `powerx_rbac_delegate`, `context_override`.
- **Usage**: 缓存 Resolver 判断结果，写入日志和指标。

## Relationships Diagram (textual)
```
Tenant 1---* Member *---1 User
Tenant 1---* Department
Tenant 1---* Role *---* Permission
Member *---* Role (member_roles)
Department self-references parent_id
AuthTokens ↔ TenantContext (runtime only)
```

## Validation Rules
- 登录 identifier 必须匹配 email/phone/username 中之一，全部标准化为 lower-case。
- Local 模式管理员凭证必须由 env/config 提供，否则 migrate 失败。
- Delegated 模式 API 请求必须携带 `tenant_uuid`（来自 cookie/localStorage），并自动附带 `Authorization: Bearer <token>`。
- 所有刷新请求若 token 已过期则先清理本地缓存再提示登录。
- `plugin_iam_mode` 指标需在服务启动和模式切换时更新。
