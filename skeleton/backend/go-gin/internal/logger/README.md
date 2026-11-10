# Logger Wiring (Skeleton)

## 目标

将 skeleton 现有日志初始化与 framework 统一日志能力对齐，作为插件侧接线层。

## 职责边界

- 读取配置并初始化 logger。
- 桥接 logrus/slog 到 framework 统一字段规范。
- 透传 tenant/trace/component 等上下文字段。
- 保持对历史调用点兼容，逐步收口到 framework 门面。

## 不负责

- 不在业务层直接定义 sink 选择策略。
- 不新增插件私有日志协议。

## 启动就绪提示

`cmd/plugin` 从进入 `main` 开始计时，在 Framework 的 `OnHTTPReady` 回调中报告 HTTP 就绪。此时依赖与路由已装配，HTTP 端口已成功绑定；端口冲突不会输出成功提示。耗时不包含 `go run` 的编译时间，也不代表 gRPC 或后台任务已完成启动。

就绪摘要单独写入终端 `stderr`，包含就绪时间、启动耗时和可访问地址。因此 `runtime.logging.output: file` 或较高日志级别不会隐藏该摘要，常规日志仍按原配置输出。具体文案来自 `internal/observability/startup/locales/`。

同时以事件 `plugin.http.ready`、消息码 `PLUGIN_HTTP_READY` 写入 INFO 结构化日志，保留 `started_at`、`ready_at`、`startup_duration_ms`、`listen_addr`、`url`、`plugin_id` 和 `mode`。当日志级别高于 INFO 时，该结构化事件遵循级别过滤，终端摘要仍可见。

修改后按原方式重启插件后端即可看到提示，无需更改日志输出配置。若没有成功提示，检查当前进程的监听端口以及 `logs/runtime.log`；单凭“开始启动”日志不能判断已就绪。
