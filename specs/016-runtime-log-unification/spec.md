# Feature Specification: Runtime 日志统一对齐

**Feature Branch**: `016-runtime-log-unification`  
**Created**: 2026-03-21  
**Status**: Draft  
**Input**: User description: "基于 `docs/plan/develop/log/` 的方案，生成并落地 runtime 日志统一对齐规范。"

## Clarifications

### Session 2026-03-22

- Q: `tenant_uuid -> tenant_key` 的迁移策略与兼容窗口如何定义？ → A: 保持 `tenant_uuid` 为主字段，同时输出由 `tenant_uuid` 派生的 `tenant_key` 镜像字段，提供迁移文档并设置 7 天回滚窗口。
- Q: `status` 字段采用哪套统一枚举？ → A: 采用标准集 `queued` / `processing` / `succeeded` / `failed` / `skipped`。
- Q: 当 `task_id/subscriber_id` 无法获取时如何记录？ → A: 写入 `unknown`，并记录 `status=skipped` 与 `reason=missing_context`。
- Q: 本次统一日志范围如何定义？ → A: 仅覆盖关键链路：Task enqueue/consume/ack/fail 与 WS publish/dispatch。
- Q: 如何定义日志字段完整率的验收样本规模？ → A: 关键链路全量验收（Task 4 类 + WS 2 类），每类至少 20 次事件。

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - 统一日志语义基线 (Priority: P1)

作为运行时治理负责人，我希望 framework 与 skeleton 产出的 runtime 日志遵循同一最小字段语义，这样跨链路排障时无需按模块切换不同字段口径。

**Why this priority**: 统一字段是后续观测、审计、告警和运维协同的前置条件，不统一会导致链路分析不可用。

**Independent Test**: 在任意一条 Task 或 WS 事件链路上检索日志，均可稳定看到统一的最小字段集合，并可据此串联上下游事件。

**Acceptance Scenarios**:

1. **Given** 系统存在跨模块 runtime 日志，**When** 运维按统一字段检索链路，**Then** 可在同一次检索中定位关键事件，不依赖模块私有字段。
2. **Given** 同一业务事件经过不同运行模式路径，**When** 比对日志字段，**Then** 最小字段语义一致且可比对。

---

### User Story 2 - 双模式日志一致排障 (Priority: P2)

作为插件开发者，我希望 Host 与 Standalone 两种运行模式在日志语义上保持一致，这样在模式切换或线上问题复盘时不需要重写排障流程。

**Why this priority**: 模式差异会显著增加排障成本，统一语义可降低误判与交付风险。

**Independent Test**: 分别在两种模式触发同类 runtime 事件，验证日志字段完整性与状态语义一致。

**Acceptance Scenarios**:

1. **Given** 同一类 runtime 事件在两种模式被触发，**When** 对比观测记录，**Then** 关键状态和最小字段语义一致。
2. **Given** 某模式出现发布或消费失败，**When** 查看失败日志，**Then** 可以通过标准状态枚举直接定位失败阶段和业务范围。

---

### User Story 3 - 文档与实现同口径 (Priority: P3)

作为文档维护者与 QA，我希望 async_runtime observability 文档与实际运行日志口径一致，避免“文档通过但实现不可验收”的偏差。

**Why this priority**: 文档口径是联调和验收入口，不一致会导致错误验收结论和重复沟通。

**Independent Test**: 按文档给出的最小字段清单进行日志检索，能够与实际运行产物一一对应。

**Acceptance Scenarios**:

1. **Given** QA 按 observability 文档执行联调检查，**When** 抽查 runtime 日志，**Then** 文档定义字段均可在实际日志中验证。
2. **Given** 文档新增或调整字段定义，**When** 执行验收清单，**Then** 能明确识别是“未实现”还是“字段退化”。

---

### Edge Cases

