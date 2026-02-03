# 插件 WS Bus 适配规范（宿主/standalone 统一接口）

> 目标：**业务层不感知模式**。任何 PowerX 插件在宿主模式下必须挂钩 PowerX 底座 WS；standalone 模式下使用插件自建 WS；两者对业务代码接口一致。

## 1. 统一接口（前端）

插件前端必须只暴露统一 API，例如：

```ts
subscribe(topic, handler)
unsubscribe(topic, handler)
connect()
disconnect()
```

业务页面**禁止**直接拼 WS URL 或判断宿主/standalone。

## 2. 运行模式切换（底层）

- **宿主模式**：连接 PowerX 底座 `/api/ws`
- **standalone 模式**：连接插件自身 `/ws`
- **协议统一**：与 `PowerX/docs/plan/wx/WS-NOTIFY.md` 完全一致

## 3. 连接地址规范

- PowerX 底座 WS Bus：`/api/ws`（兼容 `/ws`）
- 插件 standalone WS：`/ws`（可选兼容 `/api/ws`）
- **不使用** `/api/v1/ws`

## 4. 鉴权与租户透传

- `?authorization=Bearer <token>`
- 或子协议：`Sec-WebSocket-Protocol: bearer.<b64url(jwt)>`
- 可选 `tenant_uuid` query 兜底

## 5. 协议（简版提醒）

- 客户端 → 服务端
```json
{ "type": "subscribe", "topics": ["org_sync.progress"] }
```

- 服务端 → 客户端
```json
{ "type": "event", "topic": "org_sync.progress", "payload": { /*...*/ } }
```

> 详细字段与 envelope 以底座规范为准。

## 6. 降级策略

- WS 断线/不可用 → 轮询兜底
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
- 请求体示例：
```json
{
  "topic": "org_sync.progress",
  "payload": { "percent": 20, "message": "syncing" },
  "tenant_uuid": "00000000-0000-0000-0000-000000000001",
  "trace_id": "..."
}
```
- 仅宿主/插件内部可调用，必须鉴权与校验租户

### 7.3 Standalone 模式

- 直接发布到插件本地 WS Bus（`/ws` 对应的 hub）
- 与宿主模式使用 **同一套 topic + payload** 结构

### 7.4 Topic 白名单

- 业务必须使用白名单 topic
- `org_sync.progress` 为首个必须支持的 topic

## 8. 验收点（发布链路）

- 宿主模式：插件发布 → 底座 WS Bus → 前端 `/api/ws` 订阅可收到
- standalone：插件发布 → 插件 `/ws` → 前端订阅可收到
- 断线重连后仍能继续接收进度
