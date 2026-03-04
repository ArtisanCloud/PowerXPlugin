# 007 - Standalone 模式 IAM & RBAC PRD

> PowerXPlugin 在独立部署（Standalone）场景下即是一个完整子系统，必须自带租户、组织、身份认证与权限管控能力。本 PRD 描述目标组织架构、鉴权策略、交付物与验收方式，作为 005-plugin-auth 之后的加固计划。

---

## 1. 背景与问题陈述

- Skeleton 当前已具备 Local IAM 模式（`internal/services/iam/local_store.go`、`public/auth_handler.go`），但只覆盖管理员种子登录、Token 生命周期，尚未定义组织架构、RBAC 颗粒度以及运维规范。
- 插件独立部署时往往需要：
  1. 多租户隔离（不同租户的模板/数据互不可见）。
  2. 租户内多角色协同（运营、审核、审计、访客）。
  3. 与 PowerX Manifest / RBAC schema 对齐，未来可平滑迁移到宿主 Delegated 模式。
- 缺乏完整 PRD 会导致：
  - 组织模型不清晰，难以扩展部门/角色层级；
  - Local IAM 与 Delegated 映射不一致，安装/迁移风险大；
  - 缺文档/验收标准，测试难度高。

## 2. 目标

1. 给出 Standalone 模式的组织实体、层级、权限域定义，确保任何插件项目都能即插即用。
2. 在 Skeleton/Web Admin/CLI 中固化 Local IAM 的创建、登录、RBAC 管控与审计流程。
3. 对齐 `docs/guides/develop/standalone/README.md` 的运行指引，补充 IAM/RBAC 运维、观测与测试矩阵。
4. 兼容 Delegated 模式：本地角色/权限需映射到 Manifest `rbac.schema.json`，便于宿主读取。

## 3. 范围与非目标

| 范围 | 内容 |
|------|------|
| ✅ Standalone IAM 组织模型 | 租户、部门（Department）、成员（Member）、角色（Role）、权限（Permission）、策略（Policy）关系。 |
| ✅ 登录 & 会话 | 本地 `POST /api/v1/auth/login|refresh|logout|me/context`，JWT/Refresh Token 生命周期。 |
| ✅ RBAC 管理 UI/API | Web Admin 中的租户、角色、成员管理页面；后台路由/服务；Playwright 验收。 |
| ✅ 配置与运维 | 环境变量、迁移脚本、观测指标、排障指南。 |
| ✅ Manifest/RBAC 同步 | `internal/manifestx`、`docs/contracts/rbac.schema.json`、`capabilities/catalog.json` 需包含本地权限。 |
| ❌ 自建 IdP/SSO | 不引入外部 IdP，仅在 Local 模式运行。 |
| ❌ 非插件通用业务域 | 例如 CRM/账单的专用表，这里仅定义 IAM/RBAC。 |

## 4. Persona 与场景

| Persona | 诉求 | 对应角色 |
|---------|------|----------|
| 平台管理员（Platform Admin） | 安装后首位管理员，负责创建租户/部门、指派角色。 | `system.admin` |
| 运营负责人（Ops Manager） | 维护模板/业务实体，需要读取与编辑权限。 | `ops.manager` |
| 审核员（Auditor） | 仅能查看、导出日志，不能改写配置。 | `auditor.readonly` |
| 外部协作者（External Collaborator） | 只访问特定项目或模板集合。 | `project.collaborator` |

## 5. 组织模型

### 5.1 实体与层级

| 实体 | 描述 | 关键字段 | 关系 |
|------|------|----------|------|
| Tenant | 独立租户或空间，区分数据边界。 | `id` (UUID)、`key`、`name`、`status` | 1:N Departments、1:N Roles、1:N Members |
| Department | 租户内组织节点，支持树形结构。 | `id`、`tenant_id`、`parent_id`、`path` | N:1 Tenant、N:1 Parent、1:N Members |
| Member | 真实用户，绑定登录凭证。 | `id`、`tenant_id`、`user_id`、`email`、`password_hash`、`status` | N:N Roles（通过 `member_roles`）、N:1 Department |
| Role | 权限集合。 | `id`、`tenant_id`、`code`、`name`、`description`、`scope` | N:N Permissions（`role_permissions`）、N:N Members |
| Permission | 最小 RBAC 单位，对应 Manifest scope。 | `id`、`code`（如 `base.templates.manage`）、`actions`（`read/manage`）、`resource` | N:N Roles |
| Policy | 可选 JSON 规则（未来扩展 ABAC）。 | `id`、`tenant_id`、`kind`、`definition` | 关联 Role 或 Department |

> SQLite 默认迁移 `iam_*` 表；PostgreSQL 支持完整外键。Primary Tenant 使用 `PLUGIN_IAM_TENANT_KEY/NAME` 注入。

