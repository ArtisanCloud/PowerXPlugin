# Runtime 日志字段对齐矩阵（Core vs Plugin）

## 1. 当前统一契约（已实施）

| 字段 | Core `log_trace` 基线 | Plugin 文档口径 | Framework/Skeleton 实现 | 状态 |
|---|---|---|---|---|
| `trace_id` | 必需 | 必需 | 已接入关键链路 | 对齐 |
| `task_id` | 必需 | 必需 | 缺失时回填 `unknown` | 对齐 |
| `tenant_uuid` | 插件主字段 | 必需 | 主字段 | 对齐 |
| `tenant_key` | 兼容对齐字段 | 必需（镜像） | 由 `tenant_uuid` 派生 | 对齐 |
| `subscriber_id` | 必需 | 必需 | 缺失时回填 `unknown` | 对齐 |
| `topic` | 必需 | 必需 | 已接入关键链路 | 对齐 |
| `status` | 必需 | 必需 | 统一枚举约束 | 对齐 |
| `reason` | 缺失上下文必带 | 条件必需 | `missing_context` | 对齐 |

## 2. 状态枚举与缺失上下文规则

状态枚举固定为：

1. `queued`
2. `processing`
3. `succeeded`
4. `failed`
5. `skipped`

缺失上下文默认策略：

1. 当 `task_id` 或 `subscriber_id` 缺失时，字段写入 `unknown`
2. 同时写入 `status=skipped`
3. 同时写入 `reason=missing_context`

## 3. 插件扩展字段（保留，不破坏最小契约）

1. `gateway_auth_scheme`
2. `outbound_token_source`
3. `plugin_id`
4. `component`

## 4. 迁移与兼容策略

1. `tenant_uuid` 作为统一主字段，不变更语义
2. `tenant_key` 作为镜像字段，始终由 `tenant_uuid` 派生
3. 检索/告警规则兼容窗口：7 天同时支持 `tenant_uuid` 与 `tenant_key`
4. 兼容窗口后，按消费方改造进度评估是否收敛为单主检索字段（默认仍推荐 `tenant_uuid`）

## 5. 验收样本与范围

关键链路范围：

1. `task.enqueue`
2. `task.consume`
3. `task.ack`
4. `task.fail`
5. `ws.publish`
6. `ws.dispatch`

样本规则：

1. 每类链路至少 20 条事件
2. 全量关键链路验收，不降采样
3. 每类需覆盖成功与失败路径