- 运行时上下文不完整（例如缺少租户标识或追踪标识）时，系统如何记录并标记日志状态？
- 同一事件在不同模块写出同义字段时，如何避免字段冲突导致检索歧义？
- 当事件发布成功但消费失败时，如何确保状态字段可反映分阶段结果而非单一结果？
- 历史日志仍按旧字段检索时，如何通过迁移文档在切换后快速完成规则改造？
- 当 `task_id` 或 `subscriber_id` 无法获取时，如何确保链路可观测且不造成静默缺失？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 定义并发布统一的 runtime 日志最小字段集合，包含 `trace_id`、`task_id`、`tenant_uuid`、`subscriber_id`、`topic`、`status`。
- **FR-001A**: 本次范围 MUST 限定为关键链路：Task enqueue/consume/ack/fail 与 WS publish/dispatch。
- **FR-002**: 系统 MUST 在 framework 与 skeleton 两类运行路径中使用同一日志能力入口，避免业务层直接依赖底层日志库类型。
- **FR-003**: 系统 MUST 在 Task 链路的关键阶段输出统一字段，并保证失败阶段同样可观测。
- **FR-004**: 系统 MUST 在 WebSocket 相关链路的关键阶段输出统一字段，并明确成功/失败状态。
- **FR-005**: 系统 MUST 保留插件侧扩展观测字段 `gateway_auth_scheme`、`outbound_token_source`、`plugin_id`、`component`，且不得破坏最小字段契约。
- **FR-006**: 系统 MUST 以 `tenant_uuid` 作为统一租户主字段，并输出 `tenant_key` 作为由 `tenant_uuid` 派生的镜像字段用于跨系统对齐。
- **FR-007**: 系统 MUST 更新并发布插件 async_runtime observability 文档，使其最小字段定义与 Core 规范一致。
- **FR-008**: 系统 MUST 提供明确的验收方式，使 QA 能通过日志检索验证字段完整性与状态语义。
- **FR-009**: 系统 MUST 在关键字段缺失时记录可识别的异常状态，以支持排障和质量追踪。
- **FR-010**: 系统 MUST 提供字段切换迁移文档，并在切换后保留 7 天回滚窗口。
- **FR-011**: 系统 MUST 将 `status` 统一限定为 `queued`、`processing`、`succeeded`、`failed`、`skipped`，不得由模块自定义新增状态值。
- **FR-012**: 系统 MUST 在 `task_id` 或 `subscriber_id` 无法获取时写入 `unknown`，并同时记录 `status=skipped` 与 `reason=missing_context`。

### Key Entities *(include if feature involves data)*

- **Runtime Log Record**: 一条 runtime 日志记录，包含最小字段集合、扩展字段与事件状态。
- **Field Contract**: 日志字段契约，定义字段名、语义、必填级别与兼容关系。
- **Migration Policy**: 字段切换策略，定义旧字段下线、迁移说明与回滚窗口。
- **Status Enum**: 运行态状态枚举，定义跨 Task/WS 链路统一可观测状态集合。
- **Verification Checklist**: 验收清单，用于校验文档定义与实际日志实现的一致性。
- **Metric Baseline**: 指标基线定义，说明首次通过率与排障返工次数的采样窗口与计算方式。

## Assumptions & Dependencies

- Core `async_runtime/log_trace` 将继续作为最小字段语义基线。
- 现有日志消费方会在切换窗口内按迁移文档完成检索规则改造。
- 运行模式切换不会改变业务事件语义，只允许改变底层执行路径。
- SC-003/SC-004 的度量窗口默认以发布后连续 14 天统计。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 在验收范围内的 Task 与 WS 关键链路日志中，最小字段完整率达到 100%。
- **SC-002**: 在 Host 与 Standalone 两种模式下，对同类事件抽样比对时，字段语义一致率达到 100%。
- **SC-001A**: 验收样本采用关键链路全量覆盖（Task 4 类 + WS 2 类），且每一类至少验证 20 次事件记录。
- **SC-003**: QA 按 observability 文档执行最小检查时，首次通过率达到 95% 及以上。
- **SC-004**: 由于字段口径不一致导致的 runtime 日志排障返工次数，相比改造前减少至少 50%。
- **SC-005**: SC-003 统计口径 MUST 基于发布后 14 天内的首次验收记录计算，并在交付文档中可追溯。
- **SC-006**: SC-004 统计口径 MUST 基于发布前 14 天与发布后 14 天的排障工单对比计算，并在交付文档中可追溯。
