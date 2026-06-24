# Feature Specification: Framework Realtime Transport

**Feature Branch**: `022-framework-realtime-transport`  
**Created**: 2026-06-11  
**Status**: Draft  
**Input**: 统一 PowerXPlugin Framework 的 WS/SSE 实时通信传输层，覆盖 standalone/host/proxy 模式下的 URL、认证、租户/member 作用域、事件 envelope、连接生命周期、manifest/RBAC 治理与业务侧接入规范。

## Clarifications

### Session 2026-06-11

- Q: 是否放入 `008-framework-task-bus`？ → A: 不放。008 负责后端事件/TaskBus/EventBridge；本 feature 负责实时传输层治理。
- Q: 是否放入 `021-powerx-agent-skill-bridge`？ → A: 不放。021 可作为 Agent SSE 验收场景，但 WS/SSE 统一封装属于 framework 基础设施。
- Q: 是否复用 `015-framework-websocket`？ → A: 是。015 的 WS Bus Adapter 是前置基础，本 feature 扩展为 WS + SSE 统一实时传输。
- Q: Agent SSE 是否直接套 `ssebus.Event` envelope？ → A: 不直接套。Agent Runtime 的 `start/token/final/end/error` 原始语义必须保留；framework 需要提供 stream-through/typed reader 适配。

## User Scenarios & Testing

### User Story 1 - 业务页面不再手写实时连接 (Priority: P1)

作为插件前端开发者，我希望通过 Framework Client 创建 WS/SSE 连接，而不是在页面里手写 `new WebSocket`、`new EventSource` 或 `fetch + getReader`，以便宿主模式、本地模式和代理模式行为一致。

**Why this priority**: 这是统一治理的入口。如果业务代码仍能随意手写连接，URL、token、tenant、cookie、跨域策略会继续漂移。

**Independent Test**: 静态扫描 skeleton 前端，除 framework client 和测试 fixture 外，不存在直接 `new WebSocket`、`new EventSource` 或手写 SSE reader。

**Acceptance Scenarios**:

1. **Given** 插件运行在 standalone 模式，**When** 页面订阅任务进度 SSE，**Then** URL、token、tenant 由 Framework Client 统一生成。
2. **Given** 插件挂载在 PowerX 宿主 `/_p/{plugin_id}` 下，**When** 页面订阅 WS topic，**Then** 客户端连接命中宿主/代理约定路径，不需要页面判断运行模式。
3. **Given** 页面卸载、HMR 或租户切换，**When** Framework Client 发现上下文变化，**Then** 自动关闭旧连接并按策略重连。

---

### User Story 2 - 后端统一发布与订阅治理 (Priority: P1)

作为插件后端开发者，我希望 WS/SSE 都通过 framework runtime 包发布、订阅和输出事件，避免每个 handler 自己设置 headers、heartbeat、flush、topic 拼接和订阅清理。

**Why this priority**: 后端是租户隔离和事件审计边界。散落实现会导致跨租户泄露、连接泄露和事件字段漂移。

**Independent Test**: MCP SSE、runtime 任务进度、Agent SSE proxy 至少三条链路均通过 framework realtime transport 或其 adapter；后端不再直接依赖 `gin-contrib/sse`。

**Acceptance Scenarios**:

1. **Given** 后端向某个 channel 推送进度事件，**When** 客户端断开，**Then** 订阅自动清理，goroutine/channel 不泄漏。
2. **Given** 事件 topic/channel 未在 manifest 声明，**When** 后端尝试发布或订阅，**Then** framework 拒绝并记录结构化日志和指标。
3. **Given** 事件缺少 tenant/member 作用域，**When** 该事件属于租户隔离范围，**Then** framework fail-fast。

---

### User Story 3 - 统一租户/member 作用域和 envelope (Priority: P1)

作为平台治理负责人，我希望 WS topic 与 SSE channel 都遵循同一作用域和 envelope 规范，以便跨模式审计、排障和权限校验。

**Why this priority**: topic/channel 不是 UI 细节，是多租户隔离边界。必须由 framework 统一生成和校验。

**Independent Test**: 使用 topic/channel builder 生成 tenant/member 作用域事件；运行时拒绝手写不合规 topic/channel；事件输出包含统一治理字段。

