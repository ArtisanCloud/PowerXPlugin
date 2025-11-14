# Tasks: Plugin Auth Integration

**Input**: Design documents from `/specs/005-plugin-auth/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Update `skeleton/backend/etc/config.example.yaml` & `skeleton/backend/etc/README.md` 记录 `POWERX_CORE_ENDPOINT`、`POWERX_AUTH_TOKEN`、`POWERX_RBAC_DELEGATE`、`PLUGIN_IAM_ADMIN_*` 等环境变量及使用建议。
- [X] T002 将 `powerxCoreBase` 暴露到 Nuxt runtime：在 `skeleton/web-admin/nuxt.config.ts`（或等效配置）中读取 `POWERX_CORE_ENDPOINT` 并注入 `useRuntimeConfig().public.powerxCoreBase`。

---

## Phase 2: Foundational (Blocking Prerequisites)

- [X] T003 创建 `skeleton/backend/internal/services/iam/directory.go`，定义 `IAMDirectory` 接口、`IAMMode` 枚举、Token DTO 与通用错误类型。
- [X] T004 在 `skeleton/backend/internal/bootstrap/iam_resolver.go` 实现 IAM 模式解析逻辑，依序读取 `context.iam_mode`、`POWERX_RBAC_DELEGATE`、`POWERX_PROXY` 并缓存在依赖容器。
- [X] T005 新增 IAM 实体（`Tenant`/`User`/`Member`/`Role`/`Permission`/`Department`）到 `skeleton/backend/internal/entity/models/iam/`，含 Gorm 标签与关系定义。
- [X] T006 拆分 `skeleton/backend/cmd/database/migrate/migrate.go` 的 AutoMigrate 流程，使 IAM 表仅在 Local 模式执行；更新 `cmd/database/main.go` 以读取 resolver 结果。
- [X] T007 在 `skeleton/backend/internal/services/iam/seeder.go` 实现本地管理员种子（依赖 `PLUGIN_IAM_ADMIN_*`），并在 `cmd/database/main.go setup` 中强制校验/失败。
- [X] T008 实现/更新 `skeleton/backend/internal/transport/http/middleware/auth_jwt.go`（及签名上下文相关文件），确保所有受保护路由同时支持 Bearer Token 与 `X-PowerX-CTX`，并添加 Go 测试覆盖 JWT 与 Signed-Context 流程。

**Checkpoint**：IAM 接口 + Resolver + 中间件栈就绪。

---

## Phase 3: User Story 1 – Delegated Login Flow (Priority P1) 🎯

**Goal**：宿主模式下一次性登录、自动刷新、路由守卫、manifest 权限声明。

**Independent Test**：设置 `POWERX_PROXY=1`，运行 Playwright 用例验证登录→访问受保护页→token 过期刷新；停掉 Core 触发 fail-closed。

### Implementation

- [ ] T009 [US1] 在 `skeleton/web-admin/app/composables/useAuth.ts` 实现宿主同款的 `setAuth/clearAuth/initAuth/logout`，并加入 localStorage 失败时自动回退 cookie/强制登录的逻辑。
- [ ] T010 [P] [US1] 扩展 `skeleton/web-admin/app/composables/api/_client.ts`，注入 `Authorization` / `X-Tenant-ID`，在 401 时自动调 `authService.refreshToken` 并重放请求。
- [ ] T011 [P] [US1] 新建 `skeleton/web-admin/app/composables/api/services/authService.ts`，封装 `/auth/login|refresh|logout|me`，支持 `skipAuth` 选项。
- [ ] T012 [US1] 添加 `skeleton/web-admin/app/middleware/auth.global.ts`，保护除 `/users/*` 外的所有路由并处理 `redirect` query。
- [ ] T013 [US1] 完成 `skeleton/web-admin/app/pages/users/login.vue`（复用 register/forgot 布局），调用 `useAuth` + `useAuthService`。
- [ ] T014 [P] [US1] 注册 `skeleton/web-admin/app/plugins/auth.client.ts` 在客户端初始化 `initAuth`，并在全局导航（如 `app/components/AppNavbar.vue`）加入 Logout。
- [ ] T015 [P] [US1] 为 `useAuth` 新增单元/组件测试（例如 `skeleton/web-admin/tests/unit/useAuth.fallback.spec.ts`），覆盖 localStorage 访问失败时的 cookie 回退/强制登录行为。
- [ ] T016 [US1] 在 `skeleton/backend/internal/services/authproxy/delegated_client.go` 调用 `POWERX_CORE_ENDPOINT` (`/admin/user/auth/*`) 并附带 `POWERX_AUTH_TOKEN`。
- [ ] T017 [US1] 新增 `skeleton/backend/internal/transport/http/public/auth_handler.go` 与路由注册 `/api/v1/auth/login|refresh|logout|me/context`，代理 Delegated client 并执行 fail-closed。
- [ ] T018 [US1] 更新 `skeleton/backend/internal/router/router.go`，挂载 public auth 路由、受保护 `me` 路由，并确认 Phase2 中间件在所有业务路由生效。
- [ ] T019 [US1] 更新 `skeleton/backend/internal/manifestx/manifest.go` 与 `docs/contracts/manifest.yaml` / `docs/contracts/rbac.schema.json`，声明插件运行所需的 `iam.user.read`/`iam.role.read` 等 scope，并同步 `scaffold/templates/**` 与 CLI manifest。
- [ ] T020 [P] [US1] 编写 Playwright E2E (`skeleton/web-admin/tests/e2e/auth-delegated.spec.ts`)，覆盖登录成功、token 刷新、Core 断连提示。
- [ ] T021 [US1] 在 `skeleton/backend/internal/transport/http/public/auth_handler_test.go` 编写 Go 测试，mock Delegated client，验证成功/401/超时。
- [ ] T022 [US1] 执行 `npm run sync:templates`，同步 Skeleton 改动到 `scaffold/templates/**` 与 `tools/cli/internal/templates/**`，并更新 `CHANGELOG.md` 记录命令。

**Checkpoint**：Delegated 模式可独立演示 + manifest 权限齐备。

---

## Phase 4: User Story 2 – Standalone Local IAM (Priority P2)

**Goal**：离线/独立模式具备 IAM 表、管理员种子、登录/登出/刷新能力。

**Independent Test**：`POWERX_PROXY=0` + `PLUGIN_IAM_ADMIN_*`，运行 `go run ./cmd/database/main.go setup` 后使用 Playwright 验证本地登录流程。

### Implementation

- [ ] T023 [US2] 实现 `skeleton/backend/internal/services/iam/local_store.go`（`Login/Refresh/Logout/GetCurrentUser/ListRoles/ListDepartments/CheckPermission`）。
- [ ] T024 [US2] 在 Resolver 注入 local store，并在 `auth_handler` 根据模式选择 Delegated 或 Local 实现；覆盖 `/auth/*` 与 `/auth/me/context`。
- [ ] T025 [P] [US2] 扩展 `skeleton/backend/cmd/database/seed/local_iam_seed.go`（或等效文件）创建示例租户/角色/权限/部门。
- [ ] T026 [US2] 新增 Playwright/E2E (`skeleton/web-admin/tests/e2e/auth-local.spec.ts`) 使用本地管理员账号验证登录/登出。
- [ ] T027 [US2] 将 Local 模式步骤与 env 说明写入 `specs/005-plugin-auth/quickstart.md` 和 `docs/guides/develop/standalone-mode.md`。
- [ ] T028 [US2] 更新 `docs/plan/004-plugin-auth-integration.md` 与 `docs/operations/runbooks/auth-troubleshooting.md`，覆盖 Local IAM、管理员注入及排障流程。

**Checkpoint**：Local 模式运行完整，并有文档说明。

---

## Phase 5: User Story 3 – Token Lifecycle & Observability (Priority P3)

**Goal**：fail-closed 提示、跨 Tab 同步、可观测指标与日志。

**Independent Test**：Delegated 模式下主动断开 Core、在多 Tab 同时登录/登出，确认 UI 提示、storage 同步、Prometheus 指标。

### Implementation

- [ ] T029 [US3] 在 `skeleton/web-admin/app/composables/useAuth.ts` 与登录页面中增加 Core 断连的错误提示（“宿主认证不可用”），并确保 UI 行为与 Clarification 一致。
- [ ] T030 [P] [US3] 在 `useAuth` 中监听 `window.addEventListener('storage')` 同步多 Tab token/logout 状态，并添加单元测试覆盖。
- [ ] T031 [US3] 在 `skeleton/backend/internal/observability/metrics/auth_metrics.go`（新文件）注册 `plugin_auth_login_total`, `plugin_auth_refresh_total`, `plugin_auth_logout_total`, `plugin_iam_mode`, `plugin_iam_delegate_errors_total`，并在 server 启动时初始化。
- [ ] T032 [US3] 强化 `skeleton/backend/internal/transport/http/middleware/request_trace.go`，记录 `auth_mode`, `tenant_id`, `user_id`, `trace_id`，并遮蔽 token。
- [ ] T033 [US3] 在 `skeleton/backend/internal/services/iam/metrics_test.go`（新文件）编写 Go 测试，验证指标累积与 fail-closed 日志触发。
- [ ] T034 [US3] 更新 `docs/operations/runbooks/auth-troubleshooting.md`，加入指标字段解释、Prometheus 告警建议与多 Tab 故障排查步骤。
- [ ] T035 [US3] 为 localStorage/cookie 双失效场景新增 Playwright 测试（扩展现有 E2E 用例），确保用户被强制跳转 `/users/login` 并看到提示。

**Checkpoint**：token 生命周期具备观测/安全保障。

---

## Phase 6: Polish & Cross-Cutting

- [ ] T036 [P] 执行 `npm run sync:templates -- --check`、`go test ./...`、`npm run lint` 并在 `CHANGELOG.md` 记录结果。
- [ ] T037 更新/新增 `docs/guides/develop/auth.md`，描述 Delegated + Local 流程、fail-closed 行为、测试步骤。
- [ ] T038 [P] 按照 `specs/005-plugin-auth/quickstart.md` 实际走 Delegated/Local 步骤，填充验证结果与已知问题。
- [ ] T039 记录性能指标：编写或运行脚本（如 `skeleton/web-admin/tests/perf/login-latency.mjs`）测量 Delegated 登录 p90/刷新成功率，并在 quickstart/README 中记下数据。
- [ ] T040 测量本地 IAM 迁移 + 种子耗时（确保 ≤60s），在 `specs/005-plugin-auth/quickstart.md` 和 `docs/operations/runbooks/auth-troubleshooting.md` 中记录方法与结果。
- [ ] T041 最终审查：确认 `AGENTS.md`、`docs/plan/004-plugin-auth-integration.md`、`spec.md`、`tasks.md` 均与实现一致，清理 TODO/占位符。

---

## Dependencies & Execution Order

1. Phase 1 → Phase 2：完成基础配置后再构建 IAM 架构。
2. Phase 2 → US1/US2/US3：IAM 接口、resolver、中间件就绪方可开发具体场景。
3. US1 完成后 US2/US3 才能依托其公共组件；manifest/RBAC 更新在 US1。
4. US2 依赖 Local store/迁移；US3 依赖 US1 的 token 生命周期钩子与 Phase5 指标基础。
5. Polish 阶段在所有目标 user story 完成后执行，输出性能/文档/验证。

## Parallel Opportunities
- 标记 [P] 的任务（如 T010、T011、T014、T019、T025、T030、T036、T038、T039）可在满足依赖后并行。
- US2 与 US3 可在 US1 核心组件合并后并行开发。

## MVP Scope
- 完成 Phase 1–2 + **User Story 1 (P1)** 即可交付可用于宿主集成的 MVP；US2/US3 可作为迭代增强。
