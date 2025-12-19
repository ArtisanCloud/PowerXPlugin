# Feature Specification: Standalone 模式 IAM & RBAC

**Feature Branch**: `007-standalone-iam-rbac`  
**Created**: 2025-12-13  
**Status**: Draft  
**Input**: User description: "Standalone IAM & RBAC spec derived from docs/plan/007-standalone-iam-rbac.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 初始化本地 IAM (Priority: P1)

作为平台管理员，我需要通过一次性的迁移/种子流程创建默认租户和管理员账户，并能够在 Standalone 模式下成功登录管理后台。

**Why this priority**: 没有默认租户和管理员，本地系统无法运行，也无法进行后续组织治理。

**Independent Test**: 在全新环境执行 `setup` 流程后，使用默认管理员凭据访问 `/users/login` 并完成登录；若切换到 Delegated 模式，该入口不会出现。

**Acceptance Scenarios**:

1. **Given** 系统首次安装，**When** 管理员运行初始化命令并配置必需环境变量，**Then** 默认租户、部门、角色与管理员被创建且可成功登录。
2. **Given** 系统运行在非 Standalone 场景，**When** 用户访问组织/RBAC 菜单，**Then** 菜单被隐藏并提示使用宿主 IAM。

---

### User Story 2 - 组织结构与成员管理 (Priority: P1)

作为租户管理员，我需要在 Standalone 模式下维护租户信息、部门树以及成员（含邀请、启用/禁用、批量导入），以保证组织数据正确并可审计。

**Why this priority**: 没有清晰的组织结构与成员入驻，就无法将权限授予合适的人或满足审计要求。

**Independent Test**: 登录管理员帐号后，通过 Web Admin 创建/更新租户与部门、邀请成员并验证操作日志，可完全独立于角色管理实现价值。

**Acceptance Scenarios**:

1. **Given** 已初始化的默认租户，**When** 管理员创建新的部门并调整层级，**Then** 变更被持久化并产生审计记录。
2. **Given** 待邀请成员列表，**When** 管理员批量导入成员并分配部门，**Then** 成员状态更新且可在列表中启用/禁用。

---

### User Story 3 - 角色与权限治理 (Priority: P2)

作为安全负责人，我需要基于 Manifest 中定义的 scope 创建/克隆角色、批量分配权限与成员，并确保改动即时生效且可观察。

**Why this priority**: RBAC 是控制访问的核心，缺少角色/权限治理无法保证最小权限原则。

**Independent Test**: 在 Web Admin 中新增角色、配置权限树、将其绑定到成员，并通过 API 调用验证受保护资源的访问结果；同时检查指标是否记录成功/拒绝次数。

**Acceptance Scenarios**:

1. **Given** Manifest 中的权限目录，**When** 管理员创建自定义角色并勾选 scope，**Then** 角色与权限绑定被保存并出现在查询接口中。
2. **Given** 某成员被授予或撤销角色，**When** 该成员再次访问受保护 API，**Then** 即刻按照新权限决策通过或被拒绝，并在日志与指标中留下记录。

---

### User Story 4 - RBAC Enforcement & STS 合规 (Priority: P2)

作为平台架构师，我需要 Standalone 模式下的服务与宿主 PowerX 一样，通过 `plugin/resource/action` 三元模型做访问控制，并准备好在未来接受宿主网关颁发的短期令牌（STS）。

**Why this priority**: 统一的权限模型是将插件迁回宿主或被其他系统调用的前提。

**Independent Test**: 制定本地路由策略映射、启用 RBAC 中间件、验证健康检查放行与业务接口鉴权、调用本地 STS mint 功能；将 `POWERX_PROXY=1` 时确认相关菜单隐藏、接口改为宿主代理。

**Acceptance Scenarios**:

1. **Given** 某受保护路由未声明显式 scope，**When** 系统根据 `METHOD:/path` 自动推导资源/动作，**Then** RBAC 判定使用与宿主一致的三元值并记录结果。
2. **Given** 内部服务需要获取短期令牌，**When** 调用本地 STS 接口，**Then** 系统签发与宿主字段一致的 60 秒令牌并可用于后续请求验权。

---

### Edge Cases

- 当管理员意外删除唯一租户或将默认角色禁用时，系统必须阻止操作并提醒需要至少一个可用租户/角色。
- 当成员被禁用或密码锁定后，其现有 Refresh Token 需立刻失效，防止继续访问。
- 当系统切换到 Delegated 模式时，所有本地 IAM API 返回明确提示“由宿主托管”，避免双重来源。
- 当 SQLite 环境无法满足外键/唯一约束时，需要给出切换 Postgres 的警告并阻止破坏性迁移。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 必须提供一次性初始化流程，在 Standalone 模式下创建默认租户、部门、角色与管理员，并阻止在 Delegated 模式中执行相同操作。
- **FR-002**: 必须在 Standalone 模式暴露 `/auth/login|refresh|logout|me/context`，并确保菜单与登录入口仅在 `POWERX_PROXY=0` 时显示。
- **FR-003**: 必须提供租户基本信息查看/编辑接口与页面，并限制为 `system.admin`。
- **FR-004**: 必须允许管理员创建、修改、移动部门节点，支持树形展示与排序，并记录审计日志。
- **FR-005**: 必须支持成员的邀请、批量导入、启用/禁用、重置密码，并在所有列表/API 中按租户过滤。
- **FR-006**: 必须提供角色 CRUD、克隆功能以及从权限树批量勾选 scope，操作后需实时影响成员权限。
- **FR-007**: 必须实现 `plugin/resource/action` 三元 RBAC 中间件，包含显式声明与路由自动推导，并对健康检查等白名单放行。
- **FR-008**: 必须输出 IAM 相关指标（如成员总数、登录限流、RBAC 拒绝、种子耗时）与审计日志，并确保系统管理员可查看全部日志、租户管理员仅能查看所属租户日志，满足审计分权。
- **FR-009**: 必须提供 CLI 工具导出/备份本地租户、角色、权限，以及重新注入管理员凭据。
- **FR-010**: 必须更新文档（Standalone 指南、Runbook、Quickstart）记载环境变量、模式切换、菜单显隐与验证步骤。

