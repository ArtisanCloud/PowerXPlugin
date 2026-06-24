# Research — PowerX Agent Skill Bridge Framework 对齐

## 决策 1：新开 021 feature

**Decision**: 新建 `021-powerx-agent-skill-bridge`。

**Rationale**: 该机制横跨插件 Skill 暴露、PowerX Agent Client、本地 Chat、STS/delegated 鉴权和事件解析，不适合塞入单一既有 feature。

**Alternatives considered**:

1. 放入 `006-plugin-capability`：拒绝。`006` 关注能力目录与暴露治理，不负责 Agent Session Client。
2. 放入 `009-consume-powerx-capability`：拒绝。`009` 关注插件消费 PowerX corex capability，不负责插件 Capability Handler 标准。
3. 放入 `015-framework-websocket`：拒绝。`015` 关注 WS Bus 事件发布订阅，不负责 Agent Runtime 事件语义。

## 决策 2：双层 Skill 模型

**Decision**: 插件维护源定义态 Skill，PowerX 维护治理态 Skill。

**Rationale**: 插件需要声明领域能力，但租户可见性、审批、版本、权限、Agent 绑定和审计必须由 PowerX 统一治理。

**Implication**:

1. 插件 `GET /api/v1/plugin/skills` 只提供源定义。
2. PowerX 导入后形成 `source=plugin` 的治理态 Skill。
3. 未发布的插件 Skill 不得进入 Agent 候选池。

## 决策 3：Framework 统一 executor 路由

**Decision**: 插件业务只注册 executor handler，HTTP 路由、上下文校验、错误包装由 Framework 统一处理。

**Rationale**: 每个插件手写 executor 会导致上下文和错误语义漂移。

**Implication**:

1. 统一 PowerX Capability Invocation。
2. 按 `skill_id` 分发。
3. 强校验 `tenant_uuid/user_uuid/agent_id/session_id/trace_id`。

## 决策 4：Agent SSE/WS 进入 Framework Client

**Decision**: Framework Client 封装 PowerX Agent HTTP/SSE/WS。

**Rationale**: 插件本地 Chat、调试页、后台任务都可能需要调用 Agent Runtime；重复解析 SSE/WS 会导致行为不一致。

**Implication**:

1. 提供 typed event decoder。
2. 标准事件包含 `intent/plan/node_start/node_end/token/final/end/error`。
3. 插件前端不直接拼接 PowerX Agent 流协议。

## 决策 5：本地 Chat 不拥有长期对话系统

**Decision**: 插件本地 Chat 是 PowerX Agent Session 的客户端，不是独立对话系统。

**Rationale**: 长期并行对话系统会绕过 PowerX 统一会话、权限、租户和审计。

**Implication**:

1. 本地 Chat 请求目标必须是 PowerX Agent API。
2. 本地 Chat 不直连插件业务 API。
3. 会话权威状态在 PowerX。

## 决策 6：Fail-fast 不做兼容 fallback

**Decision**: 缺少关键上下文、delegated 凭证缺失、capability 不匹配、Skill 未注册时全部 fail-fast。

**Rationale**: PowerXPlugin 与 PowerX 的安全边界要求明确失败，不允许匿名、跨租户或业务直连 fallback。

**Implication**:

1. 启动期检查配置。
2. executor 前检查上下文。
3. E2E 检查本地 Chat 不绕过 Agent Runtime。
