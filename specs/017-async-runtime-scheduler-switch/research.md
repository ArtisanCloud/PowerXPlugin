# Research: Async Runtime Scheduler 模式切换

## Decision 1: 模式冲突采用严格失败（Fail-Fast）

- **Decision**: 当 `POWERX_PROXY` 与 `taskbus_provider` 不匹配时，启动即失败并返回明确错误，禁止静默继续执行。
- **Rationale**: 能最早暴露配置错误并防止任务误投递，符合“高风险路径先阻断”的运维策略。
- **Alternatives considered**:
  - 自动覆盖配置继续启动：会掩盖错误，增加隐性风险。
  - 启动成功但禁用部分功能：故障表现不直观，排障成本更高。

## Decision 2: 调度触发与手动触发统一入口

- **Decision**: 两种触发方式统一进入事件主链路（同一语义入口），不允许形成双轨执行。
- **Rationale**: 可保证验收与排障流程一致，避免“调度成功但手动失败”这类语义分叉。
- **Alternatives considered**:
  - 调度走独立执行通道：短期快，但长期维护和测试成本高。
  - 仅统一结果不统一过程：会导致中间状态不可比。

## Decision 3: Proxy 权限失败采用“有限重试 + 人工闭环”

- **Decision**: `delegated proxy` 权限失败时，任务进入有上限重试；超限后创建工单并暂停任务。
- **Rationale**: 兼顾瞬时故障恢复能力与系统稳定性，避免无限重试造成雪崩。
- **Alternatives considered**:
  - 直接丢弃：业务可见性差，损失任务。
  - 无限重试：可能放大故障并占满资源。

## Decision 4: 恢复权限默认仅运维/管理员

- **Decision**: 超限暂停后的恢复操作仅限平台运维/管理员执行，并保留审计记录。
- **Rationale**: 恢复动作具有生产影响面，需要高权限和可追责性。
- **Alternatives considered**:
  - 开发者直接恢复：权限边界过宽。
  - 自动定时恢复：可能在根因未解决时反复失败。

## Decision 5: 结果语义一致性的验收口径

- **Decision**: 一致性定义为“状态流转 + 业务结果一致”，允许执行耗时有差异。
- **Rationale**: 既保证语义一致，又避免把非关键性能波动误判为功能不一致。
- **Alternatives considered**:
  - 要求完全一致（含耗时与顺序）：过度严格，不利于真实环境验收。
  - 只看最终成功/失败：无法覆盖过程性错误。

## Decision 6: 观测与统计窗口

- **Decision**: 成功率、首次通过率、故障定位时长采用发布后 14 天窗口统计，并要求可追溯。
- **Rationale**: 形成稳定统计基线，支持回归比较与发布决策。
- **Alternatives considered**:
  - 无固定窗口：结果不可比。
  - 仅单次验收结论：不能反映稳定性趋势。
