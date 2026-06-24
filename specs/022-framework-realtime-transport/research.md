# Research: Framework Realtime Transport

## Decision 1: 新建 022，而不是扩展 008/015/021

**Decision**: 新建 `022-framework-realtime-transport`。

**Rationale**: 008 是后端 EventBridge/TaskBus，015 是 WS Bus Adapter，021 是 Agent Skill Bridge 业务场景。当前需求是跨 WS/SSE 的实时传输治理，范围更接近 framework runtime transport。

**Alternatives considered**:

- 放入 008：会把浏览器连接生命周期和 SSE proxy 混入 TaskBus，边界不清。
- 放入 015：015 只覆盖 WS，无法承载 SSE 与 Agent stream-through。
- 放入 021：会把基础设施治理塞进 Agent 业务 feature，影响验收。

## Decision 2: 保留 wsbus/ssebus，新增 realtime facade

**Decision**: 不删除现有 `wsbus` 和 `ssebus`，新增 `runtime/realtime` facade 统一 scope、envelope、validation、client contracts。

**Rationale**: wsbus 已有 host/standalone 发布能力；ssebus 已有 ServeStream/WriteEvent。新增 facade 可以减少破坏性改动，并给外部插件保留迁移路径。

## Decision 3: SSE 分为普通 bus stream 与 upstream stream-through

**Decision**: SSE 支持两类 API：

1. Managed SSE：framework 产生 envelope，用于任务进度、MCP session、日志流。
2. Stream-through SSE：代理上游 SSE，保留原始 event/data，用于 PowerX Agent Runtime。

**Rationale**: Agent SSE 的 token/final/end 是运行时协议，不应被普通 bus envelope 改写。

## Decision 4: 前端同时支持 EventSource 与 fetch SSE

**Decision**: `createPluginSSEClient` 保留 EventSource 模式，同时新增 fetch-based SSE 模式。

**Rationale**: EventSource 自动重连简单，但不能设置 Authorization header；Agent Chat 和部分代理链路需要 header、AbortController 和自定义错误处理。

## Decision 5: topic/channel builder 是强制接口

**Decision**: framework 提供 tenant/member scope builder，并在 CI 和运行时限制业务手写 scope topic/channel。

**Rationale**: tenant/member scope 是隔离边界，不是字符串便利函数。手写会导致跨租户泄露和审计困难。

## Decision 6: Manifest/RBAC 双层校验

**Decision**: CI 校验 `plugin.d/events.yaml`、runtime allowlist 和前端订阅常量；运行时再次校验 publish/subscribe 权限。

**Rationale**: CI 防漂移，运行时防绕过。只做其中一层都不足以满足安全边界。

## Decision 7: skeleton 迁移作为验收样本

**Decision**: MCP SSE、能力注册页 MCP stream、Agent Chat SSE 是本 feature 的必须迁移样本。

**Rationale**: 这三条链路覆盖普通 SSE、前端订阅、上游 stream-through，能证明 framework 不是只存在于包内。
