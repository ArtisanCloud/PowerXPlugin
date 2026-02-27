# Scheduler / Cron（插件侧）

## 1. 范围

1. 何时触发任务（调度）
2. 触发后是否进入同一 Task/Event 链路
3. 触发结果是否可通过 WS 与指标观察

## 2. 约束

1. Scheduler 只负责触发，不替代 Task 执行
2. 页面不通过轮询驱动任务执行
3. 手动触发与定时触发必须进入同一链路

## 3. 手工验证（最短）

### Step 1：准备 WS 订阅

1. standalone：订阅 `ws://127.0.0.1:8078/api/ws`
2. standalone + proxy：按 websocket playbook 准备 proxy 订阅与凭证

### Step 2：触发调度窗口或手动触发

1. 触发后应看到 `_topic.*` 事件推送
2. 并看到 `plugin_event_bridge_emit_total` 增长

### Step 3：结果判定

1. 收到 `ack` + `event`
2. 日志存在 `trace_id/topic` 链路字段

## 4. 常见问题

1. 有调度日志无 WS 事件：检查 topic ACL / grant / subscribe
2. proxy 下无事件：检查 ApiKey 配置与底座权限快照是否已更新