### 5.2 核心角色矩阵

| 角色编码 | 权限组合 | 备注 |
|----------|----------|------|
| `system.admin` | 全部权限（管理租户、RBAC、模板、监控） | 仅平台管理员拥有。 |
| `tenant.admin` | 管理本租户成员、部门、配置；可授予非系统权限 | 不可管理系统设置。 |
| `ops.manager` | `base.templates.manage`、`base.capabilities.manage`、`base.runtime.manage` | 面向模板/能力运营。 |
| `auditor.readonly` | 所有 `*.read` 权限 + 审计日志访问 | 禁止写操作。 |
| `project.collaborator` | 精确到项目/模板的子权限（通过 Policy/Scope） | 默认只读，可手动添加写权限。 |

### 5.3 与 PowerX RBAC 三元模型保持一致

参考宿主仓库的《docs/standards/powerx/backend/rbac/readme.md》，PowerX 的权限抽象统一为 `plugin/resource/action` 三元组（Triple），配以系统级与租户级作用域。Standalone 模式需要：

1. **资源/动作命名规范对齐**：本地 `iam_permissions` 表分别记录 `plugin`（固定 `com.powerx.plugin.base`）、`resource`（如 `templates`、`iam.roles`）、`action`（`read|manage|assign|audit`），并保持唯一索引 `(plugin, resource, action)`，方便与宿主 `permission_repo.Sync` 或 Manifest 进行数据对接。
2. **主体与 Claims**：JWT Payload 中注入 `tenant_id/uuid`、`member_id`、`roles`、`scope` 等声明，与宿主 `pkg/corex/iam/reqctx` 可读取的字段同构，确保未来切换 Delegated 模式时，RBAC 中间件只需替换 Token 解析器。
3. **作用域字段**：`Role`/`Permission` 增加 `scope_type`（`system` or `tenant`），`system.admin` 可跨租户管理；其它角色默认只在所属租户生效。
4. **策略来源追踪**：`iam_permissions.source` 标记 `local_seed`、`manifest_sync` 等，便于 CLI/脚本与宿主的权限同步器保持一致。

> 通过沿用宿主的三元模型，可复用其自动推导策略、OpenAPI 同步工具与 STS 策略缓存，减少自定义实现带来的割裂。所有组织/RBAC 菜单仅在 Standalone（`POWERX_PROXY=0` 且 `POWERX_RBAC_DELEGATE=false`）时展示；Delegated 模式读取宿主 IAM，因此需隐藏相关入口以免混淆。

## 6. 权限模型与资源域

- 资源域（Resource Domain）：`templates`、`capabilities`、`runtime`、`iam.members`、`iam.roles` 等，与 Manifest `rbac.schema.json` 中的 `resources[*].name` 对齐。
- 动作（Action）：`read`、`manage`、`assign`、`audit`。
- Scope 表达式：`resource.action`，例如 `base.templates.manage`。Manifest 中需声明：
  ```yaml
  resources:
    - name: base.templates
      actions:
        - key: read
          description: 查看模板列表/详情
        - key: manage
          description: 创建/更新/删除模板
  ```
- 本地 RBAC 评估：
  1. `auth middleware` 将 JWT 中的 `role_ids` 注入 `context`。
  2. `RBAC middleware` 查询缓存（`role_permissions`）或 Redis（未来扩展）判断是否包含所需 scope。
  3. 支持 `AllowList` 中的系统任务（健康检查等）跳过校验。
  4. **路由自动推导**：参考 PowerX 网关策略，若 Handler 未显式声明 scope，则根据 `METHOD:/path` 自动推导 `resource/action`：`GET|HEAD→read`、`POST→create`（映射到 `manage`）、`PUT|PATCH→update`（映射到 `manage`）、`DELETE→delete`（映射到 `manage`），路径按首段归一，去除 `/api/v1` 与租户前缀。

## 7. 核心流程

1. **安装/初始化**
   - 运行 `go run ./cmd/database/main.go setup`：迁移 `iam_*` + 插件业务表、写入默认租户/角色/权限。
   - 管理员凭 `PLUGIN_IAM_ADMIN_EMAIL/PASSWORD` 登录 `skeleton/web-admin/nuxt`。
2. **租户管理**
   - API：`POST /api/v1/admin/iam/tenants`（仅 `system.admin`）创建租户；`PATCH /.../{id}` 更改状态。
   - UI：Web Admin `Settings → Organization` 提供租户信息查看、删除确认。
3. **部门管理**
   - API：`/api/v1/admin/iam/departments` 支持树形 CRUD，字段 `parent_id`、`path`。
   - 变更需写审计日志：`IAM_DEPARTMENT_UPDATED`。
