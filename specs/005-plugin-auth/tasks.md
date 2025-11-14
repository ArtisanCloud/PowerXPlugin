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

- [X] T009 [US1] 在 `skeleton/web-admin/app/composables/useAuth.ts` 实现宿主同款的 `setAuth/clearAuth/initAuth/logout`，并加入 localStorage 失败时自动回退 cookie/强制登录的逻辑。
- [X] T010 [P] [US1] 扩展 `skeleton/web-admin/app/composables/api/_client.ts`，注入 `Authorization` / `X-Tenant-ID`，在 401 时自动调 `authService.refreshToken` 并重放请求。
- [X] T011 [P] [US1] 新建 `skeleton/web-admin/app/composables/api/services/authService.ts`，封装 `/auth/login|refresh|logout|me`，支持 `skipAuth` 选项。
- [X] T012 [US1] 添加 `skeleton/web-admin/app/middleware/auth.global.ts`，保护除 `/users/*` 外的所有路由并处理 `redirect` query。
- [X] T013 [US1] 完成 `skeleton/web-admin/app/pages/users/login.vue`（复用 register/forgot 布局），调用 `useAuth` + `useAuthService`。
- [X] T014 [P] [US1] 注册 `skeleton/web-admin/app/plugins/auth.client.ts` 在客户端初始化 `initAuth`，并在全局导航（如 `app/components/AppNavbar.vue`）加入 Logout。
- [X] T015 [P] [US1] 为 `useAuth` 新增单元/组件测试（`skeleton/web-admin/tests/unit/useAuth.fallback.spec.ts` + `vitest.config.ts`），覆盖 localStorage 回退与强制登录行为。
- [X] T016 [US1] 在 `skeleton/backend/internal/services/authproxy/delegated_client.go` 实现对宿主 `/admin/user/auth/*` 的代理并附带 `POWERX_AUTH_TOKEN`。
- [X] T017 [US1] 新增 `skeleton/backend/internal/transport/http/public/auth_handler.go` 与公共路由 `/api/v1/auth/login|refresh|logout|me/context`，完成 fail-closed 代理。
- [X] T018 [US1] 更新 `skeleton/backend/internal/router/router.go`，挂载 public auth 路由并沿用 Phase2 中间件；`shared/app/deps.go`、`cmd/plugin/main.go` 注入依赖。
- [X] T019 [US1] 更新 `skeleton/backend/internal/manifestx/manifest.go`、`docs/contracts/manifest.yaml`、`docs/contracts/rbac.schema.json` 及模板/CLI，声明所需 IAM scope。
- [X] T020 [P] [US1] 编写 Playwright E2E `skeleton/web-admin/tests/e2e/auth-delegated.spec.ts`，覆盖登录成功与 Core 不可用提示。
- [X] T021 [US1] 在 `skeleton/backend/internal/transport/http/public/auth_handler_test.go` 添加 Go 测试，mock Delegated client 覆盖成功/401/503 分支。
- [X] T022 [US1] 执行 `npm run sync:templates`，同步 Skeleton → `scaffold/templates/**` → CLI，并在 `CHANGELOG.md` 记录。

**Checkpoint**：Delegated 模式可独立演示 + manifest 权限齐备。

---

## Phase 4: User Story 2 – Standalone Local IAM (Priority P2)

**Goal**：离线/独立模式具备 IAM 表、管理员种子、登录/登出/刷新能力。

**Independent Test**：`POWERX_PROXY=0` + `PLUGIN_IAM_ADMIN_*`，运行 `go run ./cmd/database/main.go setup` 后使用 Playwright 验证本地登录流程。

### Implementation

