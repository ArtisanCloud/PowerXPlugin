# Quickstart: Framework WS Bus Adapter

## 1. 目标

验证 WS Bus 适配在两种模式下都可工作，并确认“业务写路径触发事件，前端仅消费事件”。

## 2. 前置条件

```bash
export PLUGIN_BASE_URL="http://127.0.0.1:8078/api/v1"
export USER_TOKEN="<plugin-user-token>"
```

并准备：

1. Standalone：`POWERX_PROXY=0`
2. Proxy：`POWERX_PROXY=1` + 可用网关契约凭证（Bearer 或 ApiKey）
3. 已在 `plugin.yaml` 声明 `_topic.template.update` 的 publish 权限

Proxy 凭证要求：

1. `PX_GATEWAY_AUTH_SCHEME=bearer` 时，使用 `PX_PLUGIN_TOOL_TOKEN`
2. `PX_GATEWAY_AUTH_SCHEME=apikey` 时，使用 `PX_GATEWAY_API_KEY`
3. `grant/publish` 默认不透传当前 delegated/user bearer

## 3. 验证步骤

### 3.1 建立订阅

使用 `wscat` 连接插件 `/api/ws` 并订阅 `_topic.template.update`。

### 3.2 触发真实业务写路径

```bash
# A. 创建模板（真实业务入口）
curl -sS -X POST "$PLUGIN_BASE_URL/templates" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"ws-template-demo","description":"ws bus e2e","content":"hello template"}'
```

记下响应中的 `data.id`，再执行：

```bash
# B. 更新模板（真实业务入口）
curl -sS -X PUT "$PLUGIN_BASE_URL/templates/{id}" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"ws-template-demo","description":"ws bus e2e updated","content":"hello template updated"}'
```

### 3.3 Proxy 模式下 ACL 准备（如需）

```bash
curl -sS -X POST "$PLUGIN_BASE_URL/admin/runtime/internal/ws-bus/grant" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topics":["_topic.template.update"]}'
```

## 4. 验收标准

1. Create/Update 两次业务请求均成功（2xx）。
2. WS 订阅端收到 `_topic.template.update` 事件。
3. payload 至少包含：`action`（`created/updated`）、`template_id`、`tenant_uuid`、`trace_id`。
4. 页面不触发任务执行，仅消费事件更新（满足 SC-003）。
5. 若 WS 发布失败，模板 CRUD 主流程仍成功，失败信息写入日志告警。

## 5. 调试端点（仅联调辅助）

以下端点保留用于链路诊断，不作为业务事件主入口：

1. `POST /api/v1/admin/runtime/internal/ws-bus/publish`
2. `POST /api/v1/admin/runtime/internal/ws-bus/grant`
3. `POST /api/v1/admin/runtime/ws-bus/test-flow`（页面统一入口）

## 6. 宿主内嵌模式补充

1. 前端 WS 连接应命中宿主 `/api/ws`，不应命中 `/_p/<plugin>/api/ws`
2. 页面 `framework-lab` 使用统一 `test-flow` 接口执行 `grant->publish`
3. 验收建议：
   - `Grant=ok`
   - `Publish=ok`
   - `ack_ok=true`
   - `event_ok=true`
