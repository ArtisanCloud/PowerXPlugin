# Implementation Plan: FastAPI 对齐 Go Gin（以 Nuxt 联调为第一目标）

**Branch**: `011-fastapi-gin-align` | **Date**: 2026-01-24 | **Spec**: specs/011-fastapi-gin-align/spec.md
**Input**: Feature specification from `/specs/011-fastapi-gin-align/spec.md`

## Summary

在不修改现有 Gin 与 Nuxt 的前提下，补齐 FastAPI 的目录结构、契约与最小联调接口，使其可在宿主反代与独立运行模式下完成核心管理端联调，并保证数据库结构与契约一致性。

## Technical Context

**Language/Version**: Python 3.11
**Primary Dependencies**: FastAPI, SQLAlchemy 2.0, Alembic
**Storage**: PostgreSQL（schema: `powerx_plugin_base`）
**Testing**: pytest
**Target Platform**: Linux server
**Project Type**: backend
**Performance Goals**: P95 < 1s，错误率 < 1%
**Constraints**: 不修改 Gin/Nuxt；宿主反代路径与权限/租户上下文必须一致
**Scale/Scope**: 管理端联调范围（认证、IAM、模板、能力管理）

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Host Contract First：通过（路径前缀与管理端点按 Gin 规范对齐）
- Tenant Isolation & Zero Trust：通过（保持 tenant_uuid 与 RLS 规则一致）
- Service-Centric Architecture：通过（FastAPI 仅补齐结构与契约，不修改业务边界）
- Observable & Testable Delivery：通过（包含 /healthz 与测试策略）
- Minimal Footprint & Versioned Releases：通过（不引入多余依赖）

## Project Structure

### Documentation (this feature)

```text
specs/011-fastapi-gin-align/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
skeleton/
└── backend/
    ├── go-gin/
    └── python-fastapi/
        ├── app/
        │   ├── bootstrap/
        │   ├── config/
        │   ├── contracts/
        │   ├── entity/
        │   │   ├── models/
        │   │   └── repository/
        │   ├── services/
        │   ├── transport/
        │   │   └── http/
        │   ├── router/
        │   ├── middleware/
        │   ├── observability/
        │   ├── manifestx/
        │   ├── server/
        │   ├── shared/
        │   └── main.py
        ├── scripts/
        │   └── dev.sh
        ├── requirements.txt
        └── tests/
            ├── unit/
            └── integration/
```

**Structure Decision**: 仅新增 Python FastAPI 的后端结构，保持 Gin 与 Nuxt 不变。

## Complexity Tracking

无。

## Phase 0: Research

- 输出：`specs/011-fastapi-gin-align/research.md`
- 已完成决策：契约权威来源、API 前缀来源、ORM/迁移策略、数据库与 schema、测试策略、性能目标。

## Phase 1: Design & Contracts

- 输出：
  - `specs/011-fastapi-gin-align/data-model.md`
  - `specs/011-fastapi-gin-align/contracts/openapi.yaml`
  - `specs/011-fastapi-gin-align/quickstart.md`
- 设计重点：
  - 目录结构与 Gin 对齐
  - 管理端 API 合同与响应封装一致
  - 仅实现 P1 联调范围（认证、IAM、模板、能力管理）

## Phase 2: Implementation Planning

- 拆分领域模块（auth/iam/template/capability）
- 按接口优先级制定任务切片
- 安排单测/集成测与联调验证步骤

## Post-Design Constitution Check

- 目录结构、契约与租户隔离策略与宪章一致，未发现违规项。
