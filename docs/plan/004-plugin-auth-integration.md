# 004 - 插件侧 Auth 机制对齐计划

## 背景
- 当前 `PowerXPlugin` Skeleton/Web Admin 仅提供 `useApiClient` 的 Token 读取占位实现（参见 `skeleton/web-admin/app/composables/api/_base.ts#getAuthToken`），缺少登录页、`useAuth` 状态管理、Refresh/Logout 处理，也没有与宿主 PowerX `web-admin` 的认证契约对齐。
- 宿主 `PowerX` 已在 `web-admin/app/composables/useAuth.ts`、`web-admin/app/pages/users/login.vue` 以及 `backend/internal/transport/http/admin/auth` 中实现 JWT + Refresh Token 的成体系流程，并在 `docs/use_cases/_from_hub/SCN-IAM-USER-ROLE-001` 中定义了 IAM 场景对安全的一致性要求。
- 需求：插件需在「独立运行/子系统」与「嵌入 PowerX Admin」两种模式下复用同一套登录、登出、Token 存储与会话续期策略，为后续 Marketplace 分发与工具化打下基础。

## 目标
1. 在 Skeleton/Web Admin 中提供完整的登录/登出 UI、`useAuth` 可组合函数以及 Token Persist 逻辑，与宿主字段命名、生命周期保持一致。
2. 在 Skeleton/Backend 中提供 JWT 校验、`/admin/user/auth/*` 代理（或直连 Core API）的默认实现，确保插件 API 仅在通过 PowerX 签发的 Token 下工作。
3. 在 Scaffold 模板与 CLI 输出中自动带出上述能力，保证 `px-plugin init` 即含 Auth 基座。
4. 补充文档与测试，覆盖开发、部署、运行阶段的配置指引与验收准则。

## 范围
- ✅ `skeleton/web-admin`: composables、middleware、stores、pages、runtime config。
- ✅ `skeleton/backend`: JWT 中间件配置、auth proxy service、公共 handler、配置文件。
- ✅ `framework` 与 `scaffold/templates`: 同步 Skeleton 变更，便于二次生成。
- ✅ docs/guides + docs/standards: Auth 集成说明、配置样例、Troubleshooting。
- ❌ 不改动宿主 PowerX Core 代码，仅引用其契约/接口。
- ❌ 暂不构建自有 IdP，仅作为 PowerX Auth Client。

## 依赖 & 参考
- `PowerX/web-admin/app/composables/useAuth.ts`：Token 生命周期处理。
- `PowerX/web-admin/app/pages/users`：登录/Register/忘记密码场景布局与文案。
- `PowerX/web-admin/app/composables/api/services/authService.ts`：API DTO 定义。
- `PowerX/backend/internal/transport/http/admin/auth`：`/admin/user/auth` 端点协议。
- `SCN-IAM-USER-ROLE-001`：IAM 守护场景，对一致性、审计与回收的要求。

## 交付物
1. `skeleton/web-admin/app/composables/useAuth.ts` 及配套 `api/services/authService.ts`、`middleware/auth.ts`、`stores/user.ts`。
2. `skeleton/web-admin/app/pages/users/login.vue`（+ register/forgot-password 轻量版）以及 Layout 入口的登出入口。
3. `skeleton/backend/internal/services/authproxy`、`internal/transport/http/public/auth_handler.go`、`RegisterRoutes` 中的公共登录代理与受保护登出路由。
4. `docs/guides/develop/auth.md`（新增）与 `docs/standards/powerx-plugin/contract/*.md` 的配置更新。
5. Playwright 场景（登录成功、过期刷新、登出清理）与 Go `auth_jwt` 中间件单测。

## 技术方案概述

### 1. Nuxt 4 前端认证基座
- **Auth 服务抽象**：在 `skeleton/web-admin/app/composables/api/services/authService.ts` 中复刻宿主 DTO，Base URL 允许同时调用 Plugin API（默认）与宿主 Core API：  
  - 通过 `useRuntimeConfig().public.powerxCoreBase`（新增配置）决定 Core Auth API 域名；独立模式缺省为 `.env` 中的 `POWERX_CORE_ENDPOINT`。  
  - 登录、刷新、登出默认直连宿主 `/admin/user/auth/*`；`skipAuth: true` 选项保证登录前可调用。
