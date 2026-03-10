# Quickstart: Framework WS Bus Adapter

## 目标

统一插件发布进度事件的调用方式，宿主与 standalone 模式自动切换。

## 使用流程

1) 在插件后端调用 framework SDK：

```ts
publish("_topic.template.update", { percent: 20, message: "syncing" }, { tenant_uuid, trace_id })
```

2) 宿主模式：framework 自动转发至底座发布入口：

```
POST /api/v1/admin/runtime/internal/ws-bus/publish
```

同时在启动时注册 topic：

```
POST /api/v1/admin/runtime/internal/ws-bus/grant
```

注册失败不会阻塞插件启动，仅记录日志。

3) standalone 模式：framework 直接发布到本地 WS Bus。

## 本地调试发布端点（standalone）

> 仅用于开发调试（Gin/FastAPI 皆提供）。

```
POST /api/v1/admin/runtime/internal/ws-bus/publish
{
  "topic": "_topic.template.update",
  "payload": { "percent": 20, "message": "syncing" },
  "tenant_uuid": "00000000-0000-0000-0000-000000000001"
}
```

```
POST /api/v1/admin/runtime/internal/ws-bus/grant
{
  "topics": ["_topic.template.update", "_topic.audit.template.updated"],
  "tenant_uuid": "00000000-0000-0000-0000-000000000001"
}
```

## 验收

- 宿主模式前端通过 `/api/ws` 订阅可收到消息
- standalone 模式前端通过 `/api/ws` 订阅可收到消息
- 任务执行链路由 Task 机制驱动，页面仅消费 WS 事件