- [X] T023 [US2] 实现 `skeleton/backend/internal/services/iam/local_store.go`，涵盖 Login/Refresh/Logout、JWT 签发、RefreshToken 存储、角色/权限查询与 `UserContextFromToken`。
- [X] T024 [US2] Resolver 注入 Local store（`cmd/plugin/main.go` + `internal/shared/app/deps.go`），`public/auth_handler.go` 根据 IAM Mode 切换 Delegated/Local，并复用新 store。
- [X] T025 [P] [US2] 扩展 `internal/services/iam/seeder.go` 新增默认部门、权限与 role-permission 绑定。
- [X] T026 [US2] 新增 Playwright E2E `skeleton/web-admin/tests/e2e/auth-local.spec.ts`（可通过 `PLAYWRIGHT_LOCAL_IAM=1` 驱动本地管理员登录）。
- [X] T027 [US2] 更新 `specs/005-plugin-auth/quickstart.md` 与 `docs/guides/develop/standalone-mode.md`，记录 Local IAM 环境变量、登录步骤及 E2E 验证方式。
- [X] T028 [US2] 在 `docs/plan/004-plugin-auth-integration.md` 补充 Local 模式说明，并新增 `docs/operations/runbooks/auth-troubleshooting.md` 记录 Delegated/Local 排障步骤。

**Checkpoint**：Local 模式运行完整，并有文档说明。

---

## Phase 5: User Story 3 – Token Lifecycle & Observability (Priority P3)

**Goal**：fail-closed 提示、跨 Tab 同步、可观测指标与日志。

**Independent Test**：Delegated 模式下主动断开 Core、在多 Tab 同时登录/登出，确认 UI 提示、storage 同步、Prometheus 指标。

### Implementation

- [X] T029 [US3] 在 `app/composables/useAuth.ts` / `/users/login.vue` 增强 fail-closed 提示，503/refresh 失败会存储 “宿主认证不可用” 并在登录页读取展示。
- [X] T030 [P] [US3] `useAuth` 的 storage 事件现同步 token&强制跳登录，新增 Vitest 覆盖 storage 事件与错误消费。
- [X] T031 [US3] 新增 `internal/observability/auth/metrics.go`，记录 login/refresh/logout/iam_mode/delegate_errors，并在 `cmd/plugin/main.go` 初始化；Prometheus 输出合并在 `/api/v1/admin/runtime/metrics`。
- [X] T032 [US3] `request_trace` 现日志 auth_mode/tenant_id/user_id/trace_id，便于跨模式排障。
- [X] T033 [US3] `internal/observability/auth/metrics_test.go` 验证指标累积；`go test` 覆盖对应输出。
- [X] T034 [US3] 更新 `docs/operations/runbooks/auth-troubleshooting.md` / `docs/plan/004-plugin-auth-integration.md`，记录指标、Fail-Closed、多 Tab 行为。
- [X] T035 [US3] Playwright `auth-delegated.spec.ts` 新增 storage 同步测试，模拟本地储存失效后自动跳转 `/users/login`。

**Checkpoint**：token 生命周期具备观测/安全保障。

---

## Phase 6: Polish & Cross-Cutting

- [X] T036 [P] 执行 `npm run lint`、`cd skeleton/backend && go test ./...`、`npm --prefix skeleton/web-admin run test:unit`、`npm run sync:templates -- --check`，并在 `CHANGELOG.md` “Added/Changed” 项中记录观测增强。
- [X] T037 新增 `docs/guides/develop/auth.md`，覆盖 Delegated/Local 流程、Token 行为、指标与排障步骤。
- [X] T038 [P] 按照 Quickstart 重新梳理 Delegated/Local 验证，并在 `specs/005-plugin-auth/quickstart.md` 补充验收命令、Playwright 步骤与性能参考。
- [X] T039 记录 Delegated 登录性能及刷新成功率（基于 Playwright Script），结果写入 `docs/guides/develop/auth.md#6` / Quickstart 末尾。
- [X] T040 在同一章节描述本地 IAM 迁移/种子耗时和 Postgres 推荐做法，同时在 runbook 中强调 SQLite 限制。
- [X] T041 审查 `docs/plan/004-plugin-auth-integration.md`、`spec.md`、`tasks.md`，完成 Phase5/6 状态更新并清理 TODO。

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
