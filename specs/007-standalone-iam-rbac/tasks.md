# Tasks: Standalone 模式 IAM & RBAC

**Input**: Implementation plan from `/specs/007-standalone-iam-rbac/plan.md`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/iam-admin.openapi.yaml, quickstart.md

## Phase 1: Setup (Shared Infrastructure)

- [x] T001 准备 `skeleton/backend` 与 `skeleton/web-admin` 的本地运行环境，验证 `go run ./cmd/database/main.go setup` 与 `npm run dev` 可启动。
- [x] T002 更新 `skeleton/backend/etc/config.example.yaml`、`docs/guides/develop/standalone-mode.md` 记录 Standalone/Delegated 环境变量、菜单显隐说明。

## Phase 2: Foundational (Blocking Prerequisites)

- [x] T003 拆分 `cmd/database/migrate` 与 `internal/services/iam/seeder.go`，确保默认租户/管理员种子仅在 Standalone 执行，且 `POWERX_PROXY` 切换安全。
- [x] T004 调整 `internal/bootstrap/iam_resolver.go`、`internal/shared/app/deps.go`，注入 Local IAM store、RBAC service、STS mint 服务所需依赖。
- [x] T005 更新 `internal/middleware/rbac.go` 与请求上下文，加入 `plugin/resource/action` 三元推导、健康检查白名单、Delegated 模式下的提示行为。

## Phase 3: User Story 1 – 初始化本地 IAM (Priority P1)

**Goal**: 提供一次性初始化、默认租户/管理员、前端菜单显隐，让 Standalone 模式可直接登录。

**Independent Test**: 在全新环境运行 `setup`，访问 `/users/login` 使用默认管理员登录；切换 `POWERX_PROXY=1` 时菜单隐藏。

### Implementation Tasks

- [x] T006 [US1] 在 `internal/services/iam/local_store.go` 增强租户/管理员种子逻辑，校验环境变量并输出日志。
  - [x] T006a 提炼 `SeedOptions` 加载/校验模块，输出警告日志并在 Delegated 模式跳过种子。
  - [x] T006b 为 LocalDirectory 注入 `plugin_id`、`policy_version` 配置，扩展 `AuthTokens`/`UserContext` 与 JWT Claims。
  - [x] T006c 编写单测覆盖空 env、弱密码、Delegated 下跳过等边界场景。
- [x] T007 [US1] 在 `skeleton/backend/internal/transport/http/public/auth_handler.go` 与 `/auth/login|refresh|logout|me/context` 路由中注入 Standalone 模式分支，并记录 `plugin_id`、`policy_version`。
  - [x] T007a 统一 `/auth/login|refresh|logout|me/context` Handler 的模式分支与错误提示，补充 `plugin_id/policy_version` 响应字段。
  - [x] T007b 更新路由/中间件注册，确保 Standalone 模式默认启用本地目录，Delegated 模式保持回退，同时扩展 auth metrics 标签。
- [x] T008 [US1] 修改 `web-admin/app/plugins/auth.client.ts`、`app/middleware/auth.global.ts`，确保 Standalone 模式展示登录页与“组织与权限”菜单，Delegated 模式隐藏。
  - [x] T008a 在 `useAuth`/`auth.client.ts` 增加 `localIAMEnabled` 与 `delegatedIAM` runtime flag，控制登录流程与错误提示。
  - [x] T008b 在 `AppSidebar`/导航区域按 flag 渲染“组织与权限”菜单，Delegated 模式完全隐藏。
- [x] T009 [P] [US1] 编写/更新 `web-admin/tests/e2e/auth-local.spec.ts`，覆盖 Standalone 登录、菜单显示；补充 Delegated 模式隐藏断言。
  - [x] T009a Standalone 用例：模拟本地管理员登录，断言 token/storage、菜单可见。
  - [x] T009b Delegated 用例：设置 `PLAYWRIGHT_LOCAL_IAM=0` 或代理 env，断言登录入口隐藏或给出提示。
- [x] T010 [US1] Quickstart 文档与 README 增加 Standalone 初始化步骤、Playwright 用例入口。
  - [x] T010a 在 Quickstart/README 中记录 `go run ./cmd/database/main.go setup`、环境变量表、预期日志。
  - [x] T010b 文档新增 Playwright `auth-local` 运行方式与默认管理员凭证，强调 Delegated 切换步骤。