4. **成员入驻**
   - 管理员可在 UI 中邀请成员（生成临时密码/邮件文案）。
   - API：`POST /api/v1/admin/iam/members`，参数包含 `roles[]`、`department_id`。
   - 支持批量导入（CSV → 后端任务）。
5. **角色 & 权限**
   - 创建角色时需从 `permissions` 列表勾选 scope，支持“克隆角色”。
   - 角色变更后通知前端刷新 `useAuth` 上下文（Storage event）。
6. **登录与 Session**
   - 复用现有 `/users/login.vue`。本地模式 `auth_handler` 调 `local_store`，签发 JWT/Refresh。
   - Token payload 包含 `tenant_id`、`role_ids`、`permissions`（可选）以及 `plugin_id`、`policy_version` 等，用于与宿主 STS 令牌结构对齐，未来可直接在 Delegated 模式透传至插件。
7. **审计 & 观测**
   - 每次敏感操作写 `audit_logs`（表或文件），字段：`actor_id`、`action`、`resource`、`diff`。
   - 指标扩展：`plugin_iam_member_total{status}`、`plugin_iam_role_assign_total{result}`。

## 8. 功能需求清单

| ID | 描述 | 详情 | 优先级 |
|----|------|------|--------|
| F1 | Local IAM 数据模型补全 | `iam_roles` 支持 `scope_type`、`iam_permissions` 引入 `domain/action` 字段，新增 `iam_role_permissions` 迁移 | P0 |
| F2 | IAM Admin API | 在 `internal/transport/http/admin/iam` 下新增 `tenant_handler.go`、`department_handler.go`、`role_handler.go`、`member_handler.go` | P0 |
| F3 | RBAC 中间件对齐 | `internal/middleware/rbac.go` 支持资源→scope 映射和缓存；需单测 | P0 |
| F4 | Web Admin 组织管理页面 | `app/pages/admin/iam/*.vue`，含租户概览、成员列表、角色编辑；Pinia store + `$fetch` | P1 |
| F5 | 权限配置 UI | 基于 `docs/contracts/rbac.schema.json` 自动渲染权限树，支持搜索/批量勾选 | P1 |
| F6 | 审计日志 | `internal/services/audit/logger.go` + `app/pages/admin/audit/logs.vue` | P1 |
| F7 | CLI 支持 | `px-plugin iam export` 输出本地租户/角色/权限用于备份，`px-plugin iam seed` 重置管理员 | P2 |
| F8 | 文档与 Runbook | `docs/guides/develop/standalone/README.md`、`docs/operations/runbooks/iam-rbac.md`、`specs/005-plugin-auth/quickstart.md` 更新 Standalone IAM 步骤 | P0 |

## 9. 系统设计概览

### 9.1 后端（Go）
- **目录结构**：
  - `internal/services/iam/directory.go`：接口新增组织/角色/权限 CRUD 方法。
  - `internal/services/iam/store`：拆分 `auth` 与 `rbac` 存储；缓存层优先使用内存，支持注入 Redis。
  - `internal/transport/http/admin/iam/*`：REST Handler + DTO + 校验。
  - `internal/router/router.go`：`/api/v1/admin/iam` 路由组加载 RBAC 中间件，Scopes: `iam.tenant.manage`、`iam.member.manage`、`iam.role.manage` 等。
  - `internal/manifestx/manifest.go`：新增权限声明 + 默认菜单（“组织与权限”）。
  - `internal/middleware/rbac.go`：实现对 `plugin/resource/action` 三元组的判定，与宿主 `RBACService.Enforce` 对应，后续可抽象 Exporter 将授权结果同步给 `px-plugin` CLI。
- **迁移**：`cmd/database/migrate` 新增 IAM 关联表；`seed` 注入默认角色/权限映射。
- **RBAC 中间件**：根据 Handler 配置 `RequiredScopes`，未命中返回 403 + 错误码 `IAM_SCOPE_DENIED`。

### 9.2 前端（Nuxt 4）
- 新增页面：
  - `/admin/iam/overview`：租户统计、成员概览。
  - `/admin/iam/members`：列表/创建/编辑/批量导入。
  - `/admin/iam/roles`：权限树、角色分配。
  - `/admin/iam/departments`：树图 + 拖拽排序。
- 组件：`components/iam/PermissionTree.vue`、`FormInviteMember.vue` 等。
- 数据：使用 `$fetch('/api/v1/admin/iam/...')`，并利用 `useAuth` 的 `tenantId`。
- **模式感知**：在 `app/plugins/auth.client.ts` 或 Layout 层读取 `runtimeConfig.public.insidePowerX` 与 `POWERX_PROXY` 标志，仅当 `standalone=true` 时渲染上述菜单与页面；Delegated 模式自动隐藏该分组并在导航中展示宿主导向链接（如“回到 PowerX 宿主 IAM”）。

