# Event Fabric Integration Playbook（插件侧联调）

## 1. 前置条件

```bash
export PLUGIN_BASE_URL="http://127.0.0.1:8078/api/v1"
export USER_TOKEN="<plugin-user-token>"
```

并确保：

1. `skeleton/plugin.yaml` 已声明目标 `_topic.*`
2. 执行层文件 `config/event_fabric.yaml` 已同步（框架会按多路径兼容扫描）
3. 后端默认强鉴权
4. proxy 场景已准备可用出站凭证，并配置到插件：
   - 标准与默认：`PX_GATEWAY_AUTH_SCHEME=bearer` + `PX_PLUGIN_TOOL_TOKEN=<token>`
   - ApiKey（仅 local+proxy 联调可选）：`PX_GATEWAY_AUTH_SCHEME=apikey` + `PX_GATEWAY_API_KEY=<key>`

## 1.1 Standalone+Proxy 标准流程（对齐 PowerX）

1. 插件声明所需 `topics + actions`。
2. 通过插件接口创建 topic：`POST /admin/runtime/event-fabric/topics`。
3. Proxy 绑定 profile 的 topic 权限（permission_ids）。
4. 若走 ApiKey，轮换/新建 API Key（权限快照生效）。
5. 插件再代理调 `POST /api/v1/admin/runtime/ws-bus/grant`。
6. 最后再执行 `publish` 与 WS `subscribe` 联调。

## 1.2 最终规则（必须遵守）

1. Topic 真相源是底座 `event_topics`。
2. `POST /admin/runtime/event-fabric/topics` 只做 topic 创建/登记（插件代理到底座）。
3. `POST /admin/runtime/ws-bus/grant` 只做 ACL 绑定，不创建 topic。
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
2. `POST /admin/runtime/event-fabric/topics`
3. `POST /admin/runtime/ws-bus/grant`
4. `POST /admin/runtime/ws-bus/publish`
5. `GET /admin/runtime/metrics`

> 注意：`ws-bus/grant` 只做授权绑定，不创建 topic。  
> 新 topic 必须先通过插件接口创建资源：`POST /admin/runtime/event-fabric/topics`（由插件代理到底座）。

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

## 6.1 Step 4：任务驱动 + 事件消费验证（015 对齐）

目标：验证“业务写路径触发事件”，而不是只验证 runtime 调试接口。

1. 先建立 WS 订阅（推荐 `wscat` 订阅 `_topic.template.update`）。
2. 调用真实业务接口创建模板（`POST /templates`）。
3. 再调用真实业务接口更新模板（`PUT /templates/{id}`）。
4. 预期每次业务写操作都会收到 `_topic.template.update` 事件，前端只消费事件，不依赖页面轮询触发执行。

示例命令：

```bash
# A. 创建模板（业务入口，不是 runtime 调试端点）
curl -sS -X POST "$PLUGIN_BASE_URL/templates" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"ws-template-demo","description":"ws bus e2e","content":"hello template"}'

# B. 更新模板（将 {id} 替换为创建结果中的 data.id）
curl -sS -X PUT "$PLUGIN_BASE_URL/templates/{id}" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"ws-template-demo","description":"ws bus e2e updated","content":"hello template updated"}'
```

验收口径：

1. 两次请求 HTTP 均成功（2xx）。
2. WS 订阅端收到 `_topic.template.update` 的 `event`。
3. payload 至少包含：`action`（`created/updated`）、`template_id`、`tenant_uuid`、`trace_id`。
4. 若 WS 发布失败，不影响模板 CRUD 主流程（主流程成功，日志可见告警）。

## 7. 模式差异

1. standalone：所有联调均在插件本地闭环
2. standalone + proxy：
   - 入站请求仍打插件 `:8078`（Bearer）
   - 插件出站到底座按 `gateway.auth_scheme` 走 Bearer 或 ApiKey
   - 租户由底座按凭证解析
   - topic 与 key 权限准备由 Proxy 负责，不由插件业务逻辑直接处理

## 8. 常见故障最短定位

1. `event permission denied`：`plugin.yaml` 未授权该 topic
2. `401`：出站凭证错误（确认 `gateway_auth_scheme` 与对应凭证是否匹配）
3. `403 topic not allowed`：底座 ACL / profile 权限不足
4. `ack` 无 `event`：topic 不一致或未完成 grant

## 9. 代码映射

1. `framework/backend/go/eventbridge/emitter.go`
2. `framework/backend/go/eventbridge/local_emitter.go`
3. `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/event_bridge_debug_handler.go`
4. `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_grant.go`
5. `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_publish.go`
6. `skeleton/backend/go-gin/internal/transport/http/admin/templates/template_handler.go`

## 10. 外部插件迁移清单（TaskBus Adapter Mapping / T047）

目标：把外部插件从“直连本地实现”迁移到 framework `event_bridge` 统一抽象，最少 1 次迭代完成。

### 10.1 版本与依赖

1. 升级 framework 依赖到 `v0.0.4-alpha`（或更新）。
2. 运行 `go mod tidy` 并确认不存在旧版 event bridge 包冲突。

### 10.2 配置映射

1. 新增/校验 `event_bridge`：
   - `enabled`
   - `mode=local|taskbus|dual`
   - `fallback_to_local`
   - `taskbus_provider=host|redis`
2. 迁移旧配置到新语义：
   - 旧“直接写 ws/queue”配置 -> `event_bridge.*`
   - 旧“环境分支”逻辑 -> 启动期 provider 决策

### 10.3 代码映射（Adapter）

1. 业务发事件统一改为 `EventEmitter.Emit(...)`，禁止业务层直接写 ws publish。
2. 在启动期注入 `Factory.WithTaskBusProvider(...)`。
3. 保留 fallback 语义：provider 不可用时自动回落 local（按配置）。

### 10.4 权限与契约

1. 在 `plugin.yaml`（或 `plugin.d/events.yaml`）声明 topic + actions（最小权限）。
2. 运行契约校验：`./scripts/contracts/validate-taskbus-contracts.sh`。
3. 确认 topic 命名与版本后缀一致（例如 `*.v1`）。

### 10.5 验收清单

1. `mode=taskbus` + provider 可用：事件可投递。
2. `mode=taskbus` + provider 异常 + `fallback_to_local=true`：可自动回落。
3. `mode=dual`：主链路成功后本地副本写入成功。
4. 指标可见：`emit_total` / `consume_total` / `latency_ms` / `drop_total`。
5. 日志可检索：`topic`、`tenant_uuid`、`trace_id`、`status`。