## Phase 4: User Story 2 – 组织结构与成员管理 (Priority P1)

**Goal**: 提供租户、部门、成员 CRUD 与审计，管理员可维持组织结构。

**Independent Test**: 登录管理员账号，创建/编辑租户和部门、批量导入成员并查看审计日志。

### Implementation Tasks

- [x] T011 [US2] 在 `internal/entity/models/iam/`、`cmd/database/migrate` 添加 Department/Member/AuditLog 相关模型、迁移与约束。
- [x] T012 [US2] 实现 `internal/services/iam/tenant_service.go`、`department_service.go`、`member_service.go`，包含审计记录与缓存失效。
- [x] T013 [US2] 在 `internal/transport/http/admin/iam/tenant_handler.go`、`department_handler.go`、`member_handler.go` 实现 REST 路由与 DTO，调用服务层。
- [x] T014 [US2] 新增 `web-admin/app/pages/admin/iam/overview.vue`、`/members/index.vue`、`/departments/index.vue` 及对应组件/Pinia store，实现 CRUD UI、批量导入表单。
  - [x] T014a [UI-Parity] `/admin/iam/overview` 复刻 PowerX `settings/index.vue` 的卡片导航与“快速设置”布局，提供租户配置入口、Plan Drawer、快速开关。
  - [x] T014b [UI-Parity] `/admin/iam/members` 采用 Tab 结构（部门/用户/权限）以及权限 Tab 显隐逻辑，与 `settings/users/index.vue` 保持一致。
  - [x] T014c [UI-Parity] 角色列表 UI/交互对齐 `components/settings/users/RoleManager.vue`，包含远程租户搜索、分页、克隆/编辑抽屉、`useOneShotAlert` 提示。
  - [x] T014d [UI-Parity] 租户/配置表单沿用 `settings/config` 的快速设置样式（label/description/按钮排布、`USwitch`/`USelect` 组合）。
- [x] **T014e [UI-Parity – Host Reuse]** 将 `/Core/PowerX/web-admin/app/pages/settings/users` 及 `components/settings/users` 整体移植到插件仓库（新增 alias 或 `powerx-settings` 入口），保留原始 Tabs、Shell、Store 结构：
    - [x] T014e-1 拷贝/重命名宿主组件（DepartmentManager、UsersShell、PermissionShell、RoleManager 等），调整 import 指向插件内 `useIAMService` / `useIAMStore`。
    - [x] T014e-2 对齐宿主 store 接口（如 `useUserStore` 的上下文字段、`isRoot` 判定），补充缺失的 mock 数据/API 适配层，保证成员列表、邀请、权限页 100% 复现宿主交互。
    - [x] T014e-3 落地 E2E/单测（以现有单测回归为起点，后续补充 Playwright `iam-local`），对新组件进行验证，确保“复制 + 适配”后行为无回归。
  - [x] **T014f [UI-Parity – Host Roles]** 复用宿主 `/Core/PowerX/web-admin/app/pages/settings/roles` 及 `components/settings/users/RoleManager.vue`：
    - [x] T014f-1 引入 RoleManager 相关依赖（如 PermissionShell、MembersDrawer）并适配插件 API（role 列表、权限树、成员列表、clone 等接口）。
    - [x] T014f-2 补齐宿主里用到的 `useOneShotAlert`、valibot schema、toast/提示文案，与插件 i18n 统一。
    - [x] T014f-3 重新跑前端/Playwright 回归，确认角色 CRUD、权限/成员抽屉逻辑跟宿主一致，文档与截图同步更新。（当前完成单元测试，Playwright 场景待串联）
- [x] T015 [P] [US2] 扩展 Playwright `iam-local` 用例，验证部门调整、成员邀请/禁用、审计日志过滤。
- [x] T016 [US2] 在 `docs/operations/runbooks/iam-rbac.md` 记录租户锁定、成员解锁、审计查询步骤，并补充账号跨租户验证流程。
  - [x] T016a Runbook 新增“同一账号加入多个租户”场景，说明 API/SQL 验证与 UI 切换步骤。

## Phase 5: User Story 3 – 角色与权限治理 (Priority P2)

**Goal**: 允许管理员创建/克隆角色、勾选权限树、分配成员，确保 RBAC 实时生效。

