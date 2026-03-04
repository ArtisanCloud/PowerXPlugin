# Data Model

> 以 Go Gin 现有模型为权威基线；本文件描述 FastAPI 侧需要对齐的关键实体。

## Tenant
- **Purpose**: 租户隔离与配置承载。
- **Key Fields**: id, key, name, status, created_at, updated_at
- **Identity Rule**: key 全局唯一。

## User
- **Purpose**: 认证与账号信息。
- **Key Fields**: id, email, phone, display_name, avatar_url, status, created_at, updated_at
- **Identity Rule**: email/phone 至少一项唯一。

## Member
- **Purpose**: 用户与租户的关联。
- **Key Fields**: id, tenant_uuid, user_id, username, display_name, status, created_at, updated_at
- **Relations**: Tenant 1..n Members；User 1..n Members。

## Role
- **Purpose**: 权限角色集合。
- **Key Fields**: id, tenant_uuid, code, name, description, scope_type, created_at, updated_at
- **Relations**: Role n..n Permission；Role n..n Member。

## Permission
- **Purpose**: 资源与动作授权项。
- **Key Fields**: id, plugin, resource, action, effect, status, created_at, updated_at
- **Identity Rule**: (plugin, resource, action) 唯一。

## Department
- **Purpose**: 组织结构树。
- **Key Fields**: id, tenant_uuid, name, code, parent_id, description, sort_order
- **Relations**: Department 自关联（parent/children）。

## Template
- **Purpose**: 可配置模板资源。
- **Key Fields**: id, tenant_uuid, name, description, content, created_at, updated_at

## Capability
- **Purpose**: 能力清单与生命周期管理。
- **Key Fields**: id, name, status, version, created_at, updated_at

## RuntimeSession
- **Purpose**: 运行时会话与调用控制。
- **Key Fields**: id, session_id, status, created_at, updated_at

## Notes
- 第一阶段仅需对齐联调所需实体（认证、IAM、模板、能力相关）。
- 表名与字段以 Go Gin 为权威基线，对齐时不得自行改名或简化。
