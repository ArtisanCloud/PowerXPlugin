# 插件 WS Bus 适配规范（宿主/standalone 统一接口）

> 目标：**业务层不感知模式**。任何 PowerX 插件在宿主模式下挂钩 PowerX 底座 WS；standalone 模式下使用插件内置 WS Hub；对业务层保持统一订阅/发布语义。

## 1. 统一接口（前端）

插件前端必须只暴露统一 API，例如：

```ts
subscribe(topic, handler)
unsubscribe(topic, handler)
connect()
disconnect()
```

业务页面**禁止**直接拼 WS URL 或判断宿主/standalone。

> 联调与验收步骤统一参考：
> - `docs/guides/async_runtime/websocket/debug_playbook.md`
> - `docs/guides/async_runtime/event_fabric/integration_playbook.md`

## 2. 运行模式切换（底层）

- **宿主模式**：连接 PowerX 底座 `/api/ws`
- **standalone 模式**：连接插件自身 `/api/ws`
- **协议统一**：与 `PowerX/docs/plan/wx/WS-NOTIFY.md` 完全一致

## 3. 连接地址规范

- PowerX 底座 WS Bus：`/api/ws`
- 插件 standalone WS（唯一入口）：`/api/ws`
- 插件侧仅暴露 `/api/ws`

## 4. 鉴权与租户透传

- `?authorization=Bearer <token>`
- 或子协议：`Sec-WebSocket-Protocol: bearer.<b64url(jwt)>`
- 可选 `tenant_uuid` query/body 兜底
- `POST /api/v1/admin/runtime/ws-bus/publish` 的 tenant 解析优先级：入站 token/上下文 `tid` > 请求体 `tenant_uuid`（仅无入站租户时） > `gateway.tenant_uuid`
- 若请求体 `tenant_uuid` 与入站 token `tid` 不一致，返回 `tenant mismatch`（403）

## 5. 协议（简版提醒）

- 客户端 → 服务端
```json
{ "type": "subscribe", "topics": ["_topic.template.update"] }
```

- 服务端 → 客户端
```json
{ "type": "event", "topic": "_topic.template.update", "payload": { /*...*/ } }
```

> 详细字段与 envelope 以底座规范为准。

## 6. 降级策略

- WS 断线/不可用 → 重连与状态恢复，不以页面轮询驱动执行
- 断线自动重连 + 恢复订阅

## 7. 发布接口（后端统一入口）

> 目标：插件业务层只调用 **统一发布 API**，宿主/standalone 差异由 framework 内部处理。

### 7.0 本次实现范围

- 本仓库仅覆盖 **PowerXPlugin framework 与插件侧适配**。
- PowerX 底座发布入口由 PowerX 项目实现（接口契约已在此文档约定）。

### 7.1 Framework SDK 约定

插件后端统一使用：

```ts
// framework sdk
publish(topic: string, payload: Record<string, any>, options?: {
  tenant_uuid?: string
  trace_id?: string
})
```

### 7.2 宿主模式（Host）

- 通过底座发布入口转发：`POST /api/v1/admin/runtime/ws-bus/publish`
  - 启动时注册 topic：`POST /api/v1/admin/runtime/ws-bus/grant`
  - 注册失败不会阻塞插件启动，仅记录日志
- 请求体示例：
```json
{
  "topic": "_topic.template.update",
  "payload": { "percent": 20, "message": "syncing" },
  "tenant_uuid": "00000000-0000-0000-0000-000000000001",
  "trace_id": "..."
}
```
- 仅宿主/插件内部可调用，必须鉴权与校验租户

### 7.3 Standalone 模式

- 直接发布到插件本地 WS Bus（`/api/ws` 对应的 hub）
- 与宿主模式使用 **同一套 topic + payload** 结构

### 7.4 Topic 白名单

- 业务必须使用白名单 topic
- 当前示例基线为 `_topic.template.update`，统一采用 `_topic.*` 命名

## 8. 验收点（发布链路）

- 宿主模式：插件发布 → 底座 WS Bus → 前端 `/api/ws` 订阅可收到
- standalone：插件发布 → 插件 `/api/ws` → 前端订阅可收到
- 断线重连后仍能继续接收进度

### 8.1 最小联调命令（便于回归）

- standalone（`POWERX_PROXY=0`）
  - 订阅：`wscat -c "ws://127.0.0.1:8078/api/ws?authorization=Bearer $USER_TOKEN"`
  - 发布：`POST /api/v1/admin/runtime/ws-bus/publish`
  - 预期：收到 `ack` + `event`

- 宿主联调（`POWERX_PROXY=1`）
  - 订阅：`wscat -c "ws://127.0.0.1:8077/api/ws?authorization=Bearer $USER_TOKEN"`
  - 注册：`POST /api/v1/admin/runtime/ws-bus/grant`（调插件 8078，转发到底座）
  - 发布：`POST /api/v1/admin/runtime/ws-bus/publish`（调插件 8078，转发到底座）
  - 预期：底座订阅端收到 `event`

## 9. 本地调试发布端点（standalone）

> 仅用于开发调试。Gin 与 FastAPI 均提供，调试路由默认注册。

- `POST /api/v1/admin/runtime/ws-bus/publish`
  - 宿主模式下该端点会转发到 PowerX 底座 publish 接口，便于手动联调
- `POST /api/v1/admin/runtime/ws-bus/grant`
  - 宿主模式下该端点会转发到 PowerX 底座 register 接口
  - standalone 模式下该端点为 no-op（返回规范化后的 topics），不作为订阅前置条件
- WebSocket 订阅端点（standalone）：`GET /api/ws`

## 10. 中间件注意事项

- WebSocket 握手请求是长连接升级，不应套用普通 HTTP 超时中间件。
- 当前实现已对 `Upgrade: websocket` 请求跳过 `Timeout(30s)`，避免 `response.Write on hijacked connection`。
- Body 示例：
```json
{
  "topic": "_topic.template.update",
  "payload": { "percent": 20, "message": "syncing" },
  "tenant_uuid": "00000000-0000-0000-0000-000000000001",
  "trace_id": "trace-123"
}
```
