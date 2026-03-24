# Tasks: Async Runtime Scheduler 模式切换

**Input**: Design documents from `/specs/017-async-runtime-scheduler-switch/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, quickstart.md

**Tests**: 本特性包含明确验收与稳定性目标，任务中包含必要单测/集成测试。

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 为 Scheduler 模式切换建立实现骨架与文件布局

- [x] T001 创建 Scheduler 模式服务骨架文件 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_mode_service.go`
- [x] T002 [P] 创建重试策略服务骨架文件 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_retry_service.go`
- [x] T003 [P] 创建恢复工单服务骨架文件 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_ticket_service.go`
- [x] T004 [P] 创建 Scheduler 管理端 handler 骨架文件 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_mode_handler.go`
- [x] T005 [P] 创建 Scheduler 重试/暂停/恢复 handler 骨架文件 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_retry_handler.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 完成所有用户故事共享的阻塞能力（配置校验、路由、权限、启动接线）

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 扩展 Scheduler 配置结构（默认重试=3、范围=1-10、暂停策略/恢复角色）到 `skeleton/backend/go-gin/internal/config/operations.go`
- [x] T007 在 `skeleton/backend/go-gin/internal/config/config.go` 增加 Scheduler 配置默认值与合法性校验
- [x] T008 [P] 在 `skeleton/backend/etc/config.example.yaml` 补充 Scheduler 模式与重试闭环示例配置
- [x] T009 在 `skeleton/backend/go-gin/cmd/plugin/taskbus_provider.go` 实现 `POWERX_PROXY` 与 `taskbus_provider` 冲突校验函数
- [x] T010 在 `skeleton/backend/go-gin/cmd/plugin/main.go` 接入 fail-fast 启动前校验（冲突即启动失败）
- [x] T011 [P] 在 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/routes.go` 注册 scheduler 管理端路由
- [x] T012 [P] 在 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/rbac.go` 增加 scheduler 相关权限映射
- [x] T013 在 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/service.go` 注入 Scheduler 相关服务依赖
- [x] T014 [P] 在 `skeleton/backend/go-gin/internal/config/config_test.go` 增加 Scheduler 配置校验用例（含默认值与 1-10 边界）
- [x] T015 [P] 新增 fail-fast 启动校验测试 `skeleton/backend/go-gin/cmd/plugin/taskbus_provider_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - 自动识别运行模式并切换调度执行路径 (Priority: P1) 🎯 MVP

**Goal**: 启动时自动识别 `standalone local/delegated proxy`，并保证执行路径唯一且冲突即失败

**Independent Test**: 分别以 `POWERX_PROXY=0 + redis` 与 `POWERX_PROXY=1 + host` 启动并触发调度；再用冲突配置启动，验证前两者成功、冲突场景 fail-fast。

### Implementation for User Story 1

- [x] T016 [US1] 实现模式解析与判定逻辑到 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_mode_service.go`
- [x] T017 [US1] 实现 `POST /api/v1/admin/runtime/scheduler/mode/validate` handler 到 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_mode_handler.go`
- [x] T018 [US1] 在 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/routes.go` 绑定 mode validate 端点到具体 handler
- [x] T019 [P] [US1] 为模式判定服务新增单测 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_mode_service_test.go`
- [x] T020 [P] [US1] 为 mode validate handler 新增 HTTP 用例 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_mode_handler_test.go`
- [x] T021 [US1] 新增双模式 + 冲突场景集成测试 `skeleton/backend/go-gin/tests/integration/scheduler_mode_switch_test.go`

**Checkpoint**: User Story 1 可独立验收（模式识别成功 + 冲突 fail-fast）

---

## Phase 4: User Story 2 - 调度触发与手动触发保持同链路语义 (Priority: P2)

**Goal**: 调度触发与手动触发复用同一事件入口，保证状态流转与业务结果语义一致

**Independent Test**: 对同一标准 topic（如 `powerx.runtime.scheduler.triggered.v1`）先手动触发再调度触发，验证两者结果语义一致（允许耗时差异）。

### Implementation for User Story 2