- **状态管理**：新增 `useAuth` composable（结构对齐 PowerX），负责：  
  - `setAuth`：写入 `access_token`、`refresh_token`、`token_type`、`expires_in`、`scope`、`expires_at` 到 `localStorage`，并同步 `token` Cookie（供 iframe/Bridge）。  
  - `clearAuth`：清理 Local/SessionStorage、`token`/`px_*` Cookies，并触发 `useUserStore().clearUserState()`。  
  - `initAuth`：在 `plugins/auth.client.ts` 内于应用启动时执行，复用宿主逻辑。  
  - `getToken`/`isTokenExpired`：供 `useApiClient` 与拦截器复用。  
  - `logout`：调用 `authService.logout() → clearAuth → navigateTo('/users/login')`。
- **HTTP 拦截器**：拓展 `skeleton/web-admin/app/composables/api/_client.ts`：  
  - 在 `onResponseError` 中捕捉 401 → 自动尝试 `refreshToken`（若 `refresh_token` 存在）→ 成功则重播原请求，失败则 `clearAuth()` 并跳转登录。  
  - 请求头继续沿用 `Authorization: Bearer <token>` 与 `X-Tenant-ID`。
- **路由保护**：新增 `middleware/auth.global.ts`：  
  - 对除 `users/login|register|forgot-password` 外的页面强制检测 `useAuth().token`；  
  - 若缺失则重定向登录并附带 `redirect` query。  
  - 兼容 `/_p/<plugin>/bridge-dev` 等调试页面（通过白名单）。
- **UI**：在 `skeleton/web-admin/app/pages/users/` 下创建 `login.vue`、`forgot-password.vue`、`register.vue` 的压缩版，复用宿主组件/样式但删减企业 branding；  
  - 页面对接 `useAuthService`，交互一致（错误提示、Remember Me、redirect）。  
  - 在导航/用户菜单中提供 `登出` 触发器，调用 `useAuth().logout()`.

### 2. Go Backend & Token 验证
- **JWT 中间件配置**：在 `skeleton/backend/internal/router/router.go` 中确保 `POWERX_SECURITY_*` 默认为宿主同款；新增配置校验（启动时报错）。  
- **Auth Proxy Service**：新增 `internal/services/authproxy`：  
  - 封装调用宿主 `/admin/user/auth/login|refresh|logout|me/context` 的逻辑；  
  - 读取 `POWERX_CORE_ENDPOINT`、`POWERX_AUTH_TOKEN`（宿主注入的服务间鉴权 Token）。  
  - 提供 `Login(ctx, req)` 等方法以供 Public Handler 与 CLI 复用。
- **Public 路由**：在 `internal/transport/http/public` 下新增 `auth_handler.go`：  
  - `POST /api/v1/auth/login` → 代理到 Core，返回宿主响应；  
  - `POST /api/v1/auth/refresh`、`POST /api/v1/auth/logout` 同理；  
  - 可选：`GET /api/v1/auth/me/context` 调用 Core `/admin/auth/me/context`。
- **Protected 路由**：所有业务路由追加 `JWTAuth` 中间件，统一读取 `Authorization` 或 `X-PowerX-CTX`。  
- **Manifest / RBAC**：在 `internal/manifestx/manifest.go` 中声明所需的 IAM Scopes（例如 `iam.user.read`），确保宿主在安装时知晓依赖。

### 3. IAM 服务抽象与运行模式（Local ↔ Delegated）
- **接口定义**：新增 `internal/services/iam`（或 `pkg/iam`）暴露 `IAMDirectory` 接口：
  ```go
  type IAMDirectory interface {
    Login(ctx context.Context, tenant, identifier, password string) (Tokens, error)
    Refresh(ctx context.Context, refreshToken string) (Tokens, error)
    Logout(ctx context.Context, refreshToken string) error
    CurrentUser(ctx context.Context) (*UserContext, error)
    ListRoles(ctx context.Context, filters RoleFilters) ([]Role, error)
    ListDepartments(ctx context.Context, filters DeptFilters) ([]Department, error)
    CheckPermission(ctx context.Context, resource, action string) error
  }
  ```
  由 `auth_handler`、`authproxy`、RBAC 中间件与未来 CLI 共用。