**Acceptance Scenarios**:

1. **Given** 业务需要订阅 member 级事件，**When** 调用 builder，**Then** 生成包含 tenant_uuid/member_uuid 的标准 topic/channel。
2. **Given** WS/SSE 事件被前端消费，**When** 查看事件 envelope，**Then** 均包含 `protocol`、`topic/channel`、`event_type`、`tenant_uuid`、`member_uuid`、`trace_id`、`timestamp`、`payload`。
3. **Given** Agent Runtime SSE 返回原始 token/final 事件，**When** framework 代理该流，**Then** 保留原始事件名和 data，同时通过 meta 事件或 side-channel 暴露 trace/context。

---

### User Story 4 - manifest/RBAC/事件注册闭环 (Priority: P2)

作为平台管理员，我希望 WS topic / SSE channel 能与 `plugin.d/events.yaml`、capability manifest 和 RBAC 形成一致关系，明确谁能发、谁能订阅、有哪些事件。

**Why this priority**: 没有注册闭环，实时通信会变成隐式接口，无法审计、授权和发布前校验。

**Independent Test**: 新增一个未声明 topic/channel 并尝试发布，CI 与运行时均能发现；合法声明可通过本地和宿主模式验证。

**Acceptance Scenarios**:

1. **Given** `plugin.d/events.yaml` 声明 topic/channel 和 actions，**When** CI 运行 manifest 对齐检查，**Then** runtime 白名单和前端订阅常量必须一致。
2. **Given** 某用户没有 subscribe 权限，**When** 连接并订阅该 topic/channel，**Then** 后端拒绝订阅并返回标准错误事件。
3. **Given** 某后端 handler 没有 publish 权限，**When** 尝试发布事件，**Then** publish 失败并写入审计。

---

### User Story 5 - Agent SSE 通过 framework stream-through 封装 (Priority: P2)

作为插件开发者，我希望 Agent Chat 调试页使用 framework 提供的 Agent SSE stream-through/typed reader，而不是业务页自行解析 PowerX Agent SSE 协议。

**Why this priority**: Agent SSE 事件语义跨插件一致，重复手写会造成 token/final/error 解析和错误映射漂移。

**Independent Test**: Agent Chat 发送消息时，前端调用 framework realtime client；后端 proxy 使用 framework stream-through helper；PowerX Agent Runtime 的原始 SSE 事件能实时显示。

**Acceptance Scenarios**:

1. **Given** PowerX Agent SSE 返回 `start/token/final/end/error`，**When** 插件代理该流，**Then** 前端实时收到同名事件，不被强制包成普通 `ssebus.Event`。
2. **Given** PowerX Agent SSE 返回 401/403/404/5xx，**When** framework 处理响应，**Then** 返回稳定错误码并保留 `trace_id/request_id`。
3. **Given** 前端需要 Authorization header，**When** 使用 framework fetch-based SSE client，**Then** 支持 header 模式而不是只能把 token 放 query。

## Edge Cases

- EventSource 不能设置 Authorization header。
- 宿主代理路径 `/_p/{plugin_id}/api/v1` 与 standalone `/api/v1` 的 base URL 选择错误。
- PowerX Core base URL 已包含 `/api/v1` 时被重复拼接。
- 页面 HMR 后重复创建 WS/SSE 连接。
- tenant_uuid/member_uuid 切换后旧连接仍然接收旧租户事件。
- SSE heartbeat 与业务事件同名冲突。
- Agent SSE 是上游透传流，不能被普通 pub/sub envelope 破坏。
- 未声明 topic/channel 在本地可用但宿主模式被拒绝。
- 后端订阅方慢消费导致 channel 堵塞。
- token 过期后重连失败没有可观测错误。

## Requirements

### Functional Requirements

