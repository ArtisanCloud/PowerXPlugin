# Feature Specification: Framework TaskBus Event Bridge

**Feature Branch**: `008-framework-task-bus`  
**Created**: 2025-12-30  
**Status**: Draft  
**Input**: User description: "根据 docs/plan/008-framework-task-bus.md，为 PowerXPlugin Framework TaskBus 集成生成 spec 文档（为插件提供公用 event 机制，并支持与宿主共享事件流/指标/后台任务调度）。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 业务侧统一发事件 (Priority: P1)

作为插件业务模块开发者（以 Channel 主数据/商品渠道任务为代表），我希望在不改变业务接口的前提下，把“直接写日志/直接触发 job/直接写 DB”的副作用收敛到一个统一的事件出口，业务层只需要“发事件”即可。

**Why this priority**：这是迁移的最小可用切片，先把事件出口统一，才能逐步替换后续的 job/观测实现，同时不破坏现有功能。

**Independent Test**：在本地启用“本地实现（非 TaskBus）”，触发一次渠道凭证巡检/发布任务，能看到事件被发出（记录到日志或内存队列）且业务行为不回归。

**Acceptance Scenarios**：

1. **Given** Channel 业务服务执行完成一次“凭证巡检”，**When** 业务层调用统一 Emitter 发出事件，**Then** 事件包含正确的 `topic/name`、`tenant_uuid`、`occurred_at`、`payload_version` 与 payload 字段。
2. **Given** Emitter 不可用或返回错误，**When** 业务层发事件，**Then** 业务主流程不发生 panic，错误被记录并可被监控发现。

---

### User Story 2 - TaskBus 接入与双实现切换 (Priority: P2)

作为插件运行时/平台工程师，我希望插件能够通过配置开关决定走“本地实现”还是“Framework TaskBus 实现”，从而在不同环境（本地、staging、生产、宿主代理）进行灰度与回滚。

**Why this priority**：没有切换能力就无法灰度验证与快速回滚；这也是把事件机制从“本地可用”升级为“宿主可治理”的关键。

**Independent Test**：在 staging 开启 `event_bridge.enabled=true`，发布/订阅声明生效，能将一个事件发布到 TaskBus 并被 consumer 处理；关闭开关后回退本地实现仍可用。

**Acceptance Scenarios**：

1. **Given** `event_bridge.enabled=false`，**When** 发出 Channel 事件，**Then** 使用本地实现路径处理（不依赖 TaskBus）。
2. **Given** `event_bridge.enabled=true` 且 TaskBus 可用，**When** 发出 Channel 事件，**Then** 事件被发布到 TaskBus 并按订阅关系触发 consumer。
3. **Given** `event_bridge.enabled=true` 但 TaskBus 不可用，**When** 发出事件，**Then** 系统自动降级到本地实现，并记录告警/指标（主流程不 panic）；同时允许通过配置回滚到本地实现作为长期兜底。

---

### User Story 3 - 事件契约与任务/观测迁移 (Priority: P3)

作为跨团队协作的集成方（插件/宿主/运维/QA），我希望有明确的事件契约（Topic 命名、通用 meta 字段、payload schema 版本化）与迁移步骤（双写/双读、至少 1 周验证、可回滚），以便治理告警、KPI 刷新、任务中心反馈等链路。

**Why this priority**：没有契约与迁移策略会造成事件不可控（字段漂移、版本不兼容、审计缺失），无法推进到 TaskBus-only。

**Independent Test**：对一个选定 Topic（例如 `powerx.channel.master.credential_inspection.v1`）完成双写验证：旧日志/DB 与新事件消费结果一致，且可观测指标达标。

**Acceptance Scenarios**：

1. **Given** 仓库存在事件契约文档，**When** 新增/修改事件字段，**Then** 需要更新契约并通过评审流程（或 CI 校验）后才能合并。
2. **Given** 开启双写策略，**When** 同一业务动作触发旧链路与新事件链路，**Then** 两边结果在可接受的时间窗口内一致。

---

### Edge Cases

- TaskBus 侧出现重试/死信：consumer 连续失败时如何告警？死信如何人工回放？
- 同一事件被重复投递（at-least-once）：consumer 需要具备幂等性（按 `trace_id` + 业务 key 去重）。
- 同一业务在不同租户下并发：是否存在 tenant 泄露或 topic 权限配置错误导致越权？

## Clarifications

### Session 2025-12-30

