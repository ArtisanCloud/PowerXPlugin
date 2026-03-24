# Runtime 日志统一改造计划（Framework + Skeleton）

## 1. 目标

对齐 Core `async_runtime/log_trace` 与插件 runtime 日志口径，确保 framework 与 skeleton 在关键链路输出统一字段语义，支持 Host / Standalone 双模式一致排障。

## 2. 统一契约（最终口径）

最小字段（必须）：

1. `trace_id`
2. `task_id`
3. `tenant_uuid`（主字段）
4. `tenant_key`（由 `tenant_uuid` 派生镜像）
5. `subscriber_id`
6. `topic`
7. `status`

扩展字段（保留）：

1. `gateway_auth_scheme`
2. `outbound_token_source`
3. `plugin_id`
4. `component`

状态枚举：

1. `queued`
2. `processing`
3. `succeeded`
4. `failed`
5. `skipped`

缺失上下文规则：

1. 当 `task_id` 或 `subscriber_id` 缺失时写入 `unknown`
2. 同时写入 `status=skipped`
3. 同时写入 `reason=missing_context`

## 3. 范围与非范围

范围：

1. Task 关键链路：`enqueue/consume/ack/fail`
2. WS 关键链路：`publish/dispatch`
3. gateway 鉴权观测链路统一字段对齐
4. async_runtime 文档口径对齐与验收步骤补齐

非范围：

1. 不调整对外 HTTP/WS 协议
2. 不引入新的日志存储基础设施
3. 不在本期做 skeleton 全量 `slog` 迁移

## 4. 迁移与回滚策略

1. `tenant_uuid` 持续作为主字段
2. `tenant_key` 作为镜像字段同步输出
3. 检索/告警规则提供 7 天兼容窗口（`tenant_uuid` + `tenant_key`）
4. 如出现线上风险，优先回退到上一个稳定版本并保留统一字段常量，不改业务协议

## 5. 实施状态（截至 2026-03-22）

1. Phase 1（Setup）：已完成
2. Phase 2（Foundational）：已完成
3. Phase 3（US1）：已完成
4. Phase 4（US2）：已完成
5. Phase 5（US3）：已完成（文档与实现同口径）

当前未完成：

1. Phase 6（Polish & Cross-Cutting）：待执行（回滚演练、统计台账、性能对比）

## 6. 验收要点

1. 最小字段完整率：关键链路 100%
2. 双模式一致性：Host / Standalone 同类事件字段语义一致
3. 样本规模：6 类关键链路每类 >= 20 条
4. 文档一致性：`docs/guides/async_runtime/*` 与 `specs/016-runtime-log-unification/*` 口径一致