### 9.3 CLI & 模版
- `px-plugin init` 生成的 `config.example.yaml` 自动包含 Standalone IAM 配置段落。
- `px-plugin iam export`：读取 SQLite/Postgres，输出 JSON/CSV（供迁移）。
- Scaffold 模板同步新增页面、Go Handler。

### 9.4 网关/STS 对齐策略

- Standalone 模式虽不经过宿主网关，但在设计上保留“策略 → RBAC 判定 → 签发短期令牌（STS）”的链路：
  - **策略**：读取 `docs/contracts/rbac.schema.json` 与 `capabilities/catalog.json`，生成 `METHOD:/pattern → resource/action` 对映表，可由 CLI 导出。
  - **判定**：`RBACService.Enforce` 接口统一提供 `pluginID` 入参，便于未来宿主 Authorizer 直接调用。
  - **STS**：`internal/services/iam/local_store.go` 增加 `MintSTS(ctx, claims)`，可发 60 秒短期令牌（aud=`plugin:<id>`）供内部服务间调用；字段与宿主 `internal/infra/plugin/manager/rbac.go` 生成的令牌保持一致。
- 在独立部署内若需要多进程/多服务之间的权限传递，可直接使用该 STS 机制；切换到宿主时只需要改为接受 `X-PowerX-CTX` + STS 即可，无需重写业务逻辑。

## 10. 安全与合规

- 密码策略：最少 10 位，必须包含大小写/数字/特殊字符；`local_store` 采用 `bcrypt` 或 `argon2id`。
- Refresh Token 加密存储，支持黑名单（`revoked_refresh_tokens`）。
- 审计日志至少保留 180 天，可导出 CSV。
- 所有敏感 API 需开启速率限制：默认 `100 req/min`/IP。
- 环境变量：
  - `POWERX_RBAC_DELEGATE=false`（Standalone）
  - `PLUGIN_IAM_PASSWORD_POLICY_JSON`（可选）
  - `PLUGIN_IAM_LOGIN_THROTTLE=5/300s`。

## 11. 运维与观测

| 指标 | 说明 |
|------|------|
| `plugin_iam_member_total{status}` | 成员数量，按启用/禁用拆分。 |
| `plugin_iam_login_throttle_total{result}` | 登录限流统计。 |
| `plugin_rbac_denied_total{resource}` | 被拒绝的权限请求。 |
| `plugin_iam_seed_duration_seconds` | `setup` 执行耗时。 |

日志扩展：`request_trace` 添加 `role_codes`、`scope`。Runbook 需覆盖：管理员密码重置、租户锁定、RBAC 缓存刷新。

## 12. 实施阶段与验收

| 阶段 | 里程碑 | 验收方式 |
|------|--------|----------|
| Phase A：模型与 API (1 sprint) | 完成 F1/F2/F3，Go 单测覆盖率 ≥80% | `go test ./...`、`Postman collection` 自测、`PLAYWRIGHT_LOCAL_IAM=1` 登录。 |
| Phase B：前端体验 (1 sprint) | 完成 F4/F5，E2E 覆盖成员 CRUD + 角色分配 | `npm --prefix skeleton/web-admin/nuxt run test:e2e -- iam-local`（新增用例）。 |
| Phase C：审计与 CLI (0.5 sprint) | 完成 F6/F7 + 文档 | Review `docs/plan/007`、`docs/guides` 更新；执行 `px-plugin iam export`. |

## 13. 风险与缓解

- **多租户隔离不彻底**：需在所有查询加 `tenant_id` 过滤，并通过 Go 单测校验；考虑数据库约束（复合唯一键 `tenant_id + code`）。
- **RBAC 缓存失效导致权限漂移**：实现订阅机制或提供 `/api/v1/admin/iam/cache/refresh` 手动刷新。
- **SQLite 迁移限制**：复杂约束在 SQLite 不支持，文档需说明切换 Postgres 的步骤；在 `setup` 输出警告。
- **Delegated 映射**：在 Manifest 中记录 Local ↔ PowerX scope 的映射表，CLI 校验缺失项。

## 14. 依赖与后续工作

- 依赖 005-plugin-auth 已完成的 Local IAM 基座、`docs/guides/develop/auth.md`。
- 后续计划：
  1. 与 Marketplace 团队讨论 Standalone → Hosted 移植的租户/角色迁移工具。
  2. 引入 ABAC/Policy Engine（`casbin`）扩展行级权限。 
  3. 评估将审计日志输出到对象存储或 SIEM，满足企业合规需求。
