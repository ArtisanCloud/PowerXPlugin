# 统一日志查询对齐（RuntimeLogger）

本文用于插件本地/宿主两种模式下，统一查询 `labels + fields`。

## 1. 字段约定

- 固定 labels（平台/运行态）：
  - `system`
  - `service`
  - `env`
  - `instance`
  - `module`
  - `plugin_id`
- 业务 labels（白名单）：
  - `biz_scene`
  - `biz_domain`
- 高基数字段（仅正文 fields，不进 labels）：
  - `tenant_uuid`
  - `session_id`
  - `message_id`
  - `trace_id`
  - `request_id`
  - `user_id`

## 2. 推荐查询方式

先用低基数 labels 缩小范围，再按正文字段精确过滤。

### 示例 A：按模块 + 插件筛选

```logql
{service="backend",module="agent",plugin_id="com.powerx.plugins.xxx"}
| json
```

### 示例 B：在 A 的基础上追加租户/会话

```logql
{service="backend",module="agent",plugin_id="com.powerx.plugins.xxx"}
| json
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

## 3. 对齐检查清单

- 新增日志调用时，优先使用 `Deps.RuntimeLogger(...)`。
- 组件名（`component`）使用稳定命名，建议 `domain.subdomain.action`。
- 如有明确业务语义，显式传入：
  - `biz_scene`
  - `biz_domain`
- 不要把 `tenant_uuid/session_id/trace_id/request_id` 放进 labels 白名单。

