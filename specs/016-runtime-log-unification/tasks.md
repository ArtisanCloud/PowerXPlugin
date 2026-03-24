# Tasks: Runtime 日志统一对齐

**Input**: Design documents from `/specs/016-runtime-log-unification/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/runtime-log.schema.yaml, quickstart.md

**Tests**: 本特性要求关键链路字段完整率、语义一致性、扩展字段保留与统计口径验收，包含测试与验证任务。  
**Organization**: 任务按用户故事分组，确保每个故事可独立实现、独立验证。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件、无未完成前置依赖）
- **[Story]**: 所属用户故事（US1/US2/US3）
- 每个任务描述包含明确文件路径

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立日志统一改造所需的文档与契约基础

- [x] T001 对齐并固化任务范围说明到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/plan.md
- [x] T002 [P] 校对字段契约与状态枚举到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/contracts/runtime-log.schema.yaml
- [x] T003 [P] 补充快速验收命令与样本规则到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事共享且阻塞后续开发的基础能力

**⚠️ CRITICAL**: 本阶段完成前不得开始 US1/US2/US3

- [x] T004 定义统一日志字段常量与状态枚举到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/fields.go
- [x] T005 [P] 定义 runtime 统一日志接口（facade）到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/logger.go
- [x] T006 [P] 实现 slog 适配器到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/slog_adapter.go
- [x] T007 [P] 实现 logrus 适配器到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/logrus_adapter.go
- [x] T008 实现缺失上下文默认策略（`unknown` + `reason=missing_context`）到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/fallback.go
- [x] T009 [P] 新增统一日志接口单元测试到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/logger_test.go

**Checkpoint**: 基础日志能力已可复用，用户故事可开始并行推进

---

## Phase 3: User Story 1 - 统一日志语义基线 (Priority: P1) 🎯 MVP

**Goal**: framework 与 skeleton 在关键链路产出同一最小字段语义  
**Independent Test**: 在 Task/WS 任一关键链路中可检索到 `trace_id/task_id/tenant_uuid/tenant_key/subscriber_id/topic/status`

### Tests for User Story 1

- [x] T010 [P] [US1] 增加字段完整率测试（Task 关键链路，含 `tenant_uuid+tenant_key`）到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/taskbus/provider_log_fields_test.go
- [x] T011 [P] [US1] 增加字段完整率测试（WS 关键链路，含 `tenant_uuid+tenant_key`）到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/wsbus/adapter_log_fields_test.go

### Implementation for User Story 1

- [x] T012 [US1] 在 Task enqueue/consume/ack/fail 链路接入统一日志接口到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/taskbus/provider.go
- [x] T013 [US1] 在 WS publish/dispatch 链路接入统一日志接口到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/wsbus/adapter.go
- [x] T014 [P] [US1] 在 redis ws hub 转发链路补齐统一字段到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/wsbus/redis_hub.go
- [x] T015 [US1] skeleton RuntimeLogger 接入统一 facade 到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/shared/app/deps.go
- [x] T016 [P] [US1] 调整 runtime 字段注入函数到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/logger/runtime.go
- [x] T017 [US1] 补充 missing_context 路径日志行为到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/sessions_handler.go

**Checkpoint**: US1 完成后，关键链路字段基线可独立验收

---

## Phase 4: User Story 2 - 双模式日志一致排障 (Priority: P2)

**Goal**: Host / Standalone 两种模式日志语义一致，可直接比对排障  
**Independent Test**: 同类事件在两种模式下输出一致的字段语义与状态枚举

### Tests for User Story 2

- [x] T018 [P] [US2] 增加 status 枚举一致性测试到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/status_enum_test.go
- [x] T019 [P] [US2] 增加 `tenant_uuid` 主字段与 `tenant_key` 镜像字段测试到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/tenant_key_policy_test.go
- [x] T020 [P] [US2] 增加扩展字段保留测试（`gateway_auth_scheme/outbound_token_source/plugin_id/component`）到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/logger/runtime_extensions_test.go

### Implementation for User Story 2

- [x] T021 [US2] 实现 `tenant_uuid` 主字段与 `tenant_key` 镜像字段注入到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/fields.go
- [x] T022 [US2] 在 gateway 鉴权观测链路对齐统一字段并保留扩展字段到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_gateway_auth.go
- [x] T023 [P] [US2] 更新 framework 默认 logger 接入点以注入统一字段到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/bootstrap/app.go
- [x] T024 [US2] 补充 Host/Standalone 对比验证步骤到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md

**Checkpoint**: US2 完成后，双模式排障口径一致且可独立验证

---

## Phase 5: User Story 3 - 文档与实现同口径 (Priority: P3)

**Goal**: async_runtime observability 文档与实现字段完全对齐  
**Independent Test**: 按文档执行检索命令可验证最小字段、状态枚举、缺失上下文规则

### Tests for User Story 3

- [x] T025 [P] [US3] 新增文档-实现字段对齐检查清单到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/checklists/requirements.md

### Implementation for User Story 3

- [x] T026 [US3] 更新插件 observability 最小字段定义到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/docs/guides/async_runtime/observability/README.md
- [x] T027 [P] [US3] 更新插件 async_runtime 总览中的日志口径说明到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/docs/guides/async_runtime/README.md
- [x] T028 [US3] 更新字段矩阵与迁移策略到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/docs/plan/develop/log/runtime-log-field-matrix.md
- [x] T029 [US3] 更新日志统一改造计划的实施状态到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/docs/plan/develop/log/runtime-log-align-plan.md

**Checkpoint**: US3 完成后，文档与实现同口径可独立验收

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收尾、回归验证与发布前准备

- [x] T030 [P] 汇总 7 天回滚窗口执行说明到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/research.md
- [x] T031 执行 runtime 相关测试并记录结果到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md
- [x] T032 执行日志字段检索回归并记录样本统计到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md
- [x] T033 定义 SC-003（首次通过率）统计口径与样本台账到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md
- [x] T034 [P] 定义 SC-004（返工率下降）前后 14 天对比统计模板到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/research.md
- [x] T035 [P] 补充最终交付说明与实施顺序到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/plan.md
- [x] T036 执行 7 天回滚窗口演练并记录触发条件/结果到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md
- [x] T037 [P] 建立性能基线采集步骤（关键链路时延）到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/quickstart.md
- [x] T038 [P] 执行改造前后时延对比并记录 `<5%` 验收结论到 /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/016-runtime-log-unification/research.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖，可立即开始
- **Phase 2 (Foundational)**: 依赖 Phase 1，阻塞所有用户故事
- **Phase 3 (US1)**: 依赖 Phase 2
- **Phase 4 (US2)**: 依赖 Phase 2；建议在 US1 完成后做模式一致性联调
- **Phase 5 (US3)**: 依赖 US1/US2 的实现与验证结果
- **Phase 6 (Polish)**: 依赖目标用户故事全部完成

### User Story Dependencies

- **US1 (P1)**: 无用户故事前置依赖，是 MVP
- **US2 (P2)**: 可在基础能力完成后独立实现，但验收需引用 US1 统一字段输出
- **US3 (P3)**: 依赖 US1/US2 结果来完成文档与实现对齐

### Within Each User Story

- 先测试任务，再实现任务，再文档/验收任务
- 先统一接口与字段，再链路接入与行为校验

## Parallel Opportunities

- Phase 1：T002、T003 可并行
- Phase 2：T005、T006、T007、T009 可并行
- US1：T010 与 T011 可并行；T014 与 T016 可并行
- US2：T018、T019、T020 可并行；T023 可与 T022 并行
- US3：T026 与 T027 可并行
- Polish：T030 与 T034/T035 可并行；T037 与 T038 可并行

## Parallel Example: User Story 1

```bash
Task: "T010 [US1] taskbus log field tests in framework/backend/go/runtime/taskbus/provider_log_fields_test.go"
Task: "T011 [US1] wsbus log field tests in framework/backend/go/runtime/wsbus/adapter_log_fields_test.go"
Task: "T014 [US1] redis hub log field alignment in framework/backend/go/runtime/wsbus/redis_hub.go"
Task: "T016 [US1] runtime field injection alignment in skeleton/backend/go-gin/internal/logger/runtime.go"
```

## Parallel Example: User Story 2

```bash
Task: "T018 [US2] status enum consistency tests in framework/backend/go/runtime/common/logging/status_enum_test.go"
Task: "T019 [US2] tenant_uuid primary + tenant_key mirror tests in framework/backend/go/runtime/common/logging/tenant_key_policy_test.go"
Task: "T020 [US2] runtime extension fields tests in skeleton/backend/go-gin/internal/logger/runtime_extensions_test.go"
Task: "T023 [US2] bootstrap logger field injection in framework/backend/go/bootstrap/app.go"
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1 + Phase 2
2. 完成 US1（T010-T017）
3. 执行 US1 独立验证并确认关键链路字段完整率

### Incremental Delivery

1. 交付 US1：统一字段基线
2. 交付 US2：双模式一致性
3. 交付 US3：文档同口径
4. 最后执行 Polish：回归、回滚演练与性能验收

### Parallel Team Strategy

1. 基础能力组：T004-T009
2. runtime 链路组：T012-T014、T021-T023
3. skeleton 接入组：T015-T017
4. 文档与验收组：T024-T038

## Notes

- 所有任务均遵循严格 checklist 格式
- `[P]` 任务需避免修改相同文件以降低冲突
- 每个用户故事在对应 checkpoint 完成后应先独立验收再推进下一阶段
