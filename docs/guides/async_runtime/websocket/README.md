# WebSocket 子系统说明（插件侧）

> 对齐 PowerX `async_runtime/websocket` 结构。

## 1. 范围

1. 连接入口：`GET /api/ws`
2. 订阅协议：`subscribe/unsubscribe/ping`
3. 推送包结构：`welcome/ack/error/event`
4. 单连接多 topic 复用

## 2. 核心约束

1. 单页面尽量复用单条 WS 连接
2. topic 命名统一 `_topic.*`
3. 未授权 topic 必须返回 `error`
4. WebSocket 只做实时分发，不替代任务执行

## 3. 鉴权与模式边界

1. 访问插件 WS（`:8078/api/ws`）使用 Bearer（插件入站鉴权）
2. proxy 场景下插件转发到底座按网关配置分流：
   - Host：Bearer
   - Standalone + Proxy：ApiKey
3. proxy 联调前需先在底座创建 API Key（profile `permission_ids` 已覆盖）
