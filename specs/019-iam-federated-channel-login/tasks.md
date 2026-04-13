# Tasks: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

**Input**: Design documents from `/specs/019-iam-federated-channel-login/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/federated-login.openapi.yaml, quickstart.md  
**Tests**: 包含。spec 明确要求独立验证主链路、映射生效与风控拦截。  

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立 framework/skeleton 双层基础骨架与可读目录说明，确保后续任务可并行推进。

- [x] T001 创建 framework federated 模块说明与边界文档 in framework/backend/go/iam/federated/README.md
- [x] T002 [P] 创建 framework challenge/risk 子模块说明 in framework/backend/go/iam/federated/challenge/README.md
- [x] T003 [P] 创建 framework providers 注册约定说明 in framework/backend/go/iam/federated/providers/README.md
- [x] T004 [P] 创建 skeleton federated 装配模块说明 in skeleton/backend/go-gin/internal/services/iam/federated/README.md
- [x] T005 创建 019 回归记录模板 in tmp/019-iam-federated-channel-login-regression.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 实现所有用户故事共享的 framework 联邦登录基础能力。  
**⚠️ CRITICAL**: 此阶段完成前，不进入任何用户故事任务。

- [x] T006 定义 federated provider 接口与统一错误语义 in framework/backend/go/iam/federated/contracts/interfaces.go
- [x] T007 定义 challenge/binding/mapping/risk 核心类型 in framework/backend/go/iam/federated/contracts/types.go
- [x] T008 实现 provider registry/factory（framework 复用入口）in framework/backend/go/iam/federated/providers/registry.go
- [x] T009 [P] 实现 challenge manager（state/nonce/ttl）in framework/backend/go/iam/federated/challenge/manager.go
- [x] T010 [P] 实现 risk evaluator（expired/replay/cross-tenant/signature）in framework/backend/go/iam/federated/risk/evaluator.go
- [x] T011 在 framework bootstrap 注入 federated factory 初始化 in framework/backend/go/bootstrap/app.go
- [x] T012 [P] 增加 registry/challenge 单元测试 in framework/backend/go/iam/federated/providers/registry_test.go
- [x] T013 [P] 增加风险错误码语义测试矩阵（仅校验 code 与 message contract）in framework/backend/go/iam/federated/contracts/errors_test.go
- [x] T014 [P] 定义 federated 模型表名常量与 TableName 规范 in skeleton/backend/go-gin/internal/domain/models/model.go
- [x] T015 [P] 在 migrate 注册 federated 持久化模型并补迁移冒烟测试 in skeleton/backend/go-gin/cmd/database/migrate/migrate.go

**Checkpoint**: framework 已具备可版本化复用的联邦登录基础，skeleton 可直接装配调用。

---

## Phase 3: User Story 1 - 员工扫码登录插件系统 (Priority: P1) 🎯 MVP

**Goal**: 员工可通过任一渠道扫码登录并建立统一身份会话。  
**Independent Test**: 完成“challenge -> callback -> 登录态建立 -> 身份上下文输出”链路。

### Tests for User Story 1

- [x] T016 [P] [US1] 增加扫码 challenge 主链路测试 in framework/backend/go/iam/federated/challenge/flow_test.go
- [x] T017 [P] [US1] 增加 federated callback handler 集成测试 in skeleton/backend/go-gin/internal/transport/http/public/auth/federated_callback_test.go

### Implementation for User Story 1

- [x] T018 [US1] 实现 framework wecom provider 默认流程 in framework/backend/go/iam/federated/providers/wecom/provider.go
- [x] T019 [US1] 实现 framework dingtalk provider 默认流程 in framework/backend/go/iam/federated/providers/dingtalk/provider.go
- [x] T020 [US1] 实现 framework lark provider 默认流程 in framework/backend/go/iam/federated/providers/lark/provider.go
- [x] T021 [US1] 在 skeleton 启动流程装配 framework federated factory in skeleton/backend/go-gin/internal/bootstrap/app.go
- [x] T022 [US1] 实现 public federated challenge/callback 路由 in skeleton/backend/go-gin/internal/transport/http/public/auth/federated_handler.go
- [x] T023 [US1] 实现登录成功后统一身份上下文输出 in skeleton/backend/go-gin/internal/services/iam/federated/login_service.go
- [x] T024 [US1] 实现密码/扫码并存控制面（模式开关、故障切换优先级）in skeleton/backend/go-gin/internal/services/iam/federated/auth_mode_service.go

**Checkpoint**: US1 完成后，扫码登录 MVP 可独立演示与验收。

---

## Phase 4: User Story 2 - 渠道账号与 IAM 身份统一映射 (Priority: P2)

**Goal**: 管理员可治理绑定关系与角色/部门映射，并在下次登录生效。  
**Independent Test**: 绑定/解绑/映射变更后，成员再登录即时按新映射生效。

### Tests for User Story 2

- [x] T025 [P] [US2] 增加 identity binding CRUD 与租户隔离测试 in skeleton/backend/go-gin/internal/services/iam/federated/binding_service_test.go
- [x] T026 [P] [US2] 增加映射策略“版本变化才重算”测试 in skeleton/backend/go-gin/internal/services/iam/federated/mapping_policy_test.go
- [x] T027 [P] [US2] 增加租户级 JIT 策略开关/策略选择测试 in skeleton/backend/go-gin/internal/services/iam/federated/jit_policy_service_test.go
- [x] T028 [P] [US2] 增加解绑后历史会话失效测试 in skeleton/backend/go-gin/internal/services/iam/federated/session_invalidate_test.go

### Implementation for User Story 2

- [x] T029 [US2] 实现 external identity / binding 模型 in skeleton/backend/go-gin/internal/entity/models/iam/federated_binding.go
- [x] T030 [US2] 实现 binding 仓储与租户边界校验 in skeleton/backend/go-gin/internal/domain/repository/iam/federated_binding_repository.go
- [x] T031 [US2] 实现管理员绑定/解绑/查询 API in skeleton/backend/go-gin/internal/transport/http/admin/iam/federated_binding_handler.go
- [x] T032 [US2] 实现 JIT 服务并落地默认策略（唯一匹配自动绑定）in skeleton/backend/go-gin/internal/services/iam/federated/jit_service.go
- [x] T033 [US2] 实现租户级 JIT 开关与策略选择服务 in skeleton/backend/go-gin/internal/services/iam/federated/jit_policy_service.go
- [x] T034 [US2] 实现字段缺失身份的管理员处理与审计原因码 in skeleton/backend/go-gin/internal/services/iam/federated/jit_service.go
- [x] T035 [US2] 实现解绑触发会话失效机制 in skeleton/backend/go-gin/internal/services/iam/federated/session_service.go
- [x] T036 [US2] 实现角色/部门映射服务与版本化应用 in skeleton/backend/go-gin/internal/services/iam/federated/mapping_service.go

**Checkpoint**: US2 完成后，绑定治理和映射策略能力可独立验收。

---

## Phase 5: User Story 3 - 风控与审计可追溯 (Priority: P3)

**Goal**: 登录链路具备可追溯风控与审计能力，阻断高风险回调。  
**Independent Test**: 过期、重放、跨租户、签名异常均被拒绝并产生可检索事件。

### Tests for User Story 3

- [ ] T037 [P] [US3] 增加风险拦截场景回归测试（expired/replay/cross-tenant/signature）in framework/backend/go/iam/federated/risk/evaluator_test.go
- [ ] T038 [P] [US3] 增加审计事件字段完整性测试 in skeleton/backend/go-gin/internal/observability/auth/federated_audit_test.go
- [ ] T039 [P] [US3] 增加渠道故障降级与密码登录并存测试 in skeleton/backend/go-gin/internal/transport/http/public/auth/federated_fallback_test.go
- [ ] T040 [P] [US3] 增加 delegated 上游不可用错误语义一致性测试 in skeleton/backend/go-gin/internal/services/iam/federated/context_service_test.go

### Implementation for User Story 3

- [ ] T041 [US3] 实现 federated 审计日志与风险事件上报 in skeleton/backend/go-gin/internal/observability/auth/federated_audit.go
- [ ] T042 [US3] 在 callback 链路接入统一风险判定与错误码输出 in skeleton/backend/go-gin/internal/transport/http/public/auth/federated_handler.go
- [ ] T043 [US3] 对齐 standalone/delegated 上下文语义（delegated 宿主权威）in skeleton/backend/go-gin/internal/services/iam/federated/context_service.go

**Checkpoint**: US3 完成后，安全场景闭环且可审计。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 收口文档、联调指引、性能与业务指标验收、全量回归记录。

- [ ] T044 [P] 更新联邦登录契约文档与错误语义说明 in specs/019-iam-federated-channel-login/spec.md
- [ ] T045 [P] 更新接入与排障手册 in specs/019-iam-federated-channel-login/quickstart.md
- [ ] T046 更新实施计划术语一致性与里程碑状态 in specs/019-iam-federated-channel-login/plan.md
- [ ] T047 执行并记录联邦回调性能基准（p95 < 200ms） in tmp/019-iam-federated-channel-login-regression.md
- [ ] T048 增加 SC-004 口径埋点与统计说明（密码登录占比） in skeleton/backend/go-gin/internal/observability/auth/federated_metrics.go
- [ ] T049 增加 SC-005 接入效率度量清单（framework 复用步骤对比） in specs/019-iam-federated-channel-login/quickstart.md
- [ ] T050 执行并记录 019 全量回归 in tmp/019-iam-federated-channel-login-regression.md

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1): 无依赖，可立即开始。
- Foundational (Phase 2): 依赖 Phase 1，阻塞所有用户故事。
- User Stories (Phase 3-5): 全部依赖 Phase 2。
- Polish (Phase 6): 依赖 Phase 3-5。

### User Story Dependencies

- US1 (P1): 仅依赖 Foundational，可先做 MVP。
- US2 (P2): 依赖 US1 主链路与基础模型。
- US3 (P3): 依赖 US1/US2，收口风控与审计。

### Within Each User Story

- 先完成测试任务，再完成实现任务。
- 先完成 framework 规则能力，再完成 skeleton 装配能力。
- 每个故事完成后可独立验收，不要求一次性合并全部故事。

---

## Parallel Opportunities

- Phase 1: `T002`、`T003`、`T004` 可并行。
- Phase 2: `T009`、`T010`、`T012`、`T013`、`T014`、`T015` 可并行。
- US1: `T016` 与 `T017` 可并行；`T018`/`T019`/`T020` 可按文件并行。
- US2: `T025`、`T026`、`T027` 可并行。
- US3: `T037`、`T038`、`T039`、`T040` 可并行。
- Polish: `T044` 与 `T045` 可并行。

---

## Parallel Example: User Story 1

```bash
Task T016: framework/backend/go/iam/federated/challenge/flow_test.go
Task T017: skeleton/backend/go-gin/internal/transport/http/public/auth/federated_callback_test.go
Task T018: framework/backend/go/iam/federated/providers/wecom/provider.go
Task T019: framework/backend/go/iam/federated/providers/dingtalk/provider.go
Task T020: framework/backend/go/iam/federated/providers/lark/provider.go
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. 完成 Phase 1-2。  
2. 完成 Phase 3（US1）。  
3. 按 quickstart 验证扫码主链路独立可用。  
4. 以 framework factory 可复用能力作为 alpha 输出。

### Incremental Delivery

1. US1：先交付扫码登录主流程。  
2. US2：补齐绑定治理与映射策略。  
3. US3：收口风控审计与降级策略。  
4. Phase 6：完成文档与回归记录。
