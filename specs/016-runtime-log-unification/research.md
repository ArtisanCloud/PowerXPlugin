# Research: Runtime 日志统一对齐

## Decision 1: 统一字段契约基线

- **Decision**: 最小字段固定为 `trace_id`、`task_id`、`tenant_uuid`、`subscriber_id`、`topic`、`status`，并同步输出 `tenant_key` 镜像字段。
- **Rationale**: 同时满足插件宪章（租户主语义为 `tenant_uuid`）与跨系统字段对齐需求（`tenant_key`）。
- **Alternatives considered**:
  - 继续沿用插件自定义最小集：会持续与 Core 口径分叉。
  - 只保留 `trace_id/topic/status`：不足以支撑任务链路追踪。

## Decision 2: 租户字段切换策略

- **Decision**: 保持 `tenant_uuid` 为主字段；新增日志必须输出由 `tenant_uuid` 派生的 `tenant_key` 镜像字段；提供迁移文档并保留 7 天回滚窗口。
- **Rationale**: 避免与宪章冲突，同时保留跨系统检索兼容能力。
- **Alternatives considered**:
  - 以 `tenant_key` 作为主字段：与宪章冲突。
  - 永久无镜像字段：会降低与 Core 口径协同效率。

## Decision 3: status 统一枚举

- **Decision**: 统一为 `queued`、`processing`、`succeeded`、`failed`、`skipped`。
- **Rationale**: 能覆盖 Task/WS 关键阶段并保持足够简洁，便于告警和统计聚合。
- **Alternatives considered**:
  - 二值状态（success/failed）：无法表达中间过程。
  - 过度细粒度枚举：提升接入和维护复杂度。

## Decision 4: 关键字段缺失处理

- **Decision**: 当 `task_id` 或 `subscriber_id` 无法获取时，写入 `unknown`，并记录 `status=skipped` 与 `reason=missing_context`。
- **Rationale**: 保持字段契约稳定，避免静默缺失导致的排障盲区。
- **Alternatives considered**:
  - 省略字段：会破坏完整率统计和查询模板。
  - 直接丢弃日志：会丢失失败诊断信息。

## Decision 5: 实施范围

- **Decision**: 本次仅覆盖关键链路：Task `enqueue/consume/ack/fail` 与 WS `publish/dispatch`。
- **Rationale**: 在控制改造风险的前提下覆盖最高价值路径，满足当前验收目标。
- **Alternatives considered**:
  - 一次性全量改造所有 runtime 日志：周期长、风险高。
  - 仅覆盖失败日志：无法验证完整生命周期语义。

## Decision 6: 验收样本定义

- **Decision**: 关键链路全量覆盖（Task 4 类 + WS 2 类），每一类至少验证 20 次事件记录。
- **Rationale**: 保证可重复、可比较的验收基线，避免“抽样口径随执行人变化”。
- **Alternatives considered**:
  - 自由抽样：不可审计、不可复现。
  - 小样本抽检：无法证明 100% 字段完整率。

## Decision 7: 指标口径与统计窗口

- **Decision**: SC-003 与 SC-004 统一采用发布前后各 14 天统计窗口，并在交付文档记录计算口径、样本来源与结果。
- **Rationale**: 保证“首次通过率/返工下降”可审计、可复现，避免主观判断。
- **Alternatives considered**:
  - 不固定统计窗口：结果不可比。
  - 仅给目标值不定义采样：无法验证达标真实性。

## Rollback Window（T030）

### 7 天回滚窗口执行说明

1. 窗口定义：发布后第 1 天至第 7 天（含）为兼容回滚窗口。
2. 回滚触发条件（任一满足即触发）：
   - 关键链路字段完整率低于 100%
   - 出现非标准 `status` 值（不在 `queued/processing/succeeded/failed/skipped`）
   - `task_id/subscriber_id` 缺失且未按 `unknown + missing_context` 记录
3. 回滚动作：
   - 回退到上一个稳定版本（不变更业务协议）
   - 保留 `tenant_uuid` 主字段与 `tenant_key` 镜像字段常量定义
   - 恢复旧检索规则并保留 7 天双字段兼容
4. 回滚后验证：
   - 重跑 runtime 回归测试
   - 重跑关键字段检索与状态枚举检索
   - 记录回滚工单与触发证据

## SC-004 对比统计模板（T034）

统计窗口：

1. 改造前：发布日前 14 天
2. 改造后：发布日后 14 天

台账字段模板：

| 维度 | 说明 |
|---|---|
| ticket_id | 排障工单编号 |
| created_at | 工单创建时间 |
| runtime_area | taskbus/wsbus/gateway_auth |
| root_cause | 是否字段口径不一致 |
| rework_count | 返工次数 |
| first_pass | 是否首次通过 |
| mode | host/standalone |

计算公式：

`返工率下降 = (改造前返工次数 - 改造后返工次数) / 改造前返工次数`

## 性能对比结论（T038）

对比对象：`go test ./runtime/taskbus ./runtime/wsbus -count=1`

1. 改造前（US1 基线，2026-03-22 执行记录）：
   - taskbus: `0.719s`
   - wsbus: `1.013s`
2. 改造后（Phase 6，见 `tmp/phase6-performance-after.log`）：
   - taskbus: `0.520s`
   - wsbus: `0.863s`
3. 对比结果：
   - taskbus: `-27.68%`
   - wsbus: `-14.81%`
4. 验收结论：
   - 未出现正向时延退化，满足“关键链路时延增量 < 5%”约束（PASS）
