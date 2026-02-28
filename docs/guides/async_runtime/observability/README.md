# Observability（插件侧）

## 1. 范围

1. HTTP/WS 请求链路日志
2. Event/Task 指标
3. proxy 出站鉴权观测字段

## 2. 建议最小字段

1. `trace_id`
2. `topic`
3. `tenant_uuid`
4. `status`
5. `gateway_auth_scheme`
6. `outbound_token_source`

## 3. 最小指标

1. `plugin_event_bridge_emit_total`
2. `plugin_event_bridge_drop_total`
3. `plugin_event_bridge_latency_ms`

## 4. 最小验证命令

```bash
curl -sS http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_'
```

## 5. proxy 场景关键判定

日志需看到：

1. `gateway_auth_scheme=apikey`
2. `outbound_token_source=PX_GATEWAY_API_KEY`
