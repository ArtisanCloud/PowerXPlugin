# Feature Specification: Framework WS Bus Adapter

**Feature Branch**: `015-framework-websocket`  
**Created**: 2026-02-03  
**Status**: Draft  
**Input**: User description: "根据 docs/plan/develop/ws-bus-adapter.md 补齐 WS Bus 适配能力（统一发布接口、宿主/standalone 切换、topic 白名单与验收）"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Unified Progress Publish (Priority: P1)

插件后端发布业务进度事件时，不需要感知当前运行模式，仍可将事件推送到前端实时显示。

**Why this priority**: 这是业务实时体验的核心，缺失会导致宿主模式下进度无法到达前端。

**Independent Test**: 在宿主与 standalone 两种模式下触发同一发布动作，前端均能在同一连接上收到进度事件。

**Acceptance Scenarios**:

1. **Given** 插件处于宿主模式且前端已连接消息总线，**When** 后端发布进度事件，**Then** 前端在目标主题上收到事件。
2. **Given** 插件处于 standalone 模式且前端连接插件 WS，**When** 后端发布进度事件，**Then** 前端在目标主题上收到事件。

---

### User Story 2 - Secure Tenant-Scoped Publishing (Priority: P2)

只有经过授权的宿主/插件内部调用才能发布事件，并且事件必须绑定到正确的租户范围。

**Why this priority**: 避免跨租户事件泄露，保障宿主模式的安全与合规。

**Independent Test**: 使用未授权调用或缺失租户上下文的请求无法发布事件。

**Acceptance Scenarios**:

1. **Given** 缺少有效授权或租户上下文，**When** 发送发布请求，**Then** 系统拒绝发布并返回失败结果。

---

### User Story 3 - WS不可用时的体验一致性 (Priority: P3)

当消息总线不可用时，前端可回退到轮询，但业务逻辑无需变更。

**Why this priority**: 提升可用性，避免 WS 抖动导致功能不可用。

**Independent Test**: 模拟 WS 不可用时，前端仍可通过轮询获取进度更新。

**Acceptance Scenarios**:

1. **Given** WS 连接失败或中断，**When** 发生进度更新，**Then** 前端能通过轮询获取到最新状态。

---

### Edge Cases

- 当发布的主题不在白名单时如何处理？
- 当使用旧 topic（org_sync.progress）时是否需要映射到新规范 topic？
- 当租户信息缺失或不匹配时如何处理？
- 当消息总线暂时不可用时是否需要返回明确错误？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 提供统一的发布入口，使业务层不需要感知运行模式差异。
- **FR-002**: 系统 MUST 在宿主模式将事件发布到宿主消息总线，并在 standalone 模式发布到本地消息总线。
- **FR-003**: 系统 MUST 对发布主题进行白名单校验，非白名单主题不得发布。
- **FR-004**: 系统 MUST 强制校验发布请求的授权与租户范围。
- **FR-005**: 系统 MUST 保持统一的事件消息结构（含 topic、type、payload 与元信息），便于前端按主题消费。
- **FR-006**: 系统 MUST 在发布失败时返回可识别的失败结果。
- **FR-007**: 系统 SHOULD 支持可选的追踪信息用于链路排查。

### Key Entities *(include if feature involves data)*

- **Publish Request**: 发布请求，包含主题、事件载荷、租户范围与可选追踪信息。
- **Topic**: 事件主题，用于订阅与路由。
- **Event**: 统一结构的消息载荷。
- **Tenant Context**: 事件所属租户范围信息。

## Assumptions & Dependencies

- 宿主侧具备可调用的发布入口，并能将事件投递到其消息总线。
- 插件前端已有统一的订阅接口，并可在 WS 不可用时回退轮询。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 宿主与 standalone 两种模式下，发布的进度事件均能在 2 秒内被前端收到。
- **SC-002**: 未授权或非白名单主题的发布请求 100% 被拒绝。
- **SC-003**: WS 不可用时，前端轮询可在 10 秒内获取到最新进度。
- **SC-004**: 业务代码不需要区分宿主/standalone 模式即可完成发布。
