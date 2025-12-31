# Implementation Plan: Framework TaskBus Event Bridge

**Branch**: `008-framework-task-bus` | **Date**: 2025-12-30 | **Spec**: `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/spec.md`  
**Input**: Feature specification from `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/spec.md`

## Summary

为插件提供统一的事件机制（Emitter/Consumer 抽象 + 事件契约），并通过配置开关接入 PowerXPlugin Framework 的 TaskBus，支持插件与宿主之间共享事件流、指标与后台任务调度能力；迁移过程支持双写灰度与快速回滚。

## Technical Context

**Language/Version**: Go 1.24  
**Primary Dependencies**: Gin/Gorm（插件侧既有）；framework runtime 组件（用于 TaskBus 适配，细节见 research）  
**Storage**: N/A（本 feature 目标是事件机制与迁移路径；consumer 落地结果时复用既有 DB）  
**Testing**: `go test`（unit/integration）；可选 in-memory bus 集成测试  
**Target Platform**: Linux（GitHub Actions runner / 宿主部署环境）  
**Project Type**: Backend + framework libraries  
**Performance Goals**: 事件发布/消费 p95 < 200ms（本地 in-process）；TaskBus 模式按宿主队列与网络能力度量  
**Constraints**:
- 多租户：事件 meta 强制 `tenant_uuid`，并在消费侧保持租户上下文一致
- 最小权限：publish/subscribe 精确到 topic 前缀 + 版本号，尽量不用 `*`
- 可靠性：at-least-once + consumer 幂等（默认去重 key：`topic + tenant_uuid + trace_id`）
- 降级：TaskBus 不可用时自动降级到本地实现并告警/打点（主流程不 panic）
**Scale/Scope**: 先覆盖 Channel 域事件（credential_inspection/kpi_refreshed/publish_task），并提供可扩展的通用约定

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- ✅ Tenant UUID 全链路：事件 meta 强制 `tenant_uuid`，不引入/不使用 `tenant_id`
- ✅ 最小权限：Topic 权限声明按最小权限原则精确到版本号
- ✅ Service-Centric：业务编排在 service/job；handler 薄；Emitter/Consumer 抽象不侵入 handler
- ✅ 可观测：发布/消费成功率、失败率与延迟必须可观测
- ✅ Minimal Footprint：不引入外部 MQ 作为硬前置，先以本地实现 + framework 适配器推进

## Phase 0 — Research

输出：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/research.md`

已覆盖的关键决策：

- TaskBus 不可用时自动降级到本地实现（并告警/打点）
- 投递语义采用 at-least-once，consumer 必须幂等（默认幂等 key：`topic + tenant_uuid + trace_id`）
- 契约变更走 PR 评审 + CI 校验（topic 唯一、必填 meta 字段齐全）
- 权限声明采用最小权限（精确到 topic 前缀 + 版本号）

## Phase 1 — Design & Contracts

### Data Model

输出：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/data-model.md`

### Contracts

输出目录：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/contracts/`

- Channel 事件契约：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/contracts/channel-events.yaml`

### Quickstart

输出：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/quickstart.md`

### Agent Context Update

执行：`.specify/scripts/bash/update-agent-context.sh codex`（本分支已就绪，可执行）

## Project Structure

### Documentation (this feature)

```text
specs/008-framework-task-bus/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── channel-events.yaml
└── tasks.md
```

### Source Code (repository root)

```text
skeleton/backend/
├── internal/observability/channel/
├── internal/jobs/channel/
└── internal/services/admin/

framework/backend/go/
└── runtime/
```

**Structure Decision**: 文档与契约集中在 `specs/008-framework-task-bus/`；代码层面以“插件侧抽象接口 + 可插拔实现 + 灰度迁移”推进，framework TaskBus 以适配器方式接入。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _None_ |  |  |
