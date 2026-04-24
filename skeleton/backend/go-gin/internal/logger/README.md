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