- Q: TaskBus 不可用时的策略？ → A: 自动降级到本地实现，并记录告警/指标（主流程不 panic）。
- Q: 事件投递语义与 Consumer 幂等要求？ → A: 至少一次投递（at-least-once），Consumer 必须幂等（基于 `trace_id` + 业务 key 去重）。
- Q: 幂等去重的业务 key 选取？ → A: 使用 `topic + tenant_uuid + trace_id` 作为去重 key（trace_id 缺失时不保证强幂等）。
- Q: 事件契约变更的治理方式？ → A: 契约变更必须走 PR 评审 + CI 校验（至少检查 topic 唯一与必填 meta 字段齐全）。
- Q: 事件权限粒度如何定义？ → A: 最小权限，按 topic 前缀 + 版本号精确声明（尽量不用 `*`）。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**：系统 MUST 提供统一的事件抽象接口（Emitter/Consumer），业务层通过接口发事件与处理事件。
- **FR-002**：系统 MUST 支持至少两套实现并存：本地实现与 Framework TaskBus 实现，并可通过配置开关切换。
- **FR-003**：系统 MUST 定义并遵循 Topic 命名规范 `powerx.<domain>.<subdomain>.<action>.v<version>`。
- **FR-004**：系统 MUST 为每个事件携带通用 meta 字段：`tenant_uuid`、`request_id`、`source_plugin`、`trace_id`、`occurred_at`、`payload_version`。
- **FR-005**：系统 MUST 支持在插件清单（manifest/plugin.yaml；开发态路径为 `skeleton/plugin.yaml`）中声明事件发布/订阅权限，并在运行时遵守该权限边界。
- **FR-006**：权限声明 MUST 采用最小权限原则：按 topic 前缀 + 版本号精确声明，避免使用全局通配符（如 `powerx.*`），除非经安全评审。
- **FR-007**：系统 MUST 支持迁移策略：双写/双读、可回滚（通过开关退回本地实现）。
- **FR-008**：系统 MUST 提供可观测性：至少能统计事件发布/消费的成功率、失败率与处理延迟。
- **FR-009**：系统 MUST 遵循租户隔离原则：事件与消费必须在正确的 `tenant_uuid` 上下文中执行。
- **FR-010**：系统 MUST 避免在事件 payload 中包含明文敏感信息（如凭证明文）；应采用引用或脱敏字段。
- **FR-011**：系统 MUST 提供事件契约文档（Topic + payload schema）并纳入仓库管理。
- **FR-012**：系统 MUST 对事件契约变更实施治理：通过 PR 评审，并在 CI 中至少校验 topic 唯一性与必填 meta 字段齐全。
- **FR-013**：系统 SHOULD 提供最小可用的本地 TaskBus/内存 bus 用于集成测试（或替代方案）。
- **FR-014**：系统 MUST 明确错误语义：当 TaskBus 不可用时，发布/消费失败如何处理（失败闭合 vs 降级）并可被监控发现。  
  当 TaskBus 不可用时，系统 MUST 自动降级到本地实现，并记录告警/指标（主流程不 panic）。
- **FR-015**：系统 MUST 采用至少一次投递（at-least-once）语义；Consumer MUST 具备幂等性，并能基于 `trace_id` + 业务 key 去重重复事件。
- **FR-016**：系统 MUST 将 `topic + tenant_uuid + trace_id` 作为默认幂等去重 key；当 `trace_id` 缺失或为空时，允许退化为“尽力而为”的幂等（记录告警/指标以便发现链路缺失）。

### Key Entities *(include if feature involves data)*

- **Event**：一次可被发布与消费的消息，包含 `name/topic`、`payload`、`meta`。
- **Topic**：事件分类与路由键（带版本号）。
- **Subscription**：consumer 对某些 topic 的订阅关系（含并发度/重试策略）。
- **EventBridgeConfig**：决定本地实现/TaskBus 实现、双写策略与降级策略的配置集合。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**：在不修改业务对外 API 的前提下，至少一个 Channel 业务动作可以通过统一 Emitter 发出事件，并能在本地实现模式下验证成功。
- **SC-002**：在 staging 环境中，开启 TaskBus 模式后，至少一个 Topic 能完成“发布→消费→落地结果”的端到端链路，且成功率 ≥ 99%（以 24h 统计）。
- **SC-003**：双写验证期（≥ 7 天）内，新事件链路与旧链路的结果一致性达到预期（例如 KPI 刷新结果差异为 0，或告警数量差异在可接受范围内）。
- **SC-004**：当 TaskBus 出现不可用/高失败率时，系统能在 10 分钟内通过监控定位问题并通过配置回滚到本地实现，恢复核心业务可用性。
