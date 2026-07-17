# Implementation Plan: Framework Realtime Transport

**Branch**: `022-framework-realtime-transport` | **Date**: 2026-06-11 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/022-framework-realtime-transport/spec.md`

## Summary

将 PowerXPlugin Framework 的实时通信能力从局部 WS/SSE helper 收敛为统一 Realtime Transport：前端统一 client、后端统一 server/publish/subscribe helper、manifest/RBAC 事件治理、租户/member 作用域 builder、Agent SSE stream-through 适配，并迁移 skeleton 中现有手写 MCP SSE、能力注册页 stream、Agent Chat SSE。

## Technical Context

**Language/Version**: Go 1.24, TypeScript 5.x (Nuxt 4.2)  
**Primary Dependencies**: Gin, Gorm, Nuxt UI, PowerXPlugin Framework runtime, existing wsbus/ssebus  
**Storage**: PostgreSQL/SQLite only for existing audit/config records; this feature does not require new business tables  
**Testing**: go test, Vitest/Playwright where available, static grep/CI scan  
**Target Platform**: Plugin standalone backend + PowerX host/proxy embedded admin  
**Project Type**: framework backend + framework frontend client + skeleton migration  
**Performance Goals**: first event delivery < 2s; heartbeat interval configurable; no connection leaks after navigation/HMR  
**Constraints**: tenant isolation, member scoped events, EventSource header limitation, host/proxy path differences  
**Scale/Scope**: Framework infrastructure with skeleton migration for MCP SSE, capability stream, Agent stream, WS diagnostics

## Constitution Check

- Host Contract First: Realtime URLs and auth must respect host/proxy contracts.
- Tenant Isolation & Zero Trust: topic/channel must be scoped and authorized before publish/subscribe.
- Event Contracts & TaskBus Readiness: events must align with `plugin.d/events.yaml` and existing EventBridge/TaskBus topics.
- Service-Centric Architecture: business handlers should call framework transport, not manually manage sockets.
- Observability: connect/subscribe/publish/error/drop must produce traceable logs/metrics.

## Project Structure

### Documentation

```text
specs/022-framework-realtime-transport/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── realtime-events.schema.yaml
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code Targets

```text
framework/backend/go/runtime/
├── realtime/          # new facade: envelope, scope, validation, stream-through
├── ssebus/            # extend or wrap existing SSE helper
└── wsbus/             # reuse existing WS adapter

framework/frontend/nuxt/framework-client/
├── realtime.ts        # new facade
├── sse.ts             # extend with managed/header-capable fetch SSE
└── ws.ts              # align diagnostics/lifecycle API

skeleton/backend/go-gin/internal/transport/http/
├── mcp/
├── plugin/agent/
└── wsbus/

skeleton/web-admin/nuxt/app/
├── composables/api/useStream.ts
├── pages/capabilities/RegisterForm.vue
└── pages/_p/com.powerx.plugins.base/admin/agent-skill-bridge/index.vue
```

**Structure Decision**: 新增 framework realtime facade，保留 `wsbus`/`ssebus` 作为底层能力；skeleton 只迁移调用，不再复制协议细节。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 同时覆盖 WS 与 SSE | 目标是治理模型统一，不能只修 Agent SSE | 单独修 Agent 或 MCP 会继续保留多套连接策略 |
| Agent SSE stream-through 例外 | Agent Runtime 有原始 event/data 协议 | 强制套普通 envelope 会破坏 token/final 实时语义 |
| CI 禁止绕过 | 防止后续业务重新手写连接 | 仅靠文档无法保证治理落地 |
