# Tasks: Framework IAM 统一封装（Standalone/Delegated）

**Input**: Design documents from `/specs/018-framework-iam-unification/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/iam-unification.openapi.yaml, quickstart.md

**Tests**: 包含。该特性在 spec 中明确要求独立验证（模式切换、契约一致性、上下文与权限判定）。

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立实现骨架与测试入口，确保后续任务可并行推进。

- [X] T001 创建 framework IAM 模块目录与文档占位 in framework/backend/go/iam/contracts/.gitkeep
- [X] T002 [P] 创建 adapter 目录占位 in framework/backend/go/iam/adapters/.gitkeep
- [X] T003 [P] 创建 context 目录占位 in framework/backend/go/iam/context/.gitkeep
- [X] T004 [P] 创建 errors 目录占位 in framework/backend/go/iam/errors/.gitkeep
- [X] T005 增加 018 特性回归入口文档 in specs/018-framework-iam-unification/quickstart.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 实现所有用户故事共用的基础能力。  
**⚠️ CRITICAL**: 此阶段完成前，不进入任何用户故事任务。

- [X] T006 定义 IAM 统一契约接口（Directory/Authz/Context）in framework/backend/go/iam/contracts/interfaces.go
- [X] T007 定义统一数据结构（Tenant/Department/Member/Role/Permission/Decision）in framework/backend/go/iam/contracts/types.go
- [X] T008 实现模式解析器（config 优先、冲突 fail-fast）in framework/backend/go/iam/context/mode_resolver.go
- [X] T009 [P] 定义模式解析记录结构与审计字段 in framework/backend/go/iam/context/mode_resolution_record.go
- [X] T010 [P] 定义统一 IAM 错误码与错误映射 in framework/backend/go/iam/errors/errors.go
- [X] T011 实现 adapter 注册中心与启动期单选绑定 in framework/backend/go/iam/adapters/registry.go
- [X] T012 在 framework bootstrap 注入 IAM registry 初始化 in framework/backend/go/bootstrap/app.go
- [X] T013 [P] 补充模式解析与错误语义单元测试 in framework/backend/go/iam/context/mode_resolver_test.go
- [X] T014 [P] 补充 adapter 注册与单选绑定测试 in framework/backend/go/iam/adapters/registry_test.go

**Checkpoint**: Framework IAM 基础契约、模式解析、错误语义、adapter 注册机制已可用。

---

## Phase 3: User Story 1 - 插件无感切换 IAM 模式 (Priority: P1) 🎯 MVP

**Goal**: 业务层仅依赖 framework IAM 接口即可在 local/delegated 间切换，且冲突 fail-fast。  
**Independent Test**: 同一业务接口在两种模式启动并验证鉴权/上下文语义一致，业务 handler 无分支。

### Tests for User Story 1

- [X] T015 [P] [US1] 增加双模式切换集成测试 in skeleton/backend/go-gin/internal/bootstrap/iam_resolver_test.go
- [X] T016 [P] [US1] 增加管理端模式查询接口测试 in skeleton/backend/go-gin/internal/transport/http/admin/iam/routes_test.go

### Implementation for User Story 1

- [X] T017 [US1] 实现 local adapter 对 framework 契约的绑定 in skeleton/backend/go-gin/internal/services/iam/adapters/local/adapter.go
- [X] T018 [US1] 实现 delegated adapter 对 framework 契约的绑定 in skeleton/backend/go-gin/internal/services/iam/adapters/delegated/adapter.go
- [X] T019 [US1] 在 skeleton 启动流程接入 framework IAM registry 绑定 in skeleton/backend/go-gin/internal/bootstrap/app.go
- [X] T020 [US1] 改造 IAM 模式解析入口使用 framework resolver in skeleton/backend/go-gin/internal/bootstrap/iam_resolver.go
- [X] T021 [US1] 改造 admin IAM 路由以读取统一 mode/context 能力 in skeleton/backend/go-gin/internal/transport/http/admin/iam/routes.go
- [X] T022 [US1] 在 router 装配中替换业务侧 IAM 依赖为 framework contracts in skeleton/backend/go-gin/internal/router/router.go

**Checkpoint**: US1 完成后，插件可无改业务 handler 在 local/delegated 间切换。

---

## Phase 4: User Story 2 - 组织架构与权限能力统一暴露 (Priority: P2)

**Goal**: tenant/department/member/role/permission 通过统一契约暴露，delegated 写操作受限。  
**Independent Test**: 两种模式执行组织查询与授权操作，返回结构、状态码、错误语义一致。

### Tests for User Story 2

- [X] T023 [P] [US2] 增加 IAM 契约一致性测试（组织查询）in framework/backend/go/iam/contracts/directory_contract_test.go
- [X] T024 [P] [US2] 增加 delegated 写操作拒绝测试（405）in skeleton/backend/go-gin/internal/transport/http/admin/iam/department_handler_test.go
- [X] T025 [P] [US2] 增加角色授权与成员绑定语义测试 in skeleton/backend/go-gin/internal/services/iam/role_service_test.go

### Implementation for User Story 2

- [X] T026 [US2] 补齐 framework DirectoryService 契约实现入口 in framework/backend/go/iam/contracts/directory_service.go
- [X] T027 [US2] 在 local adapter 映射 tenant/department/member/role/permission 查询能力 in skeleton/backend/go-gin/internal/services/iam/adapters/local/directory_adapter.go
- [X] T028 [US2] 在 delegated adapter 映射组织只读与写拒绝策略 in skeleton/backend/go-gin/internal/services/iam/adapters/delegated/directory_adapter.go
- [X] T029 [US2] 统一 admin IAM handlers 使用 framework Directory/Authz 契约 in skeleton/backend/go-gin/internal/transport/http/admin/iam/tenant_handler.go
- [X] T030 [US2] 统一 RBAC 资源动作映射与错误输出 in skeleton/backend/go-gin/internal/transport/http/admin/iam/rbac.go

**Checkpoint**: US2 完成后，组织与权限能力在双模式下具备一致契约与可审计行为。

---

## Phase 5: User Story 3 - Token 与上下文解析规则统一 (Priority: P3)

**Goal**: framework 统一 token/context 解析、授权判定与审计字段，避免插件实现漂移。  
**Independent Test**: 注入合法/非法 token 与多来源 tenant 上下文，验证解析优先级、错误码与审计字段一致。

### Tests for User Story 3

- [X] T031 [P] [US3] 增加 identity context 解析测试（tenant/user/roles/permissions）in framework/backend/go/iam/context/identity_context_test.go
- [X] T032 [P] [US3] 增加多来源 tenant 冲突优先级测试 in framework/backend/go/middleware/tenant_context_test.go
- [X] T033 [P] [US3] 增加统一错误语义回归测试（401/403/424）in skeleton/backend/go-gin/internal/transport/http/admin/iam/tenant_handler_test.go

### Implementation for User Story 3

- [X] T034 [US3] 实现 framework identity context 解析器 in framework/backend/go/iam/context/identity_context.go
- [X] T035 [US3] 在 framework auth_guard 接入统一 IAM 授权判定输出 in framework/backend/go/middleware/auth_guard.go
- [X] T036 [US3] 在 skeleton delegated 鉴权链路接入统一 context 转换 in skeleton/backend/go-gin/internal/services/authproxy/delegated_client.go
- [X] T037 [US3] 在 skeleton gateway 集成层接入统一 token 来源与审计字段 in skeleton/backend/go-gin/internal/integrations/gateway/client.go
- [X] T038 [US3] 在 runtime 日志中补充 mode/tenant/user/permission/trace 字段 in skeleton/backend/go-gin/internal/logger/runtime.go

**Checkpoint**: US3 完成后，身份上下文与鉴权判定语义在 framework 层统一闭环。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 收口迁移说明、契约文档和全量验证。

- [X] T039 [P] 更新 IAM 统一契约 OpenAPI 细节（错误码与只读边界）in specs/018-framework-iam-unification/contracts/iam-unification.openapi.yaml
- [X] T040 [P] 更新迁移说明与对插件开发者接入指引 in specs/018-framework-iam-unification/quickstart.md
- [X] T041 更新实施计划中的结构描述与术语一致性 in specs/018-framework-iam-unification/plan.md
- [X] T042 执行并记录 018 回归结果 in tmp/018-framework-iam-unification-regression.md

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1): 无依赖，可立即开始。
- Foundational (Phase 2): 依赖 Phase 1，阻塞所有用户故事。
- User Stories (Phase 3-5): 全部依赖 Phase 2 完成。
- Polish (Phase 6): 依赖 Phase 3-5 完成。

### User Story Dependencies

- US1 (P1): 仅依赖 Foundational，可先做 MVP。
- US2 (P2): 依赖 Foundational；可复用 US1 的 adapter 绑定结果，但可独立验收。
- US3 (P3): 依赖 Foundational；与 US2 可并行开发，但集成时需合并统一错误语义。

### Within Each User Story

- 先完成测试任务，再完成实现任务。
- 先完成 contracts/context，再接入 handlers/router。
- 完成 story 后执行其独立验收再进入下一优先级。

---

## Parallel Opportunities

- Phase 1: `T002`、`T003`、`T004` 可并行。
- Phase 2: `T009`、`T010`、`T013`、`T014` 可并行。
- US1: `T015` 与 `T016` 可并行。
- US2: `T023`、`T024`、`T025` 可并行。
- US3: `T031`、`T032`、`T033` 可并行。
- Polish: `T039` 与 `T040` 可并行。

---

## Parallel Example: User Story 2

```bash
Task T023: framework/backend/go/iam/contracts/directory_contract_test.go
Task T024: skeleton/backend/go-gin/internal/transport/http/admin/iam/department_handler_test.go
Task T025: skeleton/backend/go-gin/internal/services/iam/role_service_test.go
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. 完成 Phase 1-2。
2. 完成 Phase 3（US1）。
3. 按 spec 的独立测试验证双模式无感切换。
4. 通过后先发布 framework IAM 契约 alpha 版本给插件侧试接入。

### Incremental Delivery

1. US1 上线：先解决模式切换与统一入口。
2. US2 上线：补齐组织与权限统一能力。
3. US3 上线：完成 token/context/审计一致化。
4. 最后执行 Phase 6 文档与回归收口。
