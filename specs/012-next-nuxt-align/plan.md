# Implementation Plan: Next 管理端与 Nuxt 对齐

**Branch**: `012-next-nuxt-align` | **Date**: 2026-03-14 | **Spec**: /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/012-next-nuxt-align/spec.md
**Input**: Feature specification from `/specs/012-next-nuxt-align/spec.md`

## Summary

以 Nuxt 现有行为为裁决基线，在不新增 Next 私有后端接口的前提下，将 `skeleton/web-admin/next` 分两里程碑迁移：里程碑一覆盖登录/首页/模板主链路，里程碑二覆盖 IAM、Capabilities、Integration、Operations、Security 全域页面，并在宿主/独立两种模式下完成一致性回归；联调差异需在 2 个工作日内完成归因，Gin 仅允许缺陷型最小修复。

## Technical Context

**Language/Version**: TypeScript 5.x, React 18, Next.js 14.2.5, Go 1.24（联调基线）  
**Primary Dependencies**: Next App Router, Playwright（E2E）, 既有 Go-Gin 管理端 API 契约  
**Storage**: 前端本地会话存储（token/expires）+ 后端既有数据库（由 Gin 管理）  
**Testing**: Playwright E2E, Next lint/build, 回归清单验证  
**Target Platform**: Web Admin（宿主反代 + 独立访问）  
**Project Type**: web（frontend + existing backend contract）  
**Performance Goals**: 关键管理流程页面交互保持可用，双模式主链路与关键异常场景通过率 100%  
**Constraints**: 不改 Nuxt 基线行为；不为 Next 增加私有接口；Gin 仅在确认缺陷后最小修复；差异 2 个工作日内归因  
**Scale/Scope**: 里程碑一覆盖 Auth/Intro/Templates；里程碑二覆盖 IAM/Capabilities/Integration/Operations/Security

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Host Contract First：通过（沿用既有 `/v1/**` 与宿主反代规范，不新增 Next 私有接口）。
- Tenant Isolation & Zero Trust：通过（仅消费既有鉴权与租户语义，前端不引入越权捷径）。
- Service-Centric Architecture：通过（业务逻辑仍以后端 Service 为中心，前端仅做对齐与编排）。
- Observable & Testable Delivery：通过（要求 E2E 回归、关键异常场景验证、差异归因留痕）。
- Event Contracts & TaskBus Readiness：通过（本特性不变更事件契约，保持兼容）。
- Minimal Footprint & Versioned Releases：通过（不引入额外后端子系统，以迁移与对齐为主）。
- Frontend Stack Exception（需审批）：本特性属于 Nuxt→Next 迁移专项例外，实施前需补齐治理审批记录。
- Frontend Artifact Gate：Next 构建产物需对齐插件发布链路（`web-admin/.output/` 或等价映射），发布前必须完成路径一致性校验。

### Exception Approval Record

- Exception ID：PX-FE-EXCEPTION-012
- Scope：Nuxt -> Next 迁移期间的前端技术栈例外
- Status：Approved
- Approver：Feature Request Owner（当前会话确认）
- Approval Date：2026-03-14
- Evidence Link：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/012-next-nuxt-align/exception-approval.md`
- Gate Rule：审批记录已闭环，可进入 `/speckit.implement`。

## Project Structure

### Documentation (this feature)

```text
specs/012-next-nuxt-align/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
skeleton/
├── web-admin/
│   ├── nuxt/
│   │   ├── app/
│   │   ├── i18n/
│   │   └── tests/e2e/
│   └── next/
│       ├── app/
│       ├── components/
│       ├── hooks/
│       ├── lib/
│       └── tests/
└── backend/
    └── go-gin/
        ├── cmd/plugin/
        ├── internal/transport/http/admin/
        └── internal/services/
```

**Structure Decision**: 采用现有前后端双目录结构，迁移工作集中在 `skeleton/web-admin/next`；`skeleton/backend/go-gin` 仅作为联调契约与缺陷修复基线。

## Complexity Tracking

无宪章违规项，无额外复杂度豁免。

## Phase 0: Research

- 输出：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/012-next-nuxt-align/research.md`
- 目标：确认运行模式、鉴权语义、双栈回归策略、Gin 缺陷准入门槛与归因时限。
- 结果要求：清除技术上下文中的不确定项，形成可执行决策。

## Phase 1: Design & Contracts

- 输出：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/012-next-nuxt-align/data-model.md`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/012-next-nuxt-align/contracts/openapi.yaml`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/012-next-nuxt-align/quickstart.md`
- 设计重点：
  - 页面与路由映射（Nuxt -> Next）可验证
  - 会话、权限、错误语义一致性
  - 双模式（宿主/独立）一致性验收
  - API 合同仅引用既有 Gin 管理端契约
  - 构建产物与发布路径对齐（`.output` 交付约束）及验收方法

## Phase 2: Implementation Planning

- 按页面域与共享底座切分任务：Auth/Session、Templates、IAM、Capabilities、Integration/Ops/Security、E2E。
- 为每个任务绑定“Nuxt 对照项 + 验证场景 + 回归门禁”。
- 对联调差异任务增加 SLA：2 个工作日完成归因并记录结论。

## Post-Design Constitution Check

- 设计产物未引入新后端私有接口，满足 Host Contract First。
- 数据与租户边界沿用 Gin 既有契约，满足 Tenant Isolation。
- 任务切分保持“前端迁移优先、Gin 缺陷最小修复”，满足 Service-Centric 与 Minimal Footprint。
- 回归与验收路径可执行，满足 Observable & Testable Delivery。
- 已将构建产物对齐与发布校验纳入任务，满足 Frontend Artifact Gate。
