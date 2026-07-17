# Implementation Plan: PowerX Agent Skill Bridge Framework 对齐

**Branch**: `021-powerx-agent-skill-bridge` | **Date**: 2026-06-07 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/021-powerx-agent-skill-bridge/spec.md`

## Summary

为 PowerXPlugin Framework 增加 Agent Skill Bridge 插件侧标准：插件可声明 Skill 源定义，Framework 暴露统一 Skill 发现与 executor 调用接口；Framework Client 封装 PowerX Agent HTTP/SSE/WS；插件自有 Chat 通过 PowerX Agent Runtime 调试智能任务，而不是直连插件业务 API。  
新增 Agent/Skill Plugin Registry + Sync 标准：PowerXPlugin 可在插件自有维护 Agent/Skill 开发态记录，用于管理页、调试入口和同步状态排障；插件 backend 必须把插件 Registry 同步到 PowerX 底座，生成 PowerX 治理态 Skill、运行态 Agent 和 Agent-Skill Binding。Agent Runtime 的权威数据始终在 PowerX。
该 feature 对齐 PowerX 底座 `024-ai-engineering-skills`，并依赖既有插件鉴权、Capability、Gateway Client、WS、IAM 机制。

## Technical Context

**Language/Version**: Go 1.24（framework/backend）, TypeScript/Nuxt/Next（skeleton/web-admin 验证）  
**Primary Dependencies**: framework runtime, Gin/FastAPI adapters, STS/Gateway Client, WS/SSE client, unified logger  
**Storage**: 新增插件 Agent/Skill Plugin Registry 记录与同步状态；Invocation Trace 首版可走 framework 日志/内存记录，后续按插件需要接入数据库  
**Testing**: Go unit tests, framework integration tests, skeleton E2E, contract smoke tests  
**Target Platform**: PowerX hosted plugin runtime + standalone skeleton runtime  
**Project Type**: framework reusable capability + skeleton integration + docs/spec  
**Performance Goals**: Skill manifest 注册 1 秒内完成；SSE 首事件 p95 < 2 秒；executor context 校验开销可忽略  
**Constraints**: Host Contract First、Tenant UUID 必填、Zero Trust、delegated bearer only、禁止业务直连替代 Agent Runtime  
**Scale/Scope**: 覆盖所有基于 PowerXPlugin framework 的 Go 插件；首版提供最小前端 Chat Client 和 skeleton 示例

## Constitution Check

1. **Host Contract First (PASS)**: 插件自有 Chat 与渠道调试统一经 PowerX Agent Session，不绕过宿主 Agent Runtime。
2. **Tenant Isolation & Zero Trust (PASS)**: executor 强制校验 tenant/user/session/trace 上下文。
3. **Service-Centric Architecture (PASS)**: Skill Runtime 与 Agent Client 上提 framework，业务插件仅实现 executor。
4. **Observable & Testable Delivery (PASS)**: trace/log/error 模型进入任务与 quickstart，提供 fail-fast 和 E2E 验收。
5. **Minimal Footprint & Versioned Releases (PASS)**: 首版只新增 Agent/Skill Plugin Registry 与同步状态必需持久化，不实现独立长期 Agent Runtime。

## Project Structure

### Documentation

```text
specs/021-powerx-agent-skill-bridge/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── tasks.md

docs/plan/021-powerx-agent-skill-bridge.md
docs/guides/develop/agent-skill-bridge/
```

### Source Code

```text
framework/backend/go/runtime/skills/
├── manifest.go
├── registry.go
├── invocation.go
├── executor.go
├── result.go
├── errors.go
└── http_routes.go

framework/backend/go/runtime/powerx/
├── sts/
├── capability/
└── agent/
    ├── client.go
    ├── session.go
    ├── sse.go
    ├── websocket.go
    └── events.go

skeleton/backend/go-gin/
├── internal/skills/
├── internal/agent_registry/
├── internal/agent_registry/
├── internal/sync/
└── internal/transport/http/plugin/skills/

