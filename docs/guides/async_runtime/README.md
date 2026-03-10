# PowerXPlugin 异步运行时总览（对齐 PowerX）

> 插件侧入口，结构与 PowerX `docs/guides/async_runtime` 同构。

## 1. 系统边界（插件视角）

1. 事件语义层（`_topic.*`）：表达“发生了什么”
2. 任务执行层（Task/EventBridge）：表达“如何执行”
3. 实时分发层（WebSocket）：表达“谁实时收到”
4. 调度层（Scheduler/Cron）：表达“何时触发”

## 2. 一条完整链路

1. 插件业务触发事件（`emit`）
2. 权限与命名校验（manifest + ACL）
3. TaskBus/EventBridge 投递（local/host）
4. 执行结果回流为事件
5. WS 推送到页面或订阅端
6. 指标与日志记录链路状态

## 3. 插件与底座边界（关键）

1. 插件业务层只依赖 framework 抽象，不写 Host/Standalone 分支
2. Proxy 场景租户由底座按凭证解析，插件不推导租户
3. 凭证分流：
   - 插件调试入口入站：Bearer（访问 `:8078`）
   - 插件出站到底座：Host=Bearer，Standalone+Proxy=ApiKey
4. Topic 二段校验：
   - 先校验 topic 资源存在（`event_topics`）
   - 再校验主体权限（API Key/Profile 或 Token）
   - `grant` 只绑定 ACL，不创建 topic

## 4. 文档导航

1. 命名规范：`docs/guides/async_runtime/event_fabric/naming_convention.md`
2. Event 联调：`docs/guides/async_runtime/event_fabric/integration_playbook.md`
3. Task 子系统：`docs/guides/async_runtime/task/README.md`
4. Task 机制：`docs/guides/async_runtime/task/mechanism.md`
5. WS 子系统：`docs/guides/async_runtime/websocket/README.md`
6. WS 实操：`docs/guides/async_runtime/websocket/debug_playbook.md`
7. Scheduler：`docs/guides/async_runtime/scheduler/README.md`
8. ACL：`docs/guides/async_runtime/acl_security/README.md`
9. 可观测性：`docs/guides/async_runtime/observability/README.md`
10. 配置字段：`docs/guides/config-reference.md`

## 5. 推荐联调顺序

1. `event_fabric/integration_playbook.md`（先验证 `emit -> metrics`）
2. `websocket/debug_playbook.md`（再验证 `create topic -> grant -> publish/subscribe`）
3. `task/README.md`（确认任务主链路）
4. `scheduler/README.md`（确认调度进入同一链路）
