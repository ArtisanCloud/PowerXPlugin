# Data Model: Standalone 模式 IAM & RBAC

## Tenant
- **Fields**: `id (UUID, pk)`, `key (string, unique)`, `name (string, 2-64 chars)`, `status (enum: active|suspended)`, `created_at`, `updated_at`.
- **Constraints**: Key 全局唯一；至少保留一个 active Tenant。
- **Relationships**: 1:N Departments、1:N Roles、1:N Members。
- **State Transitions**: `active → suspended`（禁止新成员登录）；不能删除最后一个 active tenant。

## Department
- **Fields**: `id (UUID)`, `tenant_uuid (FK)`, `parent_id (UUID nullable)`, `path (ltree)`, `name`, `sort_order`。
- **Constraints**: `(tenant_uuid, name)` 唯一；path 自动维护。
- **Relationships**: N:1 Tenant、N:1 Parent、1:N Children、1:N Members。
- **State Transitions**: 支持树内移动（更新 parent/path）；删除前需无成员。

## Member
- **Fields**: `id (UUID)`, `tenant_uuid`, `user_id`, `email`, `phone`, `password_hash`, `status (active|disabled|locked)`, `last_login_at`。
- **Constraints**: `(tenant_uuid, email)` 唯一；密码使用 bcrypt/argon2；禁用后刷新 token。
- **Relationships**: N:N Roles（member_roles），N:1 Department。
- **State Transitions**: `active ↔ disabled`（手动），`active → locked`（风控）。

## Account vs Member 关系梳理
- `iam_users`（Account）保存跨租户的账号凭证（邮箱、密码、头像等），不附带 `tenant_uuid`，因此同一账号可加入多个租户。
- `iam_members`（Member）是账号在具体租户下的实例，带有 `tenant_uuid`、组织与状态字段，并通过 `user_id` 指向 Account。
- `iam_member_roles` 连接 Member 与 Role，`iam_departments` 通过 `department_id` 与 Member 相联；审计、Token 等附属表同样以 Member 作为租户内主体。

```
iam_users (Account)
      │ 1:N
      ▼
iam_members (Member) ──< iam_member_roles >── iam_roles ──< iam_role_permissions >── iam_permissions
      │
      ├─ N:1 iam_departments
      └─ 1:N iam_refresh_tokens / iam_audit_logs
```

## Role
- **Fields**: `id (UUID)`, `tenant_uuid`, `code (string)`, `name`, `scope_type (system|tenant)`, `description`。
- **Constraints**: `(tenant_uuid, code)` 唯一；系统角色仅平台管理员可编辑。
- **Relationships**: N:N Members、N:N Permissions（role_permissions）。
- **State Transitions**: `draft → active`（可分配）；`active → archived`（不可新分配但历史保留）。

## Permission
- **Fields**: `id (UUID)`, `plugin (string)`, `resource (string)`, `action (string)`, `description`, `source (local_seed|manifest_sync)`。
- **Constraints**: `(plugin, resource, action)` 唯一；必须映射 Manifest。
- **Relationships**: N:N Roles。
- **State Transitions**: `active → deprecated`（当 Manifest 移除时）。

## RolePermission（关联表）
- **Fields**: `role_id`, `permission_id`, `tenant_uuid`, `policy_version`。
- **Constraints**: 复合唯一；更新后需使缓存失效。

## AuditLog
- **Fields**: `id`, `tenant_uuid`, `actor_member_id`, `action`, `resource`, `diff (jsonb)`, `created_at`。
- **Constraints**: actor 可为空（系统事件）；按租户分区或索引。
- **Relationships**: N:1 Tenant。

## RefreshToken
- **Fields**: `id`, `member_id`, `tenant_uuid`, `token_hash`, `expires_at`, `revoked_at`。
- **Constraints**: Token 存储哈希；禁用成员后批量标记 revoked。

## STS Mint Record
- **Fields**: `id`, `tenant_uuid`, `aud`, `subject`, `issued_at`, `expires_at`, `policy_version`。
- **Purpose**: 可选日志，排查内部服务之间的 STS 使用。