skeleton/web-admin/
└── (local agent chat demo)
```

**Structure Decision**: 采用 framework 可复用核心 + skeleton 示例接线。不同语言适配（FastAPI、Next/Nuxt）后续复用相同合同。

## Phase 0: Research

1. 对齐 PowerX `024-ai-engineering-skills` 的 Agent Skill Bridge 术语与上下文字段。
2. 明确 `006-plugin-capability`、`009-consume-powerx-capability`、`015-framework-websocket` 的边界，避免职责重叠。
3. 确定 Skill Runtime 首版接口：发现、schema、invoke、invocation 查询。
4. 确定 Agent Client 首版接口：invoke、SSE、WS、事件解析、错误映射。
5. 确定 fail-fast 策略：上下文缺失、capability 不匹配、delegated 凭证缺失、Skill 未注册。
6. 确定 Plugin Registry + Sync 策略：插件自有记录作为开发态声明源，PowerX 底座记录作为运行态权威源。

## Phase 1: Design

1. 数据模型：`PluginSkillManifest`、`PluginSkillInvocation`、`PluginSkillInvocationContext`、`PluginSkillResult`、`PowerXAgentClientConfig`、`AgentStreamEvent`、`PluginSkillDefinition`、`PluginAgentDefinition`、`PluginRegistrySyncRequest/Result`。
2. HTTP 合同：`GET /api/v1/plugin/skills`、`GET /api/v1/plugin/skills/:skill_id/schema`、PowerX Capability Invocation。
3. Client 合同：`CreateSession`、`Invoke`、`StreamSSE`、`ConnectWS`、`DecodeEvent`。
4. 错误模型：统一 `skill.*` 错误码，保留 `trace_id/request_id`。
5. Quickstart：模板 Skill 示例、本地 Chat 验收、fail-fast 验收、插件 Registry 同步验收。

## Phase 2: Framework Skill Runtime

1. 实现 manifest 类型与校验器。
2. 实现 registry 与 duplicate 检查。
3. 实现 executor 注册与分发。
4. 实现 HTTP 路由适配器。
5. 实现上下文强校验与错误映射。

## Phase 3: Framework PowerX Agent Client

1. 实现 Agent invoke client。
2. 实现 SSE client 与 typed event decoder。
3. 实现 WS client 与重连策略。
4. 接入 STS/Bearer delegated 认证。
5. 对齐日志字段和 trace 传播。

## Phase 4: Skeleton 示例与本地 Chat

1. 新增最小示例 Skill。
2. 新增本地 Chat 页面或组件。
3. 页面通过 Framework Client 调用 PowerX Agent Stream。
4. 禁止页面直连插件业务 API 作为智能任务路径。
5. 提供模板 Skill 示例 `powerxplugin.template.basic`。
6. Agent Chat 下拉框只展示已同步成功的 PowerX Agent。

## Phase 4A: Skeleton Agent/Skill Plugin Registry 管理与同步

1. 新增插件 Skill Plugin Definition CRUD：保存 `manifest/prompt/schema/executor/capability/checksum/sync_status`。
2. 新增插件 Agent Plugin Definition CRUD：保存 `prompt/model_profile_ref/plugin_skill_ids/powerx_skill_ids/sync_status`。
3. 新增 Skill Sync proxy：插件 backend 调 PowerX Skill Registry/Import/Publish API，回写 `powerx_skill_id/sync_status/sync_error/last_sync_at`。
4. 新增 Agent Sync proxy：插件 backend 调 PowerX Agent Admin API，传入 `skillIds` 建立绑定，回写 `powerx_agent_uuid/sync_status/sync_error/last_sync_at`。
5. 新增 Refresh 状态：从 PowerX 拉取 Agent/Skill 当前状态，发现 disabled/drifted 后更新本地 Plugin Registry。
6. 新增 PowerX 底座能力菜单：`Agent 管理`、`Skill 管理`、`Agent Chat 调试`，UI 对齐 PowerX 底座但展示插件自有同步状态。
7. 链路约束测试：前端所有管理、同步、会话和 SSE 请求必须先进入插件 backend，不得直接请求 PowerX 底座。

## Phase 5: Verification & Rollout

1. Go unit tests：manifest/registry/context/errors。
2. Integration tests：discover/schema/invoke/fail-fast。
3. Client tests：SSE/WS event decode。
4. E2E：插件 Skill/Agent Plugin Definition -> Sync PowerX -> 本地 Chat -> PowerX Agent -> 插件 Capability Handler。
5. 文档发布：quickstart、开发指南、迁移说明。

## Dependencies

1. `005-plugin-auth`
2. `006-plugin-capability`
3. `009-consume-powerx-capability`
4. `015-framework-websocket`
5. `018-framework-iam-unification`
6. PowerX `024-ai-engineering-skills`
