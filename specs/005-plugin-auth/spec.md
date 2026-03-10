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

**Independent Test**: 在启用 `IAMMode=delegated` 且 `POWERX_PROXY=1` 的环境下，仅部署前端+后端，验证管理员可以通过 `/users/login` → API 调用成功，并获得宿主返回的用户上下文。

**Acceptance Scenarios**:

1. **Given** 插件部署在宿主，`POWERX_CORE_ENDPOINT` 与 `POWERX_AUTH_TOKEN` 配置正确，**When** 管理员访问 `/users/login` 并输入有效账号，**Then** 前端写入 localStorage/token cookie、后端代理 `/admin/user/auth/login` 成功并跳转到受保护页面。
2. **Given** 管理员使用同一 Token 访问插件 API，**When** Token 过期并触发 401，**Then** 前端透明刷新 `refresh_token`，重试原请求并维持当前页面状态。

---

### User Story 2 - Standalone Local IAM (Priority: P2)

作为插件开发者，我希望在离线或独立模式下无需依赖 PowerX Core 也能登录插件，使用本地 IAM 数据库模拟租户、用户和角色，便于开发与自动化测试。

**Why this priority**: 保证在 CI、本地开发或 PoC 场景不必启动完整 PowerX 栈，提升开发速度。

**Independent Test**: 关闭 `POWERX_PROXY`，运行 `go run ./cmd/database/main.go setup` 创建本地 IAM 表与种子数据，验证从登录到调用受保护 API 都仅依赖本地存储。

**Acceptance Scenarios**:

1. **Given** `IAMMode=local` 且 `POWERX_PROXY=0`，**When** 服务启动并检测模式为 Local，**Then** `migrate` 自动包含 IAM 表且创建默认管理员，开发者可使用默认凭证登录。
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

### User Story 4 - CLI Package & Publish Workflow (Priority: P2)

作为插件开发者，我需要在完成本地开发后，通过 `px-plugin package` 自动构建前端/后端 artefact 并生成标准包，再使用 `px-plugin publish` 将包上传到指定的 PowerX Registry，这样宿主的 Marketplace/插件后台才能看到“待审核版本”，无需手动整理文件。

**Why this priority**：现阶段 CLI 仅支持热加载，package/publish 仍是占位命令，开发者必须手动运行 `go build`、`npm run build` 并上传 artefact，效率低且容易出错。补齐发布链路可确保插件进入 Marketplace 审核流程。

**Independent Test**：在带有示例插件的仓库执行 `px-plugin package --entry .`，查看 `.px-plugin/build` 下生成 `package.tar.gz`、`metadata.json`、`manifest.json` 等 artefact；随后运行 `px-plugin publish --entry . --channel dev --notes "feat"`，CLI 应向配置的 Registry（mock server）发送签名的包并返回 `publishId`。Package 成功后使用 `px-plugin doctor --check-devapi` 验证配置无误。

**Acceptance Scenarios**：

1. **Given** 插件仓库具有 `backend/` 与 `web-admin/`，`package.json`、`go.mod` 均可用，**When** 执行 `px-plugin package --entry .`，**Then** CLI 会依次运行 `npm --prefix web-admin run build`、`go build ./backend/cmd/plugin`、收集 manifest/RBAC/metadata，并在 `.px-plugin/build/<timestamp>/package.tar.gz` 与 `metadata.json` 中写入 artefact 列表、版本、hash；命令输出中列出 artefact 路径。
2. **Given** `~/.px-plugin/config.json` 或环境变量中设置了 `publishApi.baseUrl`、`publishApi.apiKey`，**When** 执行 `px-plugin publish --entry . --channel beta --notes "fix"`，**Then** CLI 会上传 package、manifest、metadata 到 `POST {publishApi.baseUrl}/internal/plugins/releases`，返回 `publishId`，在 PowerX 管理后台可看到待审核版本；如 Registry 不可达或返回错误，CLI 输出 remediation 并保持幂等。
3. **Given** 开发者尚未执行 package 或 `.px-plugin/build` 缺失 artefact，**When** 执行 publish，**Then** CLI 报错提示需先 package；若 config 中既未设置 Registry 基址也未提供 Token，则 publish 提示“缺少 publish API 配置”，并指向文档。

---

### User Story 5 - Delegated Token 失效体验 (Priority: P1)

作为被宿主 iframe 拉起的插件使用者，当宿主颁发的 Token 失效或刷新失败时，我希望仍然停留在当前页面并收到明确提示，而不是被强制跳转到插件自带的登录页，这样可以避免破坏宿主 UI/UX，并引导用户回到 PowerX 重新登录。