- **DelegatedIAMClient**（默认模式）：  
  - 生效条件：`POWERX_PROXY=1`（宿主注入）或显式设置 `POWERX_RBAC_DELEGATE=true`。  
  - 依赖 `POWERX_CORE_ENDPOINT`（Core API 基址）、`POWERX_AUTH_TOKEN`（服务 Token）、`POWERX_TENANT_ID`（如宿主按租户注入）拼装 HTTP/gRPC 请求；必要时读取 `POWERX_PLUGIN_ID` 作为 `aud`.  
  - 失败时抛出带追踪信息的错误，供上层统一处理/重试。
- **LocalIAMStore**（开发/离线）：  
  - 生效条件：`POWERX_PROXY!=1` 且未开启 `POWERX_RBAC_DELEGATE`。  
  - 使用插件自己的数据库 Schema（`POWERX_PLUGIN_SCHEMA`/SQLite DSN），新增 `internal/entity/models/iam` 及仓储实现保留用户、角色、部门等最小集合。  
  - 提供基础 Seed（默认租户、管理员账号），并通过 `go run ./cmd/database/main.go seed` 可选开启。  
  - 仅用于开发/自动化测试，文档中标注“不可在生产持久化宿主 IAM 数据”。
- **AutoMigrate 策略**：  
  - `migrate.MigratePluginModels` 拆分 `pluginTables`（业务表）与 `iamTables`（新建）；  
  - 当 `POWERX_PROXY=1` 或 `POWERX_RBAC_DELEGATE` 为 true 时仅迁业务表；否则迁两者；  
  - 继续尊重 `POWERX_RUN_MIGRATE`（或 `runtime.run_migrate`），并在日志打印当前模式。  
  - CLI 输出友好提醒：“delegated 模式下跳过 IAM 表，如需本地模型请 unset POWERX_PROXY”。
- **模式决策**：  
  - 在 `bootstrap` 层新增 `IAMResolver`，将 `Config.Context.IAMMode`（YAML 可选字段）、`POWERX_PROXY`、`POWERX_RBAC_DELEGATE` 按优先级组合，返回 `delegated`/`local`。  
  - 前端 `useAuthService` 继续指向插件 API；后端 `auth_handler` 根据 Resolver 选择实现。  
  - 文档记录如何通过 `POWERX_RBAC_DELEGATE=false` 在宿主环境短暂启用 Local 模式（仅供诊断）。
- **遥测与告警**：  
  - 新增 `plugin_iam_mode{mode="delegated|local"}` Gauge；  
  - `plugin_iam_delegate_errors_total`、`plugin_iam_local_sync_seconds` 等指标帮助定位问题；  
  - 日志中统一输出 `mode=delegated`/`mode=local` 及主要环境变量值（遮蔽 Token）。

> **模式切换环境变量速记**：`POWERX_PROXY=1` 默认启用 Delegated IAM；若 `POWERX_RBAC_DELEGATE` 显式设为 truthy（`1/true/on`）也强制委托；反之在 `POWERX_PROXY!=1` 且未设置 Delegate 的场景会落入 Local IAM。`context.iam_mode`（YAML 配置）可作为最终 override。

### 4. Scaffold & CLI 同步
- 执行 `npm run sync:templates` 将新文件同步到：  
  - `scaffold/templates/web-admin/nuxt/app/**`、`scaffold/templates/backend/go-gin/**`;  
  - `tools/cli/internal/templates/data/**`（用于 `px-plugin init`）。  
- 更新 `scaffold/templates/plugin.yaml.tmpl`，加入 `POWERX_CORE_ENDPOINT`、`POWERX_AUTH_TOKEN` 等默认配置。

