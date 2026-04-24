# Implementation Plan: Framework 统一日志适配

**Branch**: `020-framework-logger` | **Date**: 2026-04-23 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/020-framework-logger/spec.md`

## Summary

实现 framework 级统一日志门面与多 sink 路由能力，确保插件在宿主模式默认走 `stdout + json` 并由 PowerX 汇聚；同时支持受控的 `file/loki` 扩展输出。  
本次规划重点覆盖：策略下发与生效、低基数标签统一、业务日期口径统一、目标故障降级重试、遗留直写日志分阶段治理。

## Technical Context

**Language/Version**: Go 1.24（backend runtime/framework）, TypeScript 4.x（web-admin 验证）  
**Primary Dependencies**: framework runtime/common logging, slog/logrus adapter, skeleton logger bridge, observability hooks  
**Storage**: N/A（不新增业务持久化表；仅复用现有配置来源与日志后端）  
**Testing**: Go unit tests, integration tests（multi-sink/fallback）, regression tests（host/standalone）  
**Target Platform**: Linux plugin runtime + PowerX host proxy runtime  
**Project Type**: framework reusable capability + skeleton integration  
**Performance Goals**: 日志链路开启多 sink 时不降低主业务成功率；单 sink 故障不阻塞主链路  
**Constraints**: Host Contract First、Tenant UUID、Zero Trust、固定低基数标签（`plugin_id tenant_uuid component level`）、宿主默认 `stdout+json`、其余 sink 需显式授权  
**Scale/Scope**: 覆盖所有基于 PowerXPlugin framework 的插件日志接入路径；首期完成 framework + skeleton 主链路与治理开关

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Gate

1. Host Contract First: PASS  
宿主模式默认 stdout 交由 PowerX 汇聚，避免插件侧私有日志链路漂移。
2. Tenant Isolation & Zero Trust: PASS  
统一日志字段强制 tenant_uuid 与 trace 上下文，敏感字段禁止标签化。
3. Service-Centric Architecture: PASS  
日志能力上提 framework，插件通过统一门面调用，避免业务层直写实现分裂。
4. Observable & Testable Delivery: PASS  
要求记录 sink 失败告警与重试结果，并提供可测验证路径。
5. Event Contracts & TaskBus Readiness: PASS（N/A）  
本特性不新增事件总线协议，仅改进日志观测链路。
6. Minimal Footprint & Versioned Releases: PASS  
以门面适配替代插件重复实现，降低维护面；治理策略可分阶段启用。

### Post-Design Gate

Phase 1 产物（research/data-model/contracts/quickstart）完成后复查：PASS。无新增违例。

## Project Structure

### Documentation (this feature)

```text
specs/020-framework-logger/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── framework-logger.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
framework/backend/go/
└── runtime/common/logging/

framework/backend/go/
└── runtime/
    ├── wsbus/
    └── taskbus/

skeleton/backend/go-gin/internal/
├── logger/
├── config/
├── bootstrap/
└── shared/app/

skeleton/web-admin/nuxt/
└── (验证与治理提示页面，若需要)
```

**Structure Decision**: 采用“framework 统一能力 + skeleton 接线与兼容桥”结构。插件侧仅通过 framework 门面记录日志，不在业务模块重复实现 sink 选择逻辑。

## Phase 0: Research Output

已生成 [research.md](./research.md)，覆盖：
1. 多 sink 路由与故障隔离策略；
2. 宿主模式默认 stdout 策略与授权扩展边界；
3. 低基数标签与高基数字段治理原则；
4. 业务日期（UTC + biz_date + biz_tz）统一口径；
5. 遗留直写日志分阶段治理路径。

## Phase 1: Design Output

已生成：
1. [data-model.md](./data-model.md)（策略、事件、路由结果、治理状态模型）  
2. [contracts/framework-logger.openapi.yaml](./contracts/framework-logger.openapi.yaml)（日志策略与治理接口契约）  
3. [quickstart.md](./quickstart.md)（接入、验证、回归流程）

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 规格中包含平台治理约束（默认 sink/标签基线/迁移截止） | 这些约束直接决定跨插件一致性与运维成本 | 仅描述“支持日志”会导致实现分裂，无法满足平台统一检索与治理目标 |
