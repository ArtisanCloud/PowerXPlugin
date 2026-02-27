# Event Fabric Integration Playbook（插件侧联调）

## 1. 前置条件

```bash
export PLUGIN_BASE_URL="http://127.0.0.1:8078/api/v1"
export USER_TOKEN="<plugin-user-token>"
```

并确保：

1. `skeleton/plugin.yaml` 已声明目标 `_topic.*`
2. 过渡期执行层文件 `skeleton/config/event_fabric.yaml` 已同步（底座当前扫描该文件）
3. 后端默认强鉴权
4. proxy 场景已准备 API Key，并配置到插件：
   - `PX_GATEWAY_AUTH_SCHEME=apikey`
   - `PX_GATEWAY_API_KEY=<key>`

## 1.1 Standalone+Proxy 标准流程（对齐 PowerX）

1. 插件声明所需 `topics + actions`。
2. 通过插件接口创建 topic：`POST /admin/runtime/internal/event-fabric/topics`。
3. Proxy 绑定 profile 的 topic 权限（permission_ids）。
4. Proxy 轮换/新建 API Key（权限快照生效）。
5. 插件再代理调 `POST /api/v1/internal/ws-bus/grant`。
6. 最后再执行 `publish` 与 WS `subscribe` 联调。

## 1.2 最终规则（必须遵守）

1. Topic 真相源是底座 `event_topics`。
2. `POST /admin/runtime/internal/event-fabric/topics` 只做 topic 创建/登记（插件代理到底座）。
3. `POST /admin/runtime/internal/ws-bus/grant` 只做 ACL 绑定，不创建 topic。
4. 运行时必须二段校验：topic 存在 + 主体有权限，缺一不可。
5. 禁止隐式创建 topic。

## 1.3 声明层 → 执行层映射（过渡期）

1. 规范源：`plugin.yaml.events.topics[]`
2. 执行源：`config/event_fabric.yaml`
3. 字段映射：
   - `events.topics[].key -> topics[].topic`
   - `events.topics[].actions[] -> topics[].acl[].actions[]`
   - `events.topics[].description -> topics[].description`

## 2. 核心接口（插件）

1. `POST /admin/runtime/event-bridge/emit`
2. `POST /admin/runtime/internal/event-fabric/topics`
3. `POST /admin/runtime/internal/ws-bus/grant`
4. `POST /admin/runtime/internal/ws-bus/publish`
5. `GET /admin/runtime/metrics`

> 注意：`ws-bus/grant` 只做授权绑定，不创建 topic。  
> 新 topic 必须先通过插件接口创建资源：`POST /admin/runtime/internal/event-fabric/topics`（由插件代理到底座）。

## 3. 先讲清楚：`emit` / `grant` / `publish` 分工

场景：页面点击“更新模板状态”，希望后端任务链路处理并把进度实时推给前端。

1. `emit`（业务入口）：业务代码发事件，进入 EventBridge/TaskBus 主链路（可入队、重试、观测）。
2. `grant`（ACL 准备）：只在 proxy 调试链路需要，告诉底座“这个主体可访问哪些 topic”。
3. `publish`（WS 推送入口）：用于 WS 实时链路调试，直接向 ws-bus 发布消息验证订阅推送。

结论：`emit` 不等于 `publish`。  
`emit` 关注业务事件流；`publish` 关注 WS 实时广播链路。

## 4. Step 1：事件主链路（emit）

```bash
curl -sS -X POST "$PLUGIN_BASE_URL/admin/runtime/event-bridge/emit" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"_topic.template.update","payload":{"id":"demo","status":"running","progress":25}}'
```

验收：

1. 返回 `ok=true`
2. 返回 `trace_id`
3. 指标 `plugin_event_bridge_emit_total` 增长

> `emit` 是业务发事件入口（进入 EventBridge/TaskBus 主链路），不等价于 `ws-bus/publish`。

## 5. Step 2：指标校验

```bash
curl -sS "$PLUGIN_BASE_URL/admin/runtime/metrics" | rg 'plugin_event_bridge_(emit_total|drop_total|latency_ms)'
```

验收：

1. `emit_total` 增长
2. `latency_ms` 有值

## 6. Step 3：WS 调试链路（grant/publish）

1. `grant`：用于 proxy 场景准备 topic ACL
2. `publish`：用于验证实时推送链路
3. 详细步骤见：`docs/guides/async_runtime/websocket/debug_playbook.md`

## 7. 模式差异

1. standalone：所有联调均在插件本地闭环
2. standalone + proxy：
   - 入站请求仍打插件 `:8078`（Bearer）
   - 插件出站到底座走 ApiKey
   - 租户由底座按凭证解析
   - topic 与 key 权限准备由 Proxy 负责，不由插件业务逻辑直接处理

## 8. 常见故障最短定位

1. `event permission denied`：`plugin.yaml` 未授权该 topic
2. `401`：出站凭证错误（应确认 `gateway_auth_scheme=apikey`）
3. `403 topic not allowed`：底座 ACL / profile 权限不足
4. `ack` 无 `event`：topic 不一致或未完成 grant

## 9. 代码映射

1. `framework/backend/go/eventbridge/emitter.go`
2. `framework/backend/go/eventbridge/local_emitter.go`
3. `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/event_bridge_debug_handler.go`
4. `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_grant.go`
5. `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_publish.go`