- [x] T022 [US2] 新增调度事件分发适配器 `skeleton/backend/go-gin/internal/jobs/integration/scheduler_event_dispatcher.go`（统一调用 EventEmitter 并透传 trace_id）
- [x] T023 [US2] 在 `skeleton/backend/go-gin/cmd/plugin/main.go` 完成 scheduler 注册与启动接线（进入统一事件链路）
- [x] T024 [US2] 在 `skeleton/backend/go-gin/internal/jobs/integration/scheduler.go` 接入统一 dispatcher（禁止直接 WS publish）
- [ ] T025 [P] [US2] 新增手动/调度语义一致性与 trace_id 透传测试 `skeleton/backend/go-gin/tests/integration/scheduler_manual_cron_parity_test.go`
- [ ] T026 [P] [US2] 在 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/runtime_metrics_test.go` 增加手动/调度双触发指标一致性断言
- [ ] T027 [US2] 更新验收说明到 `docs/guides/async_runtime/scheduler/README.md`（明确一致性判定口径与标准 topic 命名）

**Checkpoint**: User Story 2 可独立验收（同 topic 双触发语义一致）

---

## Phase 5: User Story 3 - 双模式下可观测与联调流程一致 (Priority: P3)

**Goal**: 在 proxy 权限失败场景形成“有限重试 -> 超限工单 -> 暂停 -> 运维恢复”的闭环，并提供统一联调流程

**Independent Test**: 构造 proxy 权限失败，验证重试上限、工单创建、任务暂停、恢复权限边界与恢复后再触发。

### Implementation for User Story 3

- [ ] T028 [US3] 实现有限重试状态机到 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_retry_service.go`
- [ ] T029 [US3] 实现恢复工单与暂停控制到 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_ticket_service.go`
- [ ] T030 [US3] 实现 `POST /api/v1/admin/runtime/scheduler/dispatches/{dispatchId}/retry` handler 到 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_retry_handler.go`
- [ ] T031 [US3] 在 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_retry_handler.go` 实现 `POST /scheduler/dispatches/{dispatchId}/pause`
- [ ] T032 [US3] 在 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_retry_handler.go` 实现 `POST /scheduler/tickets/{ticketId}/resume`（仅 ops/admin）
- [ ] T033 [US3] 在 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_ticket_service.go` 实现恢复操作审计记录写入
- [ ] T034 [P] [US3] 新增重试超限与工单创建单测 `skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_retry_service_test.go`
- [ ] T035 [P] [US3] 新增恢复权限边界与审计记录测试 `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_retry_handler_test.go`
- [ ] T036 [US3] 新增失败闭环集成测试 `skeleton/backend/go-gin/tests/integration/scheduler_retry_recovery_flow_test.go`
- [ ] T037 [US3] 更新联调手册 `docs/guides/async_runtime/websocket/debug_playbook.md` 与 `specs/017-async-runtime-scheduler-switch/quickstart.md`（补齐失败闭环步骤）

**Checkpoint**: User Story 3 可独立验收（失败闭环完整且恢复权限正确）

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收尾、回归验证与交付对齐

- [ ] T038 [P] 同步开发文档到 `docs/plan/develop/async_runtime/schedule/scheduler-mode-switch-implementation.md`
- [ ] T039 执行并记录回归测试命令到 `specs/017-async-runtime-scheduler-switch/quickstart.md`
- [ ] T040 [P] 校对并更新 API 契约 `specs/017-async-runtime-scheduler-switch/contracts/scheduler-mode-switch.openapi.yaml`
- [ ] T041 运行全量验收并补充结论到 `specs/017-async-runtime-scheduler-switch/spec.md` 的 Clarifications/验收说明
- [ ] T042 [P] 新增 SC-003 台账模板与统计步骤到 `specs/017-async-runtime-scheduler-switch/quickstart.md`
- [ ] T043 [P] 新增 SC-004 前后 14 天对比台账模板到 `specs/017-async-runtime-scheduler-switch/quickstart.md`
- [ ] T044 生成一次基线统计示例并回填到 `specs/017-async-runtime-scheduler-switch/spec.md` 的成功指标说明

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: 可在 Phase 2 后立即开始；是 MVP 主路径
- **User Story 2 (P2)**: 建议在 US1 后实施以降低实现风险，但应保持独立验收能力
- **User Story 3 (P3)**: 建议在 US1/US2 后实施以复用主链路能力，但应保持独立验收能力

### Within Each User Story

- 先补服务能力，再接 handler/routing，再补单测与集成验证
- 每个故事完成后必须满足其 Independent Test 才可进入下一优先级

### Parallel Opportunities

- Setup 阶段：T002/T003/T004/T005 可并行
- Foundational 阶段：T008/T011/T012/T014/T015 可并行
- US1：T019 与 T020 可并行
- US2：T025 与 T026 可并行
- US3：T034 与 T035 可并行
- Polish：T038 与 T040 可并行

---

## Parallel Example: User Story 1

```bash
# 并行实现 US1 的服务与 handler 测试
Task: "T019 [US1] 模式判定服务单测 in skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_mode_service_test.go"
Task: "T020 [US1] mode validate handler 测试 in skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_mode_handler_test.go"
```

---

## Parallel Example: User Story 3

```bash
# 并行覆盖失败闭环核心测试
Task: "T034 [US3] 重试超限与工单创建单测 in skeleton/backend/go-gin/internal/services/admin/runtime_ops/scheduler_retry_service_test.go"
Task: "T035 [US3] 恢复权限边界测试 in skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/scheduler_retry_handler_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1-2
2. 完成 US1（T016-T021）
3. 立即执行 US1 独立验收（双模式成功 + 冲突 fail-fast）

### Incremental Delivery

1. MVP: US1（模式识别与切换）
2. 增量 1: US2（手动/调度同语义）
3. 增量 2: US3（失败闭环与恢复权限）
4. 最后执行 Phase 6 收尾与文档归档

### Parallel Team Strategy

1. 开发 A：模式识别与启动接线（US1）
2. 开发 B：调度分发与语义一致（US2）
3. 开发 C：失败闭环与恢复权限（US3）

---

## Notes

- [P] tasks = different files, no dependencies
- [USx] 标签确保任务可追踪到用户故事
- 每个用户故事都可独立完成并独立验收
- 任务描述已包含明确文件路径，可直接执行
