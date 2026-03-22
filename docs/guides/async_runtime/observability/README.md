# Observability（插件侧）

## 1. 范围

1. 关键链路日志：Task `enqueue/consume/ack/fail`、WS `publish/dispatch`
2. Event/Task 指标
3. proxy 出站鉴权观测字段

## 2. 统一最小字段（必须可检索）

1. `trace_id`
2. `task_id`
3. `tenant_uuid`
4. `tenant_key`（由 `tenant_uuid` 派生的镜像字段）
5. `subscriber_id`
6. `topic`
7. `status`

## 3. 状态与缺失上下文规则

状态枚举固定为：

1. `queued`
2. `processing`
3. `succeeded`
4. `failed`
5. `skipped`

缺失上下文默认策略：

1. 当 `task_id` 或 `subscriber_id` 无法获取时，写入 `unknown`
2. 同时写入 `status=skipped`
3. 同时写入 `reason=missing_context`

## 4. 插件扩展字段（保留）

1. `gateway_auth_scheme`
2. `outbound_token_source`
3. `plugin_id`
4. `component`

## 5. 最小指标

1. `plugin_event_bridge_emit_total`
2. `plugin_event_bridge_drop_total`
3. `plugin_event_bridge_latency_ms`

## 6. 最小验证命令

字段契约检查：

```bash
rg 'trace_id|task_id|tenant_uuid|tenant_key|subscriber_id|topic|status' <runtime-log-file>
```

缺失上下文规则检查：

```bash
rg 'task_id=unknown|subscriber_id=unknown|status=skipped|reason=missing_context' <runtime-log-file>
```

扩展字段检查：

```bash
rg 'gateway_auth_scheme|outbound_token_source|plugin_id|component' <runtime-log-file>
```

指标检查：

```bash
curl -sS http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_'
```

## 7. proxy 场景关键判定

日志需看到：

1. `gateway_auth_scheme=apikey`
2. `outbound_token_source=PX_GATEWAY_API_KEY`
