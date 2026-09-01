# Data Model: Framework IAM 统一封装（Standalone/Delegated）

## 1. IAM Adapter

- **Purpose**: 统一封装模式实现（local/delegated）。
- **Key Fields**:
  - `mode`: `local | delegated`
  - `source`: `config | env`
  - `read_only`: bool（delegated=true）
  - `bound_at`: datetime
- **Rules**:
  - 启动期只允许绑定一个 adapter。
  - 运行期不得自动切换 adapter。

## 2. Mode Resolution Record

- **Purpose**: 记录模式判定来源与冲突结果，用于审计与排障。
- **Key Fields**:
  - `config_mode`
  - `env_mode`
  - `effective_mode`
  - `conflict_detected`
  - `decision_reason`
- **Rules**:
  - 当 `config_mode` 与 `env_mode` 冲突时，必须 fail-fast。

## 3. Identity Context

- **Purpose**: 统一身份上下文载体。
- **Key Fields**:
  - `tenant_uuid`
  - `user_uuid`（可选）
  - `member_uuid`（可选）
  - `roles[]`
  - `permissions[]`
  - `policy_version`
  - `trace_id`
- **Validation**:
  - `tenant_uuid` 必填，且必须是 UUID 字符串。
  - delegated/local 输出字段语义一致。

## 4. Tenant

- **Purpose**: local 模式租户边界实体。
- **Key Fields**:
  - `tenant_uuid`（唯一）
  - `tenant_key`
  - `name`
  - `status`
- **Relationships**:
  - Tenant 1:N Departments
  - Tenant 1:N Members
  - Tenant 1:N Roles

## 5. Department

- **Purpose**: 组织层级结构。
- **Key Fields**:
  - `department_uuid`（唯一跨边界标识）
  - `tenant_uuid`
  - `name`
  - `code`
  - `parent_department_uuid`
- **Relationships**:
  - Department N:1 Tenant
  - Department 1:N Members
  - Department 1:N Child Departments

## 6. Member

- **Purpose**: 人员主体（组织维度）。
- **Key Fields**:
  - `member_uuid`（唯一跨边界标识）
  - `tenant_uuid`
  - `user_uuid`（可选；与 member_uuid 独立）
  - `display_name`
  - `status`
- **Relationships**:
- Member N:1 Tenant
  - Member N:M Roles
  - Member N:M Departments

## 7. Role

- **Purpose**: 权限集合载体。
- **Key Fields**:
  - `role_uuid`（唯一跨边界标识）
  - `tenant_uuid`
  - `code`
  - `name`
  - `description`
- **Relationships**:
  - Role N:1 Tenant
  - Role N:M Permissions
  - Role N:M Members

## 8. Permission

- **Purpose**: 原子授权单元。
- **Key Fields**:
  - `resource`
  - `action`
  - `scope`（optional）
- **Rules**:
  - 以 `resource + action` 作为唯一键语义。

## 9. Authorization Decision

- **Purpose**: 权限判定结果与原因。
- **Key Fields**:
  - `allowed`
  - `reason_code`
  - `resource`
  - `action`
  - `tenant_uuid`
  - `user_id`
  - `mode`
  - `trace_id`
- **State**:
- `allowed` / `denied`

## 10. Identifier Rules

- `tenant_uuid`、`department_uuid`、`member_uuid`、`user_uuid`、`role_uuid` 是 API、事件、审计和跨服务引用的唯一标识。
- 数据库 numeric primary key 如存在，仅限 adapter 内部持久化实现；不得出现在 framework contracts、Host Contract 或业务插件 DTO。
- `display_name` 是展示字段，不是身份字段；为空时不得以任何 UUID 替代。
