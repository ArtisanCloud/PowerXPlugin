# Research — 008-framework-task-bus

## Decision 1: TaskBus SDK 形态与包路径

**Decision**：插件侧先落地稳定的 Emitter/Consumer 抽象与本地实现；TaskBus 实现通过“适配器接口”对接 framework，包路径以 framework 后续提供为准（在本仓库 `framework/` 目录中预留落点，但不把“缺失的 SDK”作为当前实施硬前置）。  

**Rationale**：

- 在当前仓库中未发现 `framework/backend/go/event`（仅在 `docs/plan/008-framework-task-bus.md` 中被引用），说明 TaskBus SDK 可能尚未落地或命名不同。
- 先固化插件侧抽象与事件契约，可降低后续接入成本，并支持本地/离线模式与灰度迁移。

**Alternatives considered**：

- 直接引入外部消息队列（Kafka/NATS/Redis Streams）：依赖重、运维复杂，与“最小依赖”原则冲突。
- 只做日志事件：无法实现订阅/重试/死信等治理能力。

## Decision 2: TaskBus 不可用时的策略

**Decision**：自动降级到本地实现，并记录告警/指标（主流程不 panic）。  

**Rationale**：保障可用性与灰度演进；同时用指标暴露异常并可人工切回/修复。

**Alternatives considered**：

- 失败闭合（阻断业务）：一致性更强但可用性风险高。
- 丢弃事件：可用性高但链路缺失风险高。

## Decision 3: 投递语义与幂等策略

**Decision**：采用 at-least-once 语义；consumer 必须幂等。默认幂等 key：`topic + tenant_uuid + trace_id`。  

**Rationale**：at-least-once 是最常见且可治理的语义；幂等是必要前提。默认 key 简化落地与跨 topic 统一规则。

**Alternatives considered**：

- at-most-once：丢事件风险高，不适合告警/KPI/任务中心等场景。
- exactly-once：成本与依赖过高，不适合作为默认要求。

## Decision 4: 契约变更治理

**Decision**：契约变更必须走 PR 评审 + CI 校验（至少校验 topic 唯一、必填 meta 字段齐全）。  

**Rationale**：避免字段漂移与版本不兼容；形成稳定治理流程。

**Alternatives considered**：

- 仅评审不校验：容易漏掉约束错误。
- 不治理：后期成本更高。

## Decision 5: 权限粒度（最小权限）

**Decision**：按 topic 前缀 + 版本号精确声明（尽量不用 `*`）。  

**Rationale**：符合零信任与最小权限原则，降低越权与误订阅风险。