**Why this priority**：宿主模式是 Marketplace 的默认运行方式，错误地跳转到插件登录页会导致 iframe 黑屏、回退困难及审核不通过。

**Independent Test**：在 `POWERX_PROXY=1` + `insidePowerX=true` 的构建产物内，通过 Playwright 模拟 token 过期（删除 localStorage token 或让 refresh 返回 401/503），验证页面展示宿主模式专用的错误 Banner，未发起 `/users/login` 跳转，且 `px-bridge` 仍可在重新注入 token 后恢复。

**Acceptance Scenarios**：

1. **Given** 插件运行在 Delegated 模式，`useRuntimeConfig().public.insidePowerX === true`，**When** `useAuth.ensureFreshToken()` 检测到 token 缺失或 refresh 返回 401，**Then** `useAuth.failClosed()` 会清空本地存储、记录错误信息，并通过全局状态触发“请在 PowerX 中重新登录”的提示组件，页面保持在当前路由。
2. **Given** Delegated 模式下宿主 Access Token 由 postMessage 重新注入，**When** 用户点击提示中的“重试”或宿主重新发送 token，**Then** `useAuth.initAuth()` 会重新建立会话并自动移除提示，无需刷新页面或跳出 iframe。

---

### User Story 6 - Template CRUD RBAC 与模式切换 (Priority: P2)

作为插件开发者/审核者，我需要模板 CRUD API 在 Standalone 与 Delegated 模式下均声明、检查 RBAC 资源，这样在本地可以通过自有 IAM 控制权限，在宿主模式下也能让 PowerX 识别所需的 action 并做权限映射。

**Why this priority**：模板 CRUD 是脚手架交付的核心示例，缺乏 RBAC 信息会导致安全审计不过关，也让宿主无法自动收敛权限列表。

**Independent Test**：在 Standalone 模式下为角色授予 `base.templates.read/manage` 后访问 `/api/v1/templates`，未授予则收到 403；在 Delegated 模式下启用 `IAMMode=delegated` 且 `POWERX_PROXY=1`，即使插件不再维护角色，也能在 manifest 和 `/admin/rbac` 输出中看到模板相关资源，方便宿主 IAM 映射。

**Acceptance Scenarios**：

1. **Given** Standalone 模式，`IAMMode=local` 且 `POWERX_PROXY=0`，**When** 用户调用模板 CRUD API，**Then** `middleware.RBAC` 会从 Route→Permission Map 匹配 `base.templates.read`/`base.templates.manage` 并根据本地权限判定 200 或 403。
2. **Given** Delegated 模式，`IAMMode=delegated` 且 `POWERX_PROXY=1`，**When** 插件返回 manifest/RBAC 信息，**Then** PowerX 核心可读取 `base.templates.*` 资源并在宿主 IAM 中映射，同时插件前端根据 runtimeConfig 自动将模板页面切换为只读或展示“由宿主控制权限”的提示，不再要求 iframe 登录。

---

### Edge Cases

