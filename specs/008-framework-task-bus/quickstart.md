# Quickstart — Framework TaskBus Event Bridge

## 1) 本地模式（EventBridge 关闭）

适用于本地/离线/未接入宿主 TaskBus 的场景：

- `event_bridge.enabled=false`
- 业务服务通过统一 Emitter 发事件（本地实现记录日志/内存队列），consumer 可在进程内直接处理或跳过。

## 2) TaskBus 模式（EventBridge 开启）

适用于与宿主共享事件流/指标/后台任务调度的场景：

- `event_bridge.enabled=true`
- 在插件清单中按最小权限声明发布/订阅的 topic（精确到版本号，尽量不用 `*`）：
  - 开发态：`skeleton/plugin.yaml`
  - 宿主/发布态：以宿主加载的 manifest 为准（字段保持一致）

## 3) 验证：TaskBus 不可用时自动降级

当 `event_bridge.enabled=true` 但 TaskBus 不可用时：

- 系统自动降级到本地实现
- 记录告警/指标（主流程不 panic）

## 4) 契约文件

- Channel 事件契约：`specs/008-framework-task-bus/contracts/channel-events.yaml`
- 契约变更需 PR 评审 + CI 校验（topic 唯一、必填 meta 齐全）

## 5) 幂等规则

默认去重 key：

`topic + tenant_uuid + trace_id`

consumer 必须按该规则做幂等去重（at-least-once 语义）。

## 6) Staging 验证清单（对应 SC-002）

- 开启：`event_bridge.enabled=true`
- 确认清单权限：在插件 manifest 中声明 publish/subscribe（开发态见 `skeleton/plugin.yaml`），并按最小权限精确到 topic + 版本号
- 触发：选定一个已接入的 Topic（例如 `powerx.channel.master.credential_inspection.v1`）触发一次业务动作
- 观察：consumer 有落地结果（DB/日志/任务中心回执其一即可），且 metrics 中 `emit/consume` 的成功计数增加、错误计数不持续增长
- 回滚：切回 `event_bridge.enabled=false` 后核心业务仍可用（本地实现兜底）

## 7) 运维信号（对应 SC-004）

- 指标：至少应能看到 emit/consume 的成功/失败计数与延迟（本地与 TaskBus 模式一致口径）
- 告警建议：TaskBus 模式下 `emit_error_rate` 或 `consume_error_rate` 持续升高、或连续 N 分钟无消费成功时告警
- 回滚策略：优先切回 `event_bridge.enabled=false`（本地实现）；必要时关闭双写并保留契约校验