### UI/UX Reference (与 PowerX 现有实现对齐)

为降低认知成本、方便跨插件共用体验，Standalone IAM 的 Web Admin 需要参考宿主 PowerX `settings` 套件（路径示例：`Core/PowerX/web-admin/app/pages/settings/*`）的交互模式，包含但不限于：

1. **设置入口与导航一致性**  
   - 参考 `settings/index.vue` 的卡片导航 + “快速设置”布局，为 Standalone 菜单提供概览与快捷入口；组织/租户相关菜单需在栅格中呈现自解释简介与图标。
2. **用户/部门/权限 3 合 1 视图**  
   - `settings/users/index.vue` 采用 Tab 切换 + `DepartmentManager`/`UserShell`/`PermissionShell` 组件，Standalone 版本也需提供 *部门树*、*成员列表*、*权限预览* 三个区域，依据 `tabs` 动态拼装，并根据用户角色（root、租户管理员）显隐 “权限” tab。
3. **角色管理体验**  
   - `components/settings/users/RoleManager.vue` 中的列过滤、租户远程搜索、分页、克隆/编辑抽屉等交互需复用：  
     - 角色列表必须支持按 scope/内置标记过滤、支持分页与搜索；  
     - 创建/编辑抽屉需提供租户下拉（远程搜索 + 保留已选项）、角色代码/名称/描述输入与 scope 选择；  
     - 权限树勾选逻辑与 Manifest scope 对应关系需在 UI 中即时反馈；  
     - 操作完成后弹出一次性的 Alert（`useOneShotAlert`）提示成功/失败。
4. **租户/配置联动**  
   - 参考 `settings/config` 页面快速设置区块，Standalone 的租户配置页应提供基础属性（Key/Name/Status/Plan）与功能开关（注册、邮件、维护模式等）表单，操作按钮（保存/重置）布局、描述文本风格与宿主保持一致。

> **Implementation note**：上述 UI 规格要求我们在 `skeleton/web-admin` 内提供对应的组件与样式，并在 CLI scaffold 中输出同样的结构，确保插件模板延续宿主体验。

### Key Entities

- **Tenant**: 表示独立租户（含 `id`, `key`, `name`, `status`），关联多部门、角色、成员，决定数据隔离范围。
- **Department**: 租户内的组织节点，需保存层级（`parent_id`, `path`），决定成员所在结构。
- **Member**: 拥有登录身份的用户，保存联系方式、状态、所属部门，绑定多个角色。
- **Role**: 权限集合，定义作用域（系统级/租户级）、可克隆和绑定成员。
- **Permission**: 最小权限单元，与 Manifest 中的 `resource.action` 对齐，可由种子或同步工具写入。
- **Policy (可选)**: 为未来 ABAC 留扩展点，记录 JSON 规则并与角色绑定，但本阶段仅占位。

### IAM 表关系约束

- `iam_users`（Account）存储跨租户的账号凭证（邮箱、密码哈希、头像等）；同一账号可加入多个租户。
- `iam_members`（Member）是账号在特定租户下的实例，携带 `tenant_uuid`、部门、状态与审计字段，并通过 `user_id` 指向 `iam_users`；`iam_member_roles` 连接成员与 `iam_roles`。
- `iam_roles`、`iam_permissions`、`iam_role_permissions` 形成 RBAC 树；`policy_version` 字段用于缓存刷新，`scope_type` 区分系统/租户级角色。
- `iam_departments` 通过 `parent_id`、`path` 构建组织树；`iam_refresh_tokens`、`iam_audit_logs` 以成员为主体记录会话与操作，支撑 Runbook 中“账号加入多个租户”验证步骤。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 新环境执行初始化后，管理员可在 2 分钟内完成首次登录并看到“组织与权限”菜单（仅 Standalone）。
- **SC-002**: 在租户内完成创建部门→邀请成员→分配角色→成员登录的端到端流程，全程 95% 操作步骤带有审计记录且可在 Runbook 中复现。
- **SC-003**: 所有受保护 API 在权限不足时返回 403，并在 `plugin_rbac_denied_total` 中可观测；E2E 测试覆盖成功与拒绝路径。
- **SC-004**: CLI `iam export` 能输出完整租户/角色/权限映射，并在 10 秒内完成单租户导出；导出的数据可用于灾备或迁移评审。
- **SC-005**: Delegated 模式下访问 Standalone IAM 页面触发隐藏逻辑的准确率达 100%，文档明确阐述切换步骤。

## Clarifications

### Session 2025-12-13

- Q: 审计日志访问权限需如何划分？ → A: 系统管理员可查看全部日志，租户管理员仅能查看自身租户日志。
