# Quickstart: Framework WS Bus Adapter

## 目标

统一插件发布进度事件的调用方式，宿主与 standalone 模式自动切换。

## 使用流程

1) 在插件后端调用 framework SDK：

```ts
publish("org_sync.progress", { percent: 20, message: "syncing" }, { tenant_uuid, trace_id })
```

2) 宿主模式：framework 自动转发至底座发布入口：

```
POST /api/v1/admin/runtime/ws-bus/publish
```

3) standalone 模式：framework 直接发布到本地 WS Bus。

## 验收

- 宿主模式前端通过 `/api/ws` 订阅可收到消息
- standalone 模式前端通过 `/ws` 订阅可收到消息
- WS 不可用时仍可轮询兜底