- **FR-001**: Framework MUST 提供统一 Realtime Client，覆盖 WS bus、普通 SSE stream 和 fetch-based SSE stream-through。
- **FR-002**: Framework Client MUST 支持 standalone、host embedded、`/_p/{plugin_id}` proxy 三种 URL 解析模式。
- **FR-003**: Framework Client MUST 统一处理 token、tenant_uuid、member_uuid、cookie/credentials 和 query/header 鉴权策略。
- **FR-004**: Framework Client MUST 提供连接生命周期管理：connect/disconnect/reconnect、页面卸载清理、HMR 清理、上下文变化重连、重复连接控制。
- **FR-005**: Framework MUST 提供 topic/channel builder，业务代码不得手写 tenant/member 作用域 topic/channel。
- **FR-006**: Framework MUST 定义统一 Realtime Envelope，覆盖 `protocol`、`topic`、`channel`、`event_type`、`payload`、`tenant_uuid`、`member_uuid`、`trace_id`、`timestamp`。
- **FR-007**: Framework MUST 明确 Agent SSE stream-through 例外：保留 PowerX Agent 原始 event/data，并额外暴露 trace/context，不强制改写为普通 envelope。
- **FR-008**: 后端 MUST 提供统一 SSE server helper，包含 headers、flush、heartbeat、连接关闭清理和标准错误事件。
- **FR-009**: 后端 MUST 提供统一 WS/SSE publish/subscribe 权限校验，与 `plugin.d/events.yaml` actions 对齐。
- **FR-010**: 后端 MUST 拒绝未声明、未授权或缺少租户作用域的 topic/channel 发布/订阅。
- **FR-011**: Skeleton MUST 迁移 MCP SSE、能力注册页 MCP stream、通用 `useStream.ts` 到 framework realtime transport。
- **FR-012**: Skeleton Agent Chat MUST 迁移到 framework Agent SSE stream-through client/proxy。
- **FR-013**: Skeleton MUST 移除直接依赖 `github.com/gin-contrib/sse`，除 framework 内部和测试 fixture 外禁止使用。
- **FR-014**: CI MUST 扫描并拒绝业务代码中的直接 `new WebSocket`、`new EventSource`、手写 SSE reader、`gin-contrib/sse`。
- **FR-015**: Framework MUST 输出结构化日志和指标，至少覆盖 connect/open/close/reconnect/subscribe/publish/error/drop。
- **FR-016**: Framework MUST 提供开发态诊断信息，能显示最终 URL 来源、连接状态、已订阅 topic/channel、最后事件 trace_id。
- **FR-017**: 文档 MUST 明确 008/015/021 与本 feature 的边界和迁移路径。

### Key Entities

- **RealtimeTransport**: Framework 实时传输抽象，覆盖 WS 与 SSE。
- **RealtimeClient**: 前端统一客户端，负责 URL、鉴权、生命周期和事件分发。
- **RealtimeEnvelope**: WS/SSE 通用治理字段结构。
- **Topic/Channel Descriptor**: `plugin.d/events.yaml` 中声明的可发布/可订阅事件定义。
- **Scope Context**: tenant_uuid、member_uuid、user_uuid、trace_id 等隔离与观测上下文。
- **StreamThrough Session**: 上游 SSE 透传会话，保留原始事件语义并补充治理元数据。
- **Realtime Permission Decision**: publish/subscribe 权限判定结果，包含 allowed/reason/resource/action。

## Assumptions & Dependencies

- `015-framework-websocket` 已提供 WS Bus Adapter 基线。
- `008-framework-task-bus` 提供后端事件来源和 EventBridge/TaskBus 治理，不负责浏览器连接。
- `021-powerx-agent-skill-bridge` 需要消费本 feature 的 Agent SSE stream-through 能力。
- PowerX 宿主提供或将提供标准 WS/SSE 代理路径和 API key/STS/JWT 鉴权能力。

## Success Criteria

- **SC-001**: skeleton 业务代码中直接 `new WebSocket`、`new EventSource`、手写 SSE reader、`gin-contrib/sse` 的违规数量为 0。
- **SC-002**: standalone 与 host/proxy 两种模式下，同一 WS topic/SSE channel 均能在 2 秒内收到事件。
- **SC-003**: 未声明或未授权 topic/channel 的发布/订阅 100% 被拒绝并输出审计记录。
- **SC-004**: tenant/member 上下文切换后，旧连接 100% 关闭，新连接使用新上下文。
- **SC-005**: Agent Chat 能实时显示 PowerX Agent SSE 原始 `token/final/end/error` 事件，解析错误率为 0。
- **SC-006**: MCP SSE、能力注册页 stream、Agent Chat 三条迁移链路均通过 E2E 或集成测试。
