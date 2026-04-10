# Implementation Plan: Runtime 日志统一对齐

**Branch**: `016-runtime-log-unification` | **Date**: 2026-03-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/016-runtime-log-unification/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

对齐 Core `async_runtime/log_trace` 的最小日志字段语义，统一 framework 与 skeleton 的 runtime 日志入口，覆盖关键链路（Task enqueue/consume/ack/fail + WS publish/dispatch），并采用 `tenant_uuid` 主字段 + `tenant_key` 镜像字段策略（保留 7 天回滚窗口与迁移文档）。

本次实施范围在 Phase 1 先固化三项前置交付：
1. 计划范围与约束统一（plan）
2. 字段契约与状态枚举定稿（contract）
3. 快速验收命令与样本规则落地（quickstart）

## Technical Context

**Language/Version**: Go 1.24（backend runtime），TypeScript 5.x（文档/工具链侧验证）  
**Primary Dependencies**: framework runtime（taskbus/wsbus/common middleware）、Gin skeleton、`log/slog`、`logrus`  
**Storage**: N/A（本特性不新增持久化模型）  
**Testing**: `go test ./...`（重点 runtime 相关包）、日志字段检索回归（rg）  
**Target Platform**: Linux server（plugin backend）  
**Project Type**: backend runtime + docs  
**Performance Goals**: 日志补点不引入可感知链路延迟退化（关键链路事件处理时延增量 < 5%）  
**Constraints**: 关键链路字段完整率 100%；`status` 枚举固定；`tenant_uuid` 为主字段且 `tenant_key` 为镜像字段；SC-003/SC-004 需按 14 天窗口统计并可追溯  
**Scale/Scope**: 仅覆盖 Task 4 类 + WS 2 类关键链路；不做全量历史模块改造

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

预检（Phase 0 前）：

1. Host Contract First：不新增破坏性对外接口，遵循现有 runtime 边界（PASS）
2. Tenant Isolation & Zero Trust：统一租户语义为 `tenant_uuid`，并输出 `tenant_key` 镜像用于跨系统对齐（PASS）
3. Service-Centric Architecture：统一日志入口在 runtime 抽象层，不把业务逻辑下沉到 handler（PASS）
4. Observable & Testable Delivery：明确结构化字段、状态枚举、关键链路验收样本（PASS）
5. Event Contracts & TaskBus Readiness：保留 `trace_id`/`topic`/租户字段与 task/ws 链路一致性（PASS）

复检（Phase 1 后）：

1. 设计产物（research/data-model/contracts/quickstart）无宪章冲突（PASS）
2. 未引入 `tenant_id` 数字语义或弱化鉴权语义（PASS）
3. 测试与回滚窗口要求已落地到验收步骤（PASS）

## Project Structure

### Documentation (this feature)

```text
specs/016-runtime-log-unification/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── runtime-log.schema.yaml
└── tasks.md
```

### Source Code (repository root)

```text
framework/backend/go/runtime/
├── taskbus/
├── wsbus/
└── common/

skeleton/backend/go-gin/internal/
├── logger/
├── shared/app/
└── transport/http/admin/runtime_ops/

docs/guides/async_runtime/
└── observability/
```

**Structure Decision**: 采用 backend runtime + docs 结构；framework 与 skeleton 同步收敛日志语义，文档侧同步验收口径。

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 无 | N/A | N/A |

## Final Delivery（T035）

### 交付清单

1. 统一日志能力：`framework/backend/go/runtime/common/logging/*`
2. 关键链路接入：taskbus/wsbus/skeleton runtime logger 与 gateway auth 观测链路
3. 规格文档：`specs/016-runtime-log-unification/*`
4. 插件文档对齐：`docs/guides/async_runtime/*` 与 `docs/plan/develop/log/*`

### 实施顺序（最终）

1. Phase 1（Setup）完成契约与验收脚本
2. Phase 2（Foundational）完成统一日志 facade 与 fallback 策略
3. Phase 3（US1）完成关键链路字段基线接入
4. Phase 4（US2）完成 Host/Standalone 一致性与扩展字段保留
5. Phase 5（US3）完成文档与实现口径对齐
6. Phase 6（Polish）完成回归、回滚演练、统计台账与性能对比

### 发布前 Gate

1. 关键链路字段完整率达到 100%
2. `status` 仅出现标准枚举
3. 缺失上下文规则可检索
4. 文档检索命令与实现输出一致
