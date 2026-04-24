# Tasks: Framework 统一日志适配

**Input**: Design documents from `/specs/020-framework-logger/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/framework-logger.openapi.yaml, quickstart.md  
**Tests**: 包含。spec/plan 明确要求多 sink、降级重试、宿主/独立模式回归验证。  

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立日志特性实施骨架与任务落地入口。

- [x] T001 创建 020 回归记录模板 in tmp/020-framework-logger-regression.md
- [x] T002 [P] 创建 framework logging 目录说明 in framework/backend/go/runtime/common/logging/README.md
- [x] T003 [P] 创建 skeleton logging 接线说明 in skeleton/backend/go-gin/internal/logger/README.md
- [x] T004 创建 020 契约与验证索引说明 in specs/020-framework-logger/contracts/README.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 实现所有用户故事共享的日志门面、策略加载与路由基础能力。  
**⚠️ CRITICAL**: 此阶段完成前，不进入任何用户故事实现。

- [x] T005 实现 framework 日志门面接口（FromContext/With/Emit）in framework/backend/go/runtime/common/logging/facade.go
- [x] T006 [P] 实现日志策略模型与校验规则 in framework/backend/go/runtime/common/logging/policy.go
- [x] T007 [P] 实现 sink 路由器与 fan-out 执行器 in framework/backend/go/runtime/common/logging/router.go
- [x] T008 [P] 实现 stdout/file/loki sink 抽象与注册机制 in framework/backend/go/runtime/common/logging/sinks.go
- [x] T009 实现宿主模式默认策略决议（`POWERX_PROXY=1 -> stdout+json`）in framework/backend/go/runtime/common/logging/host_mode.go
- [x] T010 [P] 将 skeleton logger 初始化接入 framework 策略决议 in skeleton/backend/go-gin/internal/bootstrap/app.go
- [x] T011 [P] 增加 foundational 单元测试（策略校验、默认决议、sink 注册）in framework/backend/go/runtime/common/logging/policy_test.go

**Checkpoint**: framework 日志基础可用，插件可通过统一门面写日志并按策略路由。

---

## Phase 3: User Story 1 - 宿主模式统一采集 (Priority: P1) 🎯 MVP

**Goal**: 插件在宿主模式下默认走 `stdout+json`，并具备统一上下文字段与链路追踪能力。  
**Independent Test**: 在宿主模式触发日志，检索到统一字段并可用 `trace_id` 串联。

### Tests for User Story 1

- [x] T012 [P] [US1] 增加宿主模式默认 stdout/json 回归测试 in skeleton/backend/go-gin/internal/logger/logger_output_test.go
- [x] T013 [P] [US1] 增加统一字段注入测试（plugin_id/tenant_uuid/component/level/trace_id）in skeleton/backend/go-gin/internal/logger/runtime_test.go
- [x] T014 [P] [US1] 增加 trace 缺失自动补齐测试 in framework/backend/go/runtime/common/logging/facade_test.go

### Implementation for User Story 1

- [x] T015 [US1] 实现 framework 门面的上下文字段归并逻辑 in framework/backend/go/runtime/common/logging/context_fields.go
- [x] T016 [US1] 在 skeleton `Deps.RuntimeLogger` 对齐 framework 门面字段模型 in skeleton/backend/go-gin/internal/shared/app/deps.go
- [x] T017 [US1] 在 HTTP 中间件接入统一 trace 字段透传 in skeleton/backend/go-gin/internal/middleware/common.go
- [x] T018 [US1] 实现宿主模式下 file/loki 未授权拒绝与告警 in framework/backend/go/runtime/common/logging/router.go
- [x] T019 [US1] 补充宿主模式策略文档与示例配置 in skeleton/backend/etc/config.example.yaml
- [x] T020 [US1] 记录 US1 验收结果 in tmp/020-framework-logger-regression.md

**Checkpoint**: US1 完成后，宿主模式统一采集能力可独立演示。

---

## Phase 4: User Story 2 - 多日志源并行路由 (Priority: P2)

**Goal**: 支持多 sink 并行输出，单 sink 故障不阻断主链路，并具备告警与重试。  
**Independent Test**: 配置多 sink 后触发日志，单 sink 故障时其他 sink 仍成功且有失败记录。

### Tests for User Story 2

- [x] T021 [P] [US2] 增加多 sink fan-out 成功路径测试 in framework/backend/go/runtime/common/logging/router_test.go
- [x] T022 [P] [US2] 增加单 sink 故障降级与重试测试 in framework/backend/go/runtime/common/logging/retry_test.go
- [x] T023 [P] [US2] 增加 probe 契约测试 in skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_probe_handler_test.go

### Implementation for User Story 2

- [x] T024 [US2] 实现 sink 级重试策略与退避参数应用 in framework/backend/go/runtime/common/logging/retry.go
- [x] T025 [US2] 实现路由结果结构化输出（success/failed/retrying/dropped）in framework/backend/go/runtime/common/logging/outcome.go
- [x] T026 [US2] 实现策略查询/应用管理端接口 in skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_policy_handler.go
- [x] T027 [US2] 实现 probe 接口并输出 sink outcomes in skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_probe_handler.go
- [x] T028 [US2] 记录 US2 多 sink 回归结果 in tmp/020-framework-logger-regression.md

**Checkpoint**: US2 完成后，多 sink 并行与故障隔离能力可独立验收。

---

## Phase 5: User Story 3 - 本地调试与迁移治理 (Priority: P3)

**Goal**: 在独立模式支持策略切换，并完成直写日志分阶段治理（告警->阻断）。  
**Independent Test**: 不改业务代码切换策略可生效；截止版本后直写违规可阻断。

### Tests for User Story 3

- [x] T029 [P] [US3] 增加 standalone 策略切换回归测试 in skeleton/backend/go-gin/internal/config/config_test.go
- [x] T030 [P] [US3] 增加直写日志违规扫描规则测试 in scripts/testing/framework_logger_guard_test.sh
- [x] T031 [P] [US3] 增加治理截止版本阻断测试 in skeleton/backend/go-gin/internal/logger/governance_test.go

### Implementation for User Story 3

- [x] T032 [US3] 实现直写日志违规扫描脚本与状态输出 in scripts/testing/framework-logger-guard.sh
- [x] T033 [US3] 实现治理状态模型（detected/warned/blocked/resolved）in skeleton/backend/go-gin/internal/logger/governance.go
- [x] T034 [US3] 实现截止版本规则读取与阻断逻辑 in skeleton/backend/go-gin/internal/config/config.go
- [x] T035 [US3] 在 CI 脚本接入违规扫描步骤 in scripts/testing/regression.sh
- [x] T036 [US3] 记录 US3 治理回归结果 in tmp/020-framework-logger-regression.md

**Checkpoint**: US3 完成后，治理闭环与独立模式切换可独立验收。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 文档收口、指标验收与全量回归。

- [x] T037 [P] 更新特性文档中的日志策略与治理说明 in specs/020-framework-logger/spec.md
- [x] T038 [P] 更新 quickstart 执行与排障说明 in specs/020-framework-logger/quickstart.md
- [x] T039 [P] 对齐 OpenAPI 合同与实现字段命名 in specs/020-framework-logger/contracts/framework-logger.openapi.yaml
- [x] T040 执行并记录全量回归（US1+US2+US3）in tmp/020-framework-logger-regression.md
- [x] T041 执行 Go 回归测试并记录结果 in tmp/020-framework-logger-regression.md

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1): 无依赖，可立即开始。
- Foundational (Phase 2): 依赖 Phase 1，阻塞所有用户故事。
- User Stories (Phase 3-5): 全部依赖 Phase 2。
- Polish (Phase 6): 依赖 Phase 3-5。

### User Story Dependencies

- US1 (P1): 仅依赖 Foundational，可作为 MVP 优先交付。
- US2 (P2): 依赖 US1 的门面与策略基础能力。
- US3 (P3): 依赖 US1/US2，收口治理与 CI 规则。

### Within Each User Story

- 先完成测试任务，再完成实现任务。
- 先完成 framework 能力，再完成 skeleton 接线能力。
- 每个故事完成后必须可独立验收。

---

## Parallel Opportunities

- Phase 1: `T002`、`T003` 可并行。
- Phase 2: `T006`、`T007`、`T008`、`T010`、`T011` 可并行。
- US1: `T012`、`T013`、`T014` 可并行。
- US2: `T021`、`T022`、`T023` 可并行。
- US3: `T029`、`T030`、`T031` 可并行。
- Polish: `T037`、`T038`、`T039` 可并行。

---

## Parallel Example: User Story 2

```bash
Task T021: framework/backend/go/runtime/common/logging/router_test.go
Task T022: framework/backend/go/runtime/common/logging/retry_test.go
Task T023: skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_probe_handler_test.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1-2。  
2. 完成 Phase 3（US1）。  
3. 按 quickstart 验证宿主模式统一采集链路。  
4. 先行发布 framework 日志门面 alpha 能力。

### Incremental Delivery

1. US1：先交付宿主模式统一采集。  
2. US2：补齐多 sink 并行、重试与故障隔离。  
3. US3：完成遗留治理与 CI 阻断策略。  
4. Polish：收口合同、文档与回归证据。

### Parallel Team Strategy

1. 团队共同完成 Setup + Foundational。  
2. Foundational 完成后并行推进：
   - 开发 A：US1
   - 开发 B：US2
   - 开发 C：US3
3. 最后统一进行 Polish 与回归记录。

---

## Notes

- `[P]` 任务表示可并行（不同文件且无前置冲突）。
- `[USx]` 标签用于故事可追溯与独立验收。
- 每条任务均包含明确文件路径，可直接执行。
