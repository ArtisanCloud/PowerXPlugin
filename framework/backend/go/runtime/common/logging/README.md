# Runtime Logging (Framework)

## 目标

为插件提供统一日志门面与策略路由能力，避免业务代码直连具体日志实现。

## 职责边界

- 统一日志调用入口（context + fields + message）。
- 统一策略模型（mode/sinks/format/level/retry）。
- 统一 sink 路由与失败结果结构。
- 固定低基数标签基线：`plugin_id`, `tenant_uuid`, `component`, `level`。

## 约束

- 宿主模式默认 `stdout + json`。
- 扩展 sink（`file`/`loki`）需显式授权。
- sink 失败不能阻断主业务链路，必须可观测。
