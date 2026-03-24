# Feature Specification: Async Runtime Scheduler 模式切换

**Feature Branch**: `017-async-runtime-scheduler-switch`  
**Created**: 2026-03-23  
**Status**: Draft  
**Input**: User description: "请根据 docs/plan/develop/async_runtime/schedule 生成对应的 spec 文档。"

## Clarifications

### Session 2026-03-23

- Q: 当 `POWERX_PROXY` 与 `taskbus_provider` 配置冲突时，系统应采用哪种策略？ → A: 采用严格失败策略：启动即失败并返回明确错误，禁止静默继续执行。
- Q: `delegated proxy` 模式下权限失败时，调度任务默认处理策略是什么？ → A: 标记失败并进入有上限的重试队列，超限后转人工处理。
- Q: 调度任务重试上限达到后，人工处理的标准动作是什么？ → A: 自动创建待处理工单，并暂停该任务后续触发。
- Q: 调度触发与手动触发“结果语义一致”应按什么口径验收？ → A: 状态流转与业务结果一致，允许执行耗时存在差异。
- Q: 重试超限暂停任务后，恢复触发权限默认给谁？ → A: 仅平台运维/管理员可恢复。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 自动识别运行模式并切换调度执行路径 (Priority: P1)

作为插件后端开发者，我希望系统在启动时自动识别当前运行模式并选择正确的调度执行路径，这样业务任务无需维护两套实现。

**Why this priority**: 如果模式识别和切换不稳定，调度任务会出现误投递、权限失败或运行路径漂移，直接影响主链路可用性。

**Independent Test**: 分别在 standalone local 与 delegated proxy 两种模式启动系统，触发同一调度任务，验证任务进入正确执行路径且结果可观测。

**Acceptance Scenarios**:

1. **Given** 系统以 standalone local 模式启动，**When** 调度任务触发标准 topic（如 `powerx.runtime.scheduler.triggered.v1`），**Then** 任务通过本地执行路径进入统一事件链路并完成投递。
2. **Given** 系统以 delegated proxy 模式启动，**When** 调度任务触发标准 topic（如 `powerx.runtime.scheduler.triggered.v1`），**Then** 任务通过代理执行路径进入统一事件链路并完成投递。
3. **Given** 模式信号与执行配置不一致，**When** 系统启动并触发调度任务，**Then** 系统应输出可定位的错误并阻止静默错误执行。

---

### User Story 2 - 调度触发与手动触发保持同链路语义 (Priority: P2)

作为运行时维护者，我希望调度触发与手动触发复用同一事件入口与语义，这样验收和排障流程可以统一。

**Why this priority**: 双轨触发会导致行为不一致，增加回归成本与线上诊断复杂度。

**Independent Test**: 对同一业务主题分别执行手动触发和调度触发，验证事件语义、结果状态与观测口径一致。

**Acceptance Scenarios**:

1. **Given** 同一标准业务 topic 已声明可发布，**When** 分别执行手动触发与调度触发，**Then** 两种触发方式产生一致的业务结果语义。
2. **Given** 调度任务触发失败，**When** 运维查看事件与日志，**Then** 可使用与手动触发相同的排障路径定位失败原因。

---

### User Story 3 - 双模式下可观测与联调流程一致 (Priority: P3)

作为 QA 与联调人员，我希望在两种运行模式下都能用同一套检查步骤完成验收，这样交付门槛和回归效率可控。

**Why this priority**: 如果联调流程不统一，跨环境交付会出现重复脚本、误判和沟通成本上升。

**Independent Test**: 使用统一验收脚本在两种模式执行，验证均能得到可追踪事件、状态和成功判定。

**Acceptance Scenarios**:

1. **Given** 系统运行在任一模式，**When** 执行标准联调脚本，**Then** 均可观察到完整的事件触发与结果回传。
2. **Given** 任一模式出现鉴权或权限错误，**When** 按统一排障顺序检查，**Then** 可在有限步骤内确认根因类别。

---

### Edge Cases