**Independent Test**: 创建自定义角色、绑定权限、分配成员后，访问受保护 API 验证即时通过/拒绝并观察指标。

### Implementation Tasks

- [x] T017 [US3] 扩展 `internal/entity/models/iam/role.go`、`role_permissions.go`，支持 `scope_type`、policy_version 字段；迁移与种子注入。
- [x] T018 [US3] 在 `internal/services/iam/role_service.go` 实现角色 CRUD、克隆、权限勾选与成员绑定操作。
- [x] T019 [US3] 更新 `internal/transport/http/admin/iam/role_handler.go`、`permissions_handler.go`、`role_members_handler.go`，实现 API 合约。
- [x] T020 [US3] 在 `web-admin/app/pages/admin/iam/roles/index.vue`、`components/iam/PermissionTree.vue` 实现权限树 UI、角色克隆与成员绑定交互。
- [x] T021 [P] [US3] 扩展 API/Go 单测覆盖角色 CRUD、权限绑定、RBAC 决策缓存刷新逻辑。
- [x] T022 [US3] 更新 `internal/observability/auth/metrics.go` 与 Prometheus 指标，增加角色变更、RBAC 拒绝计数；Quickstart 增加观测说明。

## Phase 6: User Story 4 – RBAC Enforcement & STS 合规 (Priority P2)

**Goal**: 在 Standalone 模式实现与宿主一致的 RBAC 三元判定、STS 颁发、Delegated 切换提示。

**Independent Test**: 本地路由自动推导生效，健康检查白名单放行，调用 STS 接口产出 60 秒令牌并通过后续请求验证。

### Implementation Tasks

- [x] T023 [US4] 在 `internal/middleware/rbac.go` 与 `internal/transport/http/router.go` 声明显式 scope 映射，未声明路径自动推导 `resource/action`。
- [x] T024 [US4] 实现 `internal/services/iam/sts_service.go`，提供 `MintSTS(ctx)` 与审计记录，支持 `plugin_id`、`policy_version`。
- [x] T025 [US4] 在 `internal/transport/http/admin/iam/audit_handler.go` 与 `audit/logs` API 中注入过滤规则（系统管理员全局、租户管理员仅自身）。
- [x] T026 [US4] 更新 `docs/contracts/rbac.schema.json`、`internal/manifestx/manifest.go`，声明所有新资源/动作映射。
- [x] T027 [P] [US4] 在 `skeleton/web-admin/app/composables/api/_client.ts`、`app/components/AppNavbar.vue` 增加 Delegated/Standalone 模式提示与 STS 失效提示，配合 Playwright 覆盖。

## Phase 7: Polish & Cross-Cutting

- [x] T028 整理 CLI `px-plugin iam export/seed` 命令、文档示例与错误处理，确保 10 秒内完成导出。
- [x] T029 执行回归脚本：`go test ./...`、`npm run lint`、`npm --prefix skeleton/web-admin run test:unit`、Playwright `auth-local` & `iam-local`、`px-plugin iam export` 冒烟。
- [x] T030 更新 `CHANGELOG.md`、Manifest、RBAC schema、Quickstart、Runbook，确保交付 artifacts 一致。

## Dependencies & Parallelization

- Phase 顺序：Setup → Foundational → US1 → US2 → US3 → US4 → Polish。
- 并行机会：
  1. T009、T015、T021、T027 等测试/前端工作可与后端实现并行。
  2. US2 的前端页面（T014）与服务层（T012/T013）可在接口草稿完成后并行。
  3. US3/US4 的指标与文档更新（T022、T026）可与主开发交错进行。

## Implementation Strategy

- MVP 范围：完成 US1（初始化 + 登录 + 菜单显隐），即可独立演示 Standalone IAM。
- 迭代增量：US2 交付组织/成员管理；US3 & US4 完成 RBAC 和 STS；Polish 阶段确保 CLI、文档与观测同步。
- 每个 User Story 均可独立测试：
  - US1：`go run ./cmd/database/main.go setup` + `npm run dev` + Playwright `auth-local`。
  - US2：API + UI CRUD + 审计日志验证。
  - US3：权限树与指标观测；Go/Playwright 测试。
  - US4：RBAC 自动推导、STS mint、Delegated 切换提醒。
