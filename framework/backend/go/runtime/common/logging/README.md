# Runtime Logging

本目录是 framework runtime logging 的独立实现目录，不依赖 Gin skeleton。

## 已提供能力

- 统一日志接口、runtime 字段规范、context 字段注入与 `Facade`。
- `slog` / `logrus` adapter，用于框架内部或宿主接线层兼容。
- policy 决议与校验：`host` / `standalone`、`stdout` / `stderr` / `file` / `loki`、格式、级别、重试。
- sink registry、fan-out router、sink outcome 与重试退避。
- 内置 `stdout` / `stderr` writer sink。
- 内置 `file` sink，支持父目录创建、lumberjack rotate、过期轮转文件清理。
- 宿主模式默认策略：`POWERX_PROXY=1` 或调用方显式 host mode 时强制 `stdout + json`，直连 file/loki 需由策略授权。

## 边界

- skeleton 只能作为消费方或兼容接线层，不承载 framework logger 的核心实现。
- PowerX 底座策略拉取/订阅应通过本包 policy/registry/router 接入；插件业务代码不应自定义日志协议。
- Loki sink 当前保留为可注册 sink 类型，具体客户端可由 PowerX host adapter 或后续 framework 扩展提供。