- Token 在后台刷新时浏览器 localStorage 不可用（Safari 隐私模式、跨 iframe 限制）如何处理？→ 需回退到 cookie 中的 `token` 或触发强制登录。
- `POWERX_CORE_ENDPOINT` 不可达或返回慢导致登录/刷新失败时如何处理？→ Delegated 模式需立即 fail-closed 并提示“宿主认证不可用，请稍后重试”；Local 模式提示配置不匹配。
- 同时存在多个浏览器 Tab，其中一个登出后如何通知其他 Tab？→ 需要监听 `storage` 事件同步清理状态。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 插件前端必须提供与 PowerX 宿主一致的登录、注册、忘记密码页面，并在 `setAuth` 中把 `access_token`、`refresh_token`、`token_type`、`expires_in`、`scope`、`expires_at` 以及 `token` cookie 写入 localStorage/cookie。
- **FR-002**: `useAuth` 必须实现 `initAuth`、`getToken`、`isTokenExpired`、`logout`，并在登出时清理 localStorage、sessionStorage、`px_*` cookie 与 Pinia 用户 store。
- **FR-003**: `useApiClient` 必须在请求前自动附加 `Authorization` 与 `tenant_uuid`，并在收到 401 时自动调用 `/admin/user/auth/refresh`（或本地 IAM 刷新接口）后重放原请求；刷新失败时需调用 `clearAuth` 并重定向到 `/users/login`。
- **FR-004**: 前端需要在全局中间件拦截所有除 `users/*` 白名单外的路由，未登录用户必须被导航到登录页，并带上原目标的 `redirect` 参数。
- **FR-005**: 后端必须提供 `internal/services/iam.IAMDirectory` 接口及 Delegated、Local 双实现，Resolver 根据 `IAMMode` 与 `POWERX_PROXY` 决定模式。
- **FR-006**: Delegated 模式下，后端必须使用 `POWERX_CORE_ENDPOINT`、`POWERX_AUTH_TOKEN` 调用宿主 `/admin/user/auth/login|refresh|logout|me/context`，并将宿主响应原样返回前端。
- **FR-007**: Local 模式下，`migrate` 必须包含 IAM 表（用户、角色、部门、租户）并生成可配置的默认管理员；管理员账号/密码需由 `PLUGIN_IAM_ADMIN_*` 环境变量或配置文件显式提供（缺失则报错并终止）；Delegated 模式下必须跳过这些表，避免污染宿主数据库。
- **FR-008**: 所有受保护路由必须挂载 JWT / Signed-Context 中间件，优先校验 `Authorization` Bearer Token；若未提供并启用了 Signed Context，则回退 `X-PowerX-CTX`。
- **FR-009**: Manifest 与 RBAC 输出必须声明插件运行所需的 IAM scope，例如 `iam.user.read`、`iam.role.read`，以便宿主在安装期提示权限需求。
- **FR-010**: 系统必须输出认证相关指标（登录成功/失败、刷新次数、Delegated 调用错误、当前 IAM 模式 Gauge），并将关键事件写入日志（含 trace_id 与租户信息）。
- **FR-011**: Delegated 模式下一旦 `POWERX_CORE_ENDPOINT` 无法访问或超时，登录与刷新请求必须 fail-closed——立即向用户返回“宿主认证不可用”提示并阻断继续操作，不得自动降级或复用过期 Token。
- **FR-012**: CLI `px-plugin package` 必须运行前端/后端构建（允许 `--skip-frontend`、`--skip-backend`），生成包含 manifest/RBAC/二进制/静态资源的 `package.tar.gz`、`metadata.json`，并写入 `.px-plugin/build/<version>/`，以便 publish 使用。
- **FR-013**: CLI `px-plugin publish` 必须读取配置中的 Registry 基址与 Token（允许 `--publish-api`、`--publish-token` 覆盖），将 package 与 metadata 上传到 PowerX Registry，并输出 `publishId`、审核链接；若 Registry 以 envelope (`{"code":...,"data":...}`) 返回，则需解析后反馈；失败需提供 remediation。
- **FR-014**: 当 `useRuntimeConfig().public.insidePowerX === true` 且 token 失效或刷新失败时，前端不得重定向至 `/users/login`，而是展示 Delegated 专用的错误提示、维持当前路由，并允许在收到宿主重新注入 token 后自动恢复；Standalone 模式仍沿用跳转策略。
- **FR-015**: 模板 CRUD 路由必须声明并输出 `base.templates.read` / `base.templates.manage` 等权限，后端 RBAC 中间件需要在 Standalone 模式下强制校验，在 Delegated 模式下保持 route→resource 映射供宿主 IAM 使用，前端需根据权限/模式控制 CRUD 按钮或显示“宿主控制”的提示。

### Key Entities *(include if feature involves data)*

- **AuthTokens**: 表示 `access_token`、`refresh_token`、`token_type`、`expires_in`、`scope`、`expires_at` 组成的结构；由 `useAuth` 与 `IAMDirectory.Login/Refresh` 返回并存储。
- **IAMDirectory**: 封装 Delegated/Local 两种实现的接口；方法包括 `Login`、`Refresh`、`Logout`、`CurrentUser`、`ListRoles`、`ListDepartments`、`CheckPermission`，供 HTTP handler 与中间件使用。
- **TenantContext**: 包含 `tenant_uuid`、`user_id`、`roles`、`permissions`、`policy_version` 等字段，可来自 JWT Claims 或 Signed Context Header，用于 RBAC 判定与日志审计。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 在 Delegated 模式下，90% 的登录请求在 2 秒内完成并返回受保护页面，并与宿主 Token 完全一致（字段/过期时间差异 ≤ 5 秒）。
- **SC-002**: Local 模式下，`go run ./cmd/database/main.go setup` 可在 1 分钟内完成 IAM 表迁移并生成可用的默认管理员，Playwright 用例可在独立环境全部通过。
- **SC-003**: Token 刷新成功率 ≥ 98%，失败时用户看到明确提示并在 5 秒内被重定向到登录页面，无静默失败。
- **SC-004**: 指标 `plugin_iam_delegate_errors_total`、`plugin_auth_logout_total`、`plugin_iam_mode` 在 Prometheus 中可见，运维可基于这些指标设定告警，使认证相关 P1 事故的定位时间 < 10 分钟。
