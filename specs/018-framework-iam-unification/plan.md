# Implementation Plan: Framework IAM 统一封装（Standalone/Delegated）

**Branch**: `018-framework-iam-unification` | **Date**: 2026-04-11 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/018-framework-iam-unification/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

本特性将 IAM 契约从 skeleton 上提到 framework，目标是让插件业务层只依赖 framework IAM 接口即可在 `standalone(local)` 与 `delegated` 模式间切换。  
关键策略已在 clarify 阶段冻结：
1. 模式优先级：`config.context.iam_mode` > 环境变量；冲突 fail-fast。
2. adapter 切换：启动期单选绑定，运行期不自动切换。
3. delegated 写边界：组织/成员/角色/权限写操作只允许宿主侧执行，插件侧只读。
4. local 最小集：租户、部门、成员、角色、权限五类实体完整可用。

## Technical Context

**Language/Version**: Go 1.24  
**Primary Dependencies**: framework middleware/context/rbac；skeleton IAM local store；delegated auth proxy  
**Storage**: PostgreSQL/SQLite（local 模式复用现有 IAM 表）；delegated 模式不新增插件侧组织写入  
**Testing**: `go test ./...`、contract tests（adapter 一致性）、integration tests（mode switch + authz semantics）  
**Target Platform**: Linux plugin runtime（生产）+ macOS/Linux 开发环境  
**Project Type**: backend framework + skeleton adapter migration  
**Performance Goals**: IAM 上下文解析与权限判定在既有接口下 p95 不劣化超过 10%；模式切换错误首跳发现率 100%  
**Constraints**: 必须满足宪章中的 Host Contract First、Tenant UUID、Zero Trust；delegated 下禁止插件侧组织写入；运行期禁止 adapter 自动切换  
**Scale/Scope**: 首批覆盖 2+ 插件迁移；统一 tenant/department/member/role/permission 与 token/context 读取链路

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Gate

1. Host Contract First: PASS  
delegated 模式下写操作全部走宿主接口，不绕过宿主合同。
2. Tenant Isolation & Zero Trust: PASS  
统一 Tenant UUID 上下文与鉴权判定，不引入 `tenant_id` 数字语义。
3. Service-Centric Architecture: PASS  
业务层依赖 framework IAM 契约，skeleton 只做 adapter 实现。
4. Observable & Testable Delivery: PASS  
模式、权限、上下文解析均需可观测并配套 contract/integration 测试。
5. Event Contracts & TaskBus Readiness: PASS（N/A）  
本特性不新增事件协议，保持现有约束不被破坏。
6. Minimal Footprint & Versioned Releases: PASS  
不新增项目，仅重构契约边界并保留迁移路径。

### Post-Design Gate

Phase 1 产物（research/data-model/contracts/quickstart）完成后复查：PASS。无新增违例。

## Project Structure

### Documentation (this feature)

```text
specs/018-framework-iam-unification/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── iam-unification.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
framework/backend/go/
├── iam/
│   ├── contracts/
│   ├── adapters/
│   ├── context/
│   └── errors/
└── middleware/

skeleton/backend/go-gin/internal/
├── services/iam/
│   ├── local_store.go
│   ├── directory.go
│   └── adapters/
│       ├── local/
│       └── delegated/
├── services/authproxy/
└── transport/http/admin/iam/
```

**Structure Decision**: 采用“framework 契约 + skeleton 适配”结构，不新增独立服务；通过最小目录扩展承载统一接口与迁移兼容层。

## Phase 0: Research Output

已生成 [research.md](./research.md)，覆盖：
1. 模式优先级与 fail-fast 策略；
2. adapter 单选绑定策略；
3. delegated 只读/宿主写边界；
4. local 最小实体集与迁移兼容建议。

## Phase 1: Design Output

已生成：
1. [data-model.md](./data-model.md)（核心实体、关系、约束、状态）
2. [contracts/iam-unification.openapi.yaml](./contracts/iam-unification.openapi.yaml)（统一 API 契约骨架）
3. [quickstart.md](./quickstart.md)（双模式接入与回归步骤）

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 无 | N/A | N/A |