### 5. 文档、观测与回归
- 新增 `docs/guides/develop/auth.md`，内容包括：配置环境变量、与宿主 Core 通信流程、Token 调试方法。  
- 更新 `docs/guides/develop/standalone-mode.md`，加入「如何在本地运行 login」章节。  
- 在 `docs/operations/runbooks` 下补充 `auth-troubleshooting.md`（常见错误码、排查步骤）。  
- 指标：前端记录登录成功率、Token 刷新次数；后端暴露 `plugin_auth_login_total`、`plugin_auth_failed_total`。

## 实施阶段
| 阶段 | 目标 | 关键任务 |
|------|------|----------|
| Phase 0：对齐契约 (1d) | 验证宿主 API & DTO | ① 梳理 `PowerX` Auth API 与响应字段；② 明确环境变量、租户获取方式；③ 输出对齐备忘录（docs/plan 附录）。 |
| Phase 1：前端基座 (3d) | 完成 `useAuth` + Service + Middleware | ① 新增 `useAuth`、`authService`、`auth.client.ts`；② 改造 `_client.ts` 刷新逻辑；③ 补充页面/导航 UI。 |
| Phase 2：后端代理 & IAM Resolver (3d) | 提供 Proxy API + 双模式 IAM 抽象 | ① 实现 `authproxy` service 与 Handler；② 注入 `IAMDirectory` 接口 + Delegated/Local 实现与 Resolver；③ 拆分 migrate 表集并联动 `POWERX_PROXY`/`POWERX_RBAC_DELEGATE`；④ 单测 JWT/IAM 组合路径。 |
| Phase 3：模板 & CLI (1d) | 能通过 `px-plugin init` 生成带 Auth 的脚手架 | ① 执行同步脚本；② 更新 CLI 模板 & Example config；③ 自测脚手架新工程运行登录流程。 |
| Phase 4：测试 & 文档 (2d) | Playwright + Go Tests + 指南 | ① 添加登录/登出 E2E；② 后端 proxy 单测；③ 编写 `docs/guides/develop/auth.md` & Runbook；④ 验收 checklist。 |

## 测试与验收
- **前端**：Playwright 场景（成功登录→跳转、错误凭证提示、刷新 Token 后继续访问、登出后重定向 login）。  
- **后端**：Go 单测覆盖 JWT 中间件、Auth Proxy 错误分支（网络异常/401）；集成测试同时验证 Delegated 模式（接 `POWERX_CORE_ENDPOINT` mock server）与 Local 模式（直查内存仓储）的行为一致性。  
- **配置验证**：`npm run lint`, `npm test`, `go test ./...`, `make e2e-auth`（新增脚本）必须通过。  
- **验收准则**：遵循 SCN-IAM-USER-ROLE-001 中对一致性与回收的约束（例如：登出需清理所有 Token/Cookie；权限不足返回 401/403 并可审计）。

## 风险与待决事项
- **核心 API 可用性**：独立模式下若无法访问 PowerX Core，需要缓存/降级策略（可在 Phase 2 评估本地 Mock Auth）。  
- **跨域/iframe 兼容**：插件注入到宿主时的域名可能不同，需确认登录页是否允许被 iframe；必要时提供 `redirectToHostLogin` 的配置。  
- **Token 双存储**：LocalStorage + Cookie 需注意 XSS/CSRF；计划启用 `sameSite=lax` + 短期 Refresh Token。  
- **多租户上下文**：`tenant_id` 读取逻辑仍是占位（`getTenantId()`），需在 Phase 1 内与宿主桥接（例如通过 `Bridge` 同步 tenant）。  
- **退出流程**：宿主 `/logout` 依赖 `refresh_token`；需确认插件能安全保存 Refresh（可考虑加密存储）。  
- **后续扩展**：若未来插件需脱离 PowerX Auth，自建 IdP 则需另立计划。

## 后续动作
- Phase 0 完成后，在 `docs/plan/004` 附加 `AUTH.md` 子章节记录接口样例。  
- 计划落地期间与 IAM 团队评审一次（review meeting）确保 Proxy 设计合规。  
- 与发布团队确认 Marketplace 安装流程是否需要额外权限声明。
