# 配置字段参考（完整）

本文说明插件后端 `skeleton/backend/etc/config.yaml` 的完整配置字段。  
推荐以 `runtime.*` 作为运行时能力的统一入口。

## 1. 顶层结构

```yaml
server:
database:
runtime:
context:
host:
security:
gateway:
customer_auth:
grpc_upstream:
grpc_server:
marketplace:
operations:
admin_console:
```

> 兼容说明：历史顶层 `event_bridge/ws_bus/runtime_ops/monitoring/logging/integration` 仍可读，但建议统一写入 `runtime.*`。

## 2. 基础配置

### 2.1 `server`
- `bind_addr`：HTTP 监听地址（如 `:8078`）
- `api_prefix`：API 前缀（如 `/api/v1`）
- `mode`：Gin 运行模式（通常由 `runtime.logging.gin_mode` 管理）

### 2.2 `database`
- `driver`：`postgres | sqlite | memory`
- `dsn`：连接串
- `schema`：数据库 schema
- `maxIdleConns/maxOpenConns/connMaxLifetime/connMaxIdleTime`
- `slowThreshold/preferSimpleProtocol/prepareStmt/skipDefaultTx/logLevel/devMode`

### 2.3 `gateway`
- `base_url`
- `auth_scheme`：`bearer | apikey`
- 宿主 delegated + `bearer`：通过 STS token provider 获取 `aud=powerx:api` 的短期 token
- `api_key`：`auth_scheme=apikey` 时必填，仅 standalone 本地联调使用
- `tenant_uuid`：显式租户上下文字段；不再从静态 bearer token 推导
- `timeout/user_agent`
- `use_mock`

### 2.4 `context`
- `hmac_secret/key_id`
- `jwks_url/issuer/audience/ttl`
- `provider_mode`（`local | delegated`）

### 2.5 `security`
- `enable_cors/cors_origins`
- `rate_limit.*`
- `gateway_allowlist/require_tls13/toolgrant_secret`

### 2.5.1 `host`
- `web_admin_origins`：PowerX 宿主注入的 Web Admin 公开访问来源白名单，如 `https://admin.example.com`、`http://localhost:3030`、`http://127.0.0.1:3030`。

`host.*` 是 PowerX 标准 Host Contract 字段。插件可以读取它并映射到自己的私有配置，例如将 `host.web_admin_origins` 合并到 `security.cors_origins`。PowerX 底座不应猜测插件私有字段。

生产环境中，`web_admin_port` 和 `host.web_admin_origins` 是两个不同概念：

- `web_admin_port`：PowerX Web Admin 内部监听端口，通常由 PowerX setup/install 写入。
- `host.web_admin_origins`：浏览器真实访问 PowerX Admin 的公开 Origin；如果使用 Nginx/HTTPS/域名反代，必须由 PowerX 配置 `http_security.web_admin_origins` 或 `POWERX_WEB_ADMIN_ORIGINS` 后下发。

插件宿主模式下做 CORS/Origin 校验时，必须读取 `host.web_admin_origins`；不能只根据内部端口猜测外部域名。

### 2.6 `customer_auth`
- `mode`：`local | delegate`
- `delegate_endpoint/delegate_timeout`
- `jwt_issuer/jwt_audience/jwt_secret`
- `cache_ttl_seconds`

### 2.7 `grpc_upstream`
- `address/token/tenant_uuid/use_tls/ca_cert`
- `sts_client_id/sts_client_secret/sts_audience/sts_scope/sts_ttl`
- `connect_mode`（`eager | lazy`）
- `optional`

### 2.8 `grpc_server`
- `enable`
- `addr/port/port_max_retries`
- `use_tls/cert/key`

## 3. 运行时统一入口（推荐）

## 3.1 `runtime.run_migrate`
- 启动是否执行迁移。

## 3.2 `runtime.event_bridge`
- `enabled`
- `mode`：`local | taskbus | dual`
- `fallback_to_local`
- `local_queue_size`
- `taskbus_provider`：`redis | host`
- `redis_url/redis_stream/redis_group/redis_consumer/redis_max_len`
- `source_plugin/payload_version`

## 3.3 `runtime.ws_bus`
- `provider`：`memory | redis`
- `redis_url`
- `channel`

## 3.4 `runtime.runtime_ops`
- `heartbeat_seconds/heartbeat_misses`
- `quota_window_minutes`
- `restart_backoff_start_seconds/restart_backoff_max_seconds`
- `log_retention_days`
- `cpu_default/memory_default/network_profile`
- `observability.loki_endpoint/tempo_endpoint`
- `alerts.*`

## 3.5 `runtime.monitoring`
- `metrics.enabled/path`
- `health_check.enabled/path`

## 3.6 `runtime.logging`
- `level`：`debug/info/warn/error`
- `format`：`text/json`
- `output`：`stdout/stderr/file`
- `file_path`
- `http_access`
- `gin_mode`：`debug/release`
- `debug_mode`

## 3.7 `runtime.integration`
- `idempotency.provider/redis_url/ttl_hours`
- `envelope.payload_threshold_bytes`
- `webhook.retry_policy/dlq_topic`
- `secrets.rotation_days_default`
- `billing.*`（税务、分润、异步队列、重试）

## 3.8 `runtime.cache`
- 统一运行时缓存入口（预留）
- 建议放全局缓存策略（`provider/redis_url`）

## 3.9 事件声明文件（安装/启用阶段）
- 规范声明源：`plugin.yaml.events.topics[]`
  - `key`：topic 标识
  - `actions`：`publish | subscribe`
  - `description`：描述
- 过渡期执行源：`config/event_fabric.yaml`
  - `topics[].topic`
  - `topics[].acl[].actions[]`
  - `topics[].description`
- 底座扫描优先级（对齐 PowerX）：`config/event_fabric.yaml` → `platform_capabilities/event_fabric.yaml` → `event_fabric.yaml`

## 4. 业务域配置

### 4.1 `marketplace`
- `checklist.*`
- `documentation.*`
- `recommendation.*`
- `license.*`

### 4.2 `operations`
- `support.*`
- `incident.*`
- `sla.*`

### 4.3 `admin_console`
- 管理台专属配置（按模块继续细分）

## 5. 推荐最小基线

### 5.1 standalone（本地）

```yaml
runtime:
  event_bridge:
    enabled: true
    mode: taskbus
    taskbus_provider: redis
```

### 5.2 standalone + proxy / host 联调

```yaml
runtime:
  event_bridge:
    enabled: true
    mode: taskbus
    taskbus_provider: host
gateway:
  auth_scheme: apikey
  api_key: <your-api-key>
```

## 6. 模板同步

变更配置模板后执行：

```bash
npm run sync:templates
npm run sync:templates -- --check
```
