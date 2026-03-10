# Implementation Plan: Framework WS Bus Adapter

**Branch**: `015-framework-websocket` | **Date**: 2026-02-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/015-framework-websocket/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

实现插件侧统一 WS Bus 发布入口，通过 framework SDK 屏蔽宿主/standalone 模式差异；宿主模式转发至底座发布接口，standalone 直接发布本地 WS bus；加入 topic 白名单与租户校验，保证前端实时订阅一致性。

## Technical Context

**Language/Version**: Go 1.24, TypeScript 5.x (Nuxt 4.2)  
**Primary Dependencies**: Gin/Gorm (backend), PowerXPlugin Framework, Nuxt UI  
**Storage**: PostgreSQL/SQLite (schema: powerx_plugin_base)  
**Testing**: go test, npm test  
**Target Platform**: Linux server + web-admin  
**Project Type**: backend + frontend (plugin framework + web-admin)  
**Performance Goals**: WS 发布到前端 < 2s  
**Constraints**: tenant isolation, STS 短期凭证, topic 白名单  
**Scale/Scope**: 插件级功能，面向单插件多租户

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Host Contract First: 必须走宿主 /api/ws 与宿主发布入口（OK）
- Tenant Isolation & Zero Trust: 发布需租户校验与授权（OK）
- Event Contracts & TaskBus Readiness: topic 命名统一采用 `_topic.*` 规范，并与插件 manifest 白名单保持一致

## Project Structure

### Documentation (this feature)

```text
specs/015-framework-websocket/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
framework/
  ... (framework SDK + adapters)
backend/
  ... (plugin backend services)
web-admin/
  ... (frontend WS client already exists)
```

**Structure Decision**: 本功能主要改 framework/ 与 backend/，前端仅复用既有订阅接口。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Topic 命名统一（`_topic.*`） | 与 async_runtime 命名约束一致，减少多套语义并行 | 保留历史 topic 会增加联调与权限治理成本 |