- 当运行模式信号缺失或非法时，系统如何避免默认落到错误执行路径？
- 当模式信号与执行配置冲突时，系统如何防止任务被错误执行或重复执行？
- 当调度任务触发成功但下游消费失败时，如何保证结果可观测且可重试？
- 当 delegated proxy 模式下权限快照未更新时，如何在首次失败中给出可执行修复指引？
- 当同一主题在不同触发方式下出现结果不一致时，如何快速判定是模式切换问题还是业务逻辑问题？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 在启动阶段基于运行模式信号识别当前模式，并在任务触发前确定唯一执行路径。
- **FR-002**: 系统 MUST 支持至少两种运行模式：`standalone local` 与 `delegated proxy`。
- **FR-003**: 系统 MUST 在模式识别后，将调度触发统一路由到同一事件执行抽象，避免业务侧维护模式分支逻辑。
- **FR-004**: 系统 MUST 要求调度触发与手动触发复用同一事件入口语义，并在验收中可被验证。
- **FR-004A**: 系统 MUST 将“结果语义一致”定义为状态流转与业务结果一致；执行耗时差异不作为不一致判定条件。
- **FR-005**: 系统 MUST 在模式信号与执行配置冲突时采用严格失败策略：启动即失败并返回明确错误，且不得静默继续执行。
- **FR-006**: 系统 MUST 为两种模式提供一致的最小联调流程，包括触发、观察、判定与排障顺序。
- **FR-007**: 系统 MUST 在 delegated proxy 模式下执行权限校验，并在权限不满足时返回可诊断的失败结果。
- **FR-008**: 系统 MUST 在 delegated proxy 模式下发生权限失败时，将任务标记为失败并进入有限重试流程，且全程可观测。
- **FR-008A**: 系统 MUST 为重试流程定义明确默认上限与可配置范围（默认 3 次，可配置范围 1-10 次），并在重试超限后自动创建待处理工单且暂停对应任务的后续定时触发，直至人工恢复。
- **FR-008B**: 系统 MUST 将重试超限后任务恢复权限限定为平台运维/管理员角色，并保留恢复操作审计记录。
- **FR-009**: 系统 MUST 在手动触发与调度触发链路中统一透传可追踪运行标识（至少包含 `trace_id`），支持从触发到分发的链路关联。
- **FR-010**: 系统 MUST 提供明确的开发完成标准（Done Definition），用于确认调度接线、模式切换和验收覆盖均达标。

### Key Entities *(include if feature involves data)*

- **Runtime Mode Signal**: 运行模式判定输入，表示系统当前应采用的执行策略。
- **Scheduler Trigger**: 调度触发请求，表示“何时执行”的意图与上下文。
- **Event Dispatch Request**: 统一事件触发对象，承载主题、载荷与链路追踪信息。
- **Execution Path Decision**: 启动期形成的单一路径决策结果，用于约束任务投递方向。
- **Validation Evidence**: 验收证据集合，包含联调输出、观测结果和失败定位记录。

## Assumptions & Dependencies

- 调度能力以统一事件链路为中心，不在本特性中新增独立业务协议。
- 两种模式均已具备基础鉴权能力，且可提供有效凭证进行联调。
- 业务主题声明与权限配置由既有流程维护，本特性只定义调度切换与执行一致性。
- 成功指标统计窗口默认采用发布后连续 14 天。
- 事件命名遵循宪章：`powerx.<domain>.<subdomain>.<action>.v<version>`。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 在发布后连续 14 天窗口内，按 `standalone local` 与 `delegated proxy` 分组统计时，调度触发成功进入统一事件链路的通过率均达到 100%。
- **SC-002**: 对同一业务主题进行手动触发与调度触发比对时，结果语义一致率达到 100%。
- **SC-003**: 标准联调流程在两种模式下首次执行通过率达到 95% 及以上。
- **SC-004**: 与模式切换相关的调度故障平均定位时间在发布后 14 天内较基线下降至少 50%。
- **SC-005**: 由模式配置冲突导致的静默错误执行事件数为 0。
- **SC-006**: SC-003 的统计必须基于发布后 14 天首轮验收台账计算，并保留可追溯证据。
- **SC-007**: SC-004 的统计必须基于发布前后各 14 天工单对比台账计算，并保留可追溯证据。
