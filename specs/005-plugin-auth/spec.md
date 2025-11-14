# Feature Specification: Plugin Auth Integration

**Feature Branch**: `005-plugin-auth`  
**Created**: 2025-11-14  
**Status**: Draft  
**Input**: User description: "根据docs/plan/004-plugin-auth-integration.md文档，实现对应的spec相关文件"

## Clarifications

### Session 2025-11-14

- Q: 当插件运行在 Delegated 模式且 `POWERX_CORE_ENDPOINT` 暂时不可访问时，登录/刷新请求应该采用哪种行为？ → A: 立即阻断并提示“宿主认证不可用”，要求稍后重试（不中断安全策略）。
- Q: 在 Local IAM 模式下，默认管理员账号/密码应该如何配置才能兼顾安全与易用？ → A: 必须通过环境变量或 config.yaml 显式提供（缺失则报错并拒绝生成默认账号）。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Delegated Login Flow (Priority: P1)

作为插件管理员，我需要在 PowerX 宿主控制台内打开插件页面时复用宿主的登录、Token、RBAC 与组织架构信息，这样无需重复认证即可访问插件 API。

**Why this priority**: 没有 Delegated 登录，插件无法在 PowerX 生产环境运行，是 Marketplace 上架的前置条件。

**Independent Test**: 在启用 `POWERX_PROXY=1` 和 `POWERX_RBAC_DELEGATE=true` 的环境下，仅部署前端+后端，验证管理员可以通过 `/users/login` → API 调用成功，并获得宿主返回的用户上下文。

**Acceptance Scenarios**:

1. **Given** 插件部署在宿主，`POWERX_CORE_ENDPOINT` 与 `POWERX_AUTH_TOKEN` 配置正确，**When** 管理员访问 `/users/login` 并输入有效账号，**Then** 前端写入 localStorage/token cookie、后端代理 `/admin/user/auth/login` 成功并跳转到受保护页面。
2. **Given** 管理员使用同一 Token 访问插件 API，**When** Token 过期并触发 401，**Then** 前端透明刷新 `refresh_token`，重试原请求并维持当前页面状态。

---

### User Story 2 - Standalone Local IAM (Priority: P2)

作为插件开发者，我希望在离线或独立模式下无需依赖 PowerX Core 也能登录插件，使用本地 IAM 数据库模拟租户、用户和角色，便于开发与自动化测试。

**Why this priority**: 保证在 CI、本地开发或 PoC 场景不必启动完整 PowerX 栈，提升开发速度。

**Independent Test**: 关闭 `POWERX_PROXY`，运行 `go run ./cmd/database/main.go setup` 创建本地 IAM 表与种子数据，验证从登录到调用受保护 API 都仅依赖本地存储。

**Acceptance Scenarios**:

1. **Given** `POWERX_PROXY=0`、`POWERX_RBAC_DELEGATE` 未设置，**When** 服务启动并检测模式为 Local，**Then** `migrate` 自动包含 IAM 表且创建默认管理员，开发者可使用默认凭证登录。
2. **Given** Local 模式已登录，**When** 用户在 UI 中重置 Token 或登出，**Then** 所有 Token、Cookie、sessionStorage 被清除，API 请求返回 401 并重定向到登录页。

---

### User Story 3 - Token Lifecycle & Observability (Priority: P3)

作为运维人员，我需要监控插件在两种模式下的认证状态（登录次数、刷新次数、Delegated 调用错误），并在退出时确保权限被回收，便于审计。

**Why this priority**: 认证问题直接影响租户安全与合规，必须具备监控与快速排障能力。

**Independent Test**: 配置 Prometheus 指标采集，模拟成功/失败登录、登出和 Delegated 调用异常，确认指标与日志满足排障需求。

**Acceptance Scenarios**:

1. **Given** 插件运行在 Delegated 模式，**When** 宿主返回 401/5xx，**Then** 后端记录 `plugin_iam_delegate_errors_total`、错误日志包含 trace_id，且前端提示“认证已失效”并跳转登录。
2. **Given** 用户点击“退出”，**When** 登出流程完成，**Then** `plugin_auth_logout_total` 指标 +1、所有本地存储被清空、`/admin/user/auth/logout`（Delegated）或本地 IAM session（Local）被销毁。

---

### Edge Cases

