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

先明确日志文件路径（推荐统一导出）：

```bash
export RUNTIME_LOG_FILE=./tmp/runtime-backend.log
```

本地启动时建议显式落盘（stdout + 文件）：

```bash
# standalone
POWERX_PROXY=0 ./backend/cmd/plugin/plugin 2>&1 | tee "$RUNTIME_LOG_FILE"

# delegated/proxy
POWERX_PROXY=1 PX_GATEWAY_BASE_URL=http://127.0.0.1:8077 ./backend/cmd/plugin/plugin 2>&1 | tee "$RUNTIME_LOG_FILE"
```

如果是容器化部署，不落本地文件时用容器日志替代：

```bash
docker logs -f <container_name> | tee "$RUNTIME_LOG_FILE"
```

字段契约检查：

```bash
rg 'trace_id|task_id|tenant_uuid|tenant_key|subscriber_id|topic|status' "$RUNTIME_LOG_FILE"
```

缺失上下文规则检查：

```bash
rg 'task_id=unknown|subscriber_id=unknown|status=skipped|reason=missing_context' "$RUNTIME_LOG_FILE"
```

扩展字段检查：

```bash
rg 'gateway_auth_scheme|outbound_token_source|plugin_id|component' "$RUNTIME_LOG_FILE"
```

指标检查：

```bash
curl -sS http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_'
```

## 7. proxy 场景关键判定

日志需看到：

1. `gateway_auth_scheme=apikey`
2. `outbound_token_source=PX_GATEWAY_API_KEY`

## 8. 两种模式日志落点与触发方式

1. Local Standalone（`POWERX_PROXY=0`）：
   - 日志落点：插件后端进程 stdout（建议 `tee` 到 `"$RUNTIME_LOG_FILE"`）
   - 触发接口：`POST /api/v1/admin/notifications/test` 或 ws-bus 调试入口
   - 预期日志：`topic/status/tenant_uuid/tenant_key/subscriber_id` 等统一字段
2. Delegated / Proxy（`POWERX_PROXY=1`）：
   - 日志落点：同样在插件后端 stdout（不是底座日志）
   - 触发接口：同 standalone，入口仍打插件 `:8078`
   - 额外预期：出现 `component=ws_bus_gateway_auth`，并可检索 `gateway_auth_scheme/outbound_token_source`
