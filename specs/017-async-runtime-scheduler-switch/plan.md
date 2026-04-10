# Implementation Plan: Async Runtime Scheduler 模式切换

**Branch**: `017-async-runtime-scheduler-switch` | **Date**: 2026-03-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/017-async-runtime-scheduler-switch/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

为插件异步运行时补齐 Scheduler 的“启动期模式识别 + 统一执行链路”规划：
1. 启动时识别 `standalone local` / `delegated proxy`；
2. 冲突配置采用严格失败（fail-fast）；
3. 调度触发与手动触发统一走事件主链路；
4. proxy 权限失败走“有上限重试 -> 超限工单 -> 暂停任务 -> 运维恢复”；
5. 提供双模式一致的联调与验收流程。

## Technical Context

**Language/Version**: Go 1.24（backend runtime），TypeScript 5.x（文档与联调脚本）  
**Primary Dependencies**: Gin skeleton、framework eventbridge/taskbus/wsbus 抽象、runtime ops 管理端点、logrus/slog 结构化日志  
**Storage**: 复用现有插件数据库（PostgreSQL/SQLite），本特性不新增业务持久化模型  
**Testing**: `go test ./...`（scheduler/runtime 相关包）、双模式联调脚本、日志与指标回归检查  
**Target Platform**: Linux server（plugin backend, host/standalone）
**Project Type**: backend runtime + docs  
**Performance Goals**: 调度接线后关键链路时延 p95 增量不超过 5%；调度触发进入主链路成功率 100%  
**Constraints**: 冲突配置必须 fail-fast；权限失败必须有限重试（默认 3 次，可配置 1-10）并人工闭环；恢复权限仅运维/管理员；禁止业务层模式分支；topic 命名必须符合 `powerx.<domain>.<subdomain>.<action>.v<version>`  
**Scale/Scope**: 聚焦 async runtime scheduler 模式切换与故障处理，不扩展新业务域

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Phase 0 预检：

1. Host Contract First（PASS）
   - 不新增破坏性宿主合同；仍由插件入口承接，proxy 通过既有网关能力转发。
2. Tenant Isolation & Zero Trust（PASS）
   - 维持 tenant_uuid 语义；权限失败明确可观测并禁止静默放行。
3. Service-Centric Architecture（PASS）
   - Scheduler 只触发，执行走 EventBridge 抽象；业务层禁止模式分支。
4. Observable & Testable Delivery（PASS）
   - 要求 trace/status/结果可追踪，双模式统一验收流程。
5. Event Contracts & TaskBus Readiness（PASS）
   - 继续遵循 topic 契约与 at-least-once 思维，失败路径具备重试与人工兜底。

Phase 1 复检：

1. `research/data-model/contracts/quickstart` 与宪章无冲突（PASS）
2. 未引入 `tenant_id` 数字语义或弱化鉴权边界（PASS）
3. 失败处理与恢复权限边界清晰可审计（PASS）

## Project Structure

### Documentation (this feature)

```text
specs/017-async-runtime-scheduler-switch/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── scheduler-mode-switch.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
skeleton/backend/go-gin/cmd/plugin/
├── main.go
└── taskbus_provider.go

skeleton/backend/go-gin/internal/jobs/integration/
├── scheduler.go
└── scheduler_event_dispatcher.go

skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/
├── scheduler_mode_handler.go
├── scheduler_retry_handler.go
├── event_bridge_debug_handler.go
├── ws_bus_grant.go
└── ws_bus_publish.go

skeleton/backend/go-gin/internal/services/admin/runtime_ops/
├── scheduler_mode_service.go
├── scheduler_retry_service.go
└── scheduler_ticket_service.go

docs/guides/async_runtime/
├── scheduler/README.md
└── websocket/debug_playbook.md

docs/plan/develop/async_runtime/
├── schedule/scheduler-mode-switch-implementation.md
└── log/runtime-log-align-plan.md
```

**Structure Decision**: 采用 backend runtime + docs 双层结构；实现侧最小改动接入启动流程、失败闭环与审计记录，文档侧提供可执行验收流程与统计台账口径。

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 无 | N/A | N/A |