- Token 在后台刷新时浏览器 localStorage 不可用（Safari 隐私模式、跨 iframe 限制）如何处理？→ 需回退到 cookie 中的 `token` 或触发强制登录。
- `POWERX_CORE_ENDPOINT` 不可达或返回慢导致登录/刷新失败时如何处理？→ Delegated 模式需立即 fail-closed 并提示“宿主认证不可用，请稍后重试”；Local 模式提示配置不匹配。
- 同时存在多个浏览器 Tab，其中一个登出后如何通知其他 Tab？→ 需要监听 `storage` 事件同步清理状态。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 插件前端必须提供与 PowerX 宿主一致的登录、注册、忘记密码页面，并在 `setAuth` 中把 `access_token`、`refresh_token`、`token_type`、`expires_in`、`scope`、`expires_at` 以及 `token` cookie 写入 localStorage/cookie。
- **FR-002**: `useAuth` 必须实现 `initAuth`、`getToken`、`isTokenExpired`、`logout`，并在登出时清理 localStorage、sessionStorage、`px_*` cookie 与 Pinia 用户 store。
- **FR-003**: `useApiClient` 必须在请求前自动附加 `Authorization` 与 `X-Tenant-ID`，并在收到 401 时自动调用 `/admin/user/auth/refresh`（或本地 IAM 刷新接口）后重放原请求；刷新失败时需调用 `clearAuth` 并重定向到 `/users/login`。
- **FR-004**: 前端需要在全局中间件拦截所有除 `users/*` 白名单外的路由，未登录用户必须被导航到登录页，并带上原目标的 `redirect` 参数。
- **FR-005**: 后端必须提供 `internal/services/iam.IAMDirectory` 接口及 Delegated、Local 双实现，Resolver 根据 `POWERX_PROXY`、`POWERX_RBAC_DELEGATE`、`context.iam_mode` 决定模式。
- **FR-006**: Delegated 模式下，后端必须使用 `POWERX_CORE_ENDPOINT`、`POWERX_AUTH_TOKEN` 调用宿主 `/admin/user/auth/login|refresh|logout|me/context`，并将宿主响应原样返回前端。
- **FR-007**: Local 模式下，`migrate` 必须包含 IAM 表（用户、角色、部门、租户）并生成可配置的默认管理员；管理员账号/密码需由 `PLUGIN_IAM_ADMIN_*` 环境变量或配置文件显式提供（缺失则报错并终止）；Delegated 模式下必须跳过这些表，避免污染宿主数据库。
- **FR-008**: 所有受保护路由必须挂载 JWT / Signed-Context 中间件，优先校验 `Authorization` Bearer Token；若未提供并启用了 Signed Context，则回退 `X-PowerX-CTX`。
- **FR-009**: Manifest 与 RBAC 输出必须声明插件运行所需的 IAM scope，例如 `iam.user.read`、`iam.role.read`，以便宿主在安装期提示权限需求。
- **FR-010**: 系统必须输出认证相关指标（登录成功/失败、刷新次数、Delegated 调用错误、当前 IAM 模式 Gauge），并将关键事件写入日志（含 trace_id 与租户信息）。
- **FR-011**: Delegated 模式下一旦 `POWERX_CORE_ENDPOINT` 无法访问或超时，登录与刷新请求必须 fail-closed——立即向用户返回“宿主认证不可用”提示并阻断继续操作，不得自动降级或复用过期 Token。

### Key Entities *(include if feature involves data)*

- **AuthTokens**: 表示 `access_token`、`refresh_token`、`token_type`、`expires_in`、`scope`、`expires_at` 组成的结构；由 `useAuth` 与 `IAMDirectory.Login/Refresh` 返回并存储。
- **IAMDirectory**: 封装 Delegated/Local 两种实现的接口；方法包括 `Login`、`Refresh`、`Logout`、`CurrentUser`、`ListRoles`、`ListDepartments`、`CheckPermission`，供 HTTP handler 与中间件使用。
- **TenantContext**: 包含 `tenant_id`、`user_id`、`roles`、`permissions`、`policy_version` 等字段，可来自 JWT Claims 或 Signed Context Header，用于 RBAC 判定与日志审计。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 在 Delegated 模式下，90% 的登录请求在 2 秒内完成并返回受保护页面，并与宿主 Token 完全一致（字段/过期时间差异 ≤ 5 秒）。
- **SC-002**: Local 模式下，`go run ./cmd/database/main.go setup` 可在 1 分钟内完成 IAM 表迁移并生成可用的默认管理员，Playwright 用例可在独立环境全部通过。
- **SC-003**: Token 刷新成功率 ≥ 98%，失败时用户看到明确提示并在 5 秒内被重定向到登录页面，无静默失败。
- **SC-004**: 指标 `plugin_iam_delegate_errors_total`、`plugin_auth_logout_total`、`plugin_iam_mode` 在 Prometheus 中可见，运维可基于这些指标设定告警，使认证相关 P1 事故的定位时间 < 10 分钟。
