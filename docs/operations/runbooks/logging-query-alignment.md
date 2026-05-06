# 统一日志查询对齐（RuntimeLogger）

本文用于插件本地/宿主两种模式下，统一查询 `labels + 顶层 fields + meta`。

## 1. 字段约定

- 全链路主键（强制）：
  - `request_id`
  - `trace_id`
- 目标：同一个外部请求可用单个 `request_id` 串起网关/代理/插件/审计日志。

- 固定 labels（平台/运行态，低基数）：
  - `system`
  - `service`
  - `env`
  - `instance`
  - `module`
- 业务 labels（白名单）：
  - `biz_scene`
  - `biz_domain`
- 高基数字段（仅正文顶层 fields，不进 labels）：
  - `plugin_id`
  - `tenant_uuid`
  - `session_id`
  - `message_id`
  - `trace_id`
  - `request_id`
  - `user_id`

## 2. 链路日志最小字段（强制）

- 必选字段：
  - `timestamp`
  - `level`
  - `system`
  - `service`
  - `env`
  - `instance`
  - `module`
  - `plugin_id`
  - `tenant_uuid`
  - `request_id`
  - `trace_id`
- 可选字段：
  - `upstream_request_id`
  - `correlation_id`
  - `method`
  - `path`
  - `status`
  - `latency_ms`
- 命名约束：
  - 仅使用 `request_id` / `trace_id`
  - 禁止使用 `reqId`、`trace` 等别名

## 2.1 顶层字段、labels 与 meta 分层（强制）

- 以下关键字段必须输出为日志 JSON 顶层字段（不可仅放在 `meta` 字符串中）：
  - `plugin_id`
  - `request_id`
  - `trace_id`
  - `tenant_uuid`
  - `path`
  - `status`
  - `component`
- `labels` 只承载低基数索引字段：`system/service/env/instance/module/(可选 level)`。
- `meta` 只承载扩展信息（调试补充、非关键业务上下文），不能替代上述关键字段。
- 查询基线：
  - 优先 `| json | request_id="..."`、`| json | plugin_id="..."`
  - 不允许把关键排障链路长期依赖 `|= "meta contains ..."`。

## 2.2 `meta` 推荐内容（明确）

- 可以放入 `meta` 的内容（示例）：
  - `upstream_body_excerpt`
  - `debug_flags`
  - `raw_payload_size`
  - 临时排障扩展键（上线后可移除）
- 不应只放在 `meta` 的内容：
  - `plugin_id`
  - `request_id`
  - `trace_id`
  - `tenant_uuid`
  - `session_id`
  - `message_id`
  - `user_id`

## 2.3 别名标准化映射（强制）

- 入口封装层必须做字段别名归一化，统一映射到标准键：
  - `requestId` / `reqId` -> `request_id`
  - `trace` / `traceId` -> `trace_id`
  - `tid` / `tenantId` -> `tenant_uuid`
  - `plugin` / `pluginId` -> `plugin_id`

## 3. 推荐查询方式

先用低基数 labels 缩小范围，再按正文字段精确过滤。

### 示例 A：按模块筛选，再按插件字段精确过滤

```logql
{service="backend",module="agent"}
| json
| plugin_id="com.powerx.plugins.xxx"
```

### 示例 B：在 A 的基础上追加租户/会话

```logql
{service="backend",module="agent"}
| json
| plugin_id="com.powerx.plugins.xxx"
| tenant_uuid="6b5d..."
| session_id="sess_123"
```

### 示例 C：排查 delegated 登录失败

```logql
{service="backend",biz_domain="iam"}
| json
| biz_scene="auth_login"
| status="failed"
```

### 示例 D：排查 runtime 调度/投递

```logql
{service="backend",biz_domain="runtime_ops"}
| json
| topic=~"runtime_ops|wsbus|taskbus"
```

### 示例 E：按 request_id 串联全链路（JSON）

```logql
{system="powerx"}
| json
| request_id="$request_id"
```

### 示例 F：按 request_id 串联全链路（文本）

```logql
{system="powerx"} |= "request_id=$request_id"
```

## 4. 对齐检查清单

- 新增日志调用时，优先使用 `Deps.RuntimeLogger(...)`。
- 组件名（`component`）使用稳定命名，建议 `domain.subdomain.action`。
- 如有明确业务语义，显式传入：
  - `biz_scene`
  - `biz_domain`
- 不要把 `tenant_uuid/session_id/trace_id/request_id` 放进 labels 白名单。
- `_p` 代理链路日志（API-IN/GATE/PROXY-*）与插件日志必须同时带 `request_id`、`trace_id`。
- `request_id`/`trace_id` 不要提升为 Loki label（避免高基数）。
- `plugin_id` 不要提升为 Loki label；必须保留在日志顶层 fields。
