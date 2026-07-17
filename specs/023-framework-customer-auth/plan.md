# Implementation Plan: Framework Customer Identity/Auth

**Branch**: `023-framework-customer-auth` | **Date**: 2026-06-24 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/023-framework-customer-auth/spec.md`

## Summary

将 C 端 Customer Identity/Auth 从 skeleton 内部实现上提为 PowerXPlugin Framework 通用能力。Framework 负责 customer context、token 校验契约、tenant 解析、membership 校验、bootstrap 入口解析、委托身份源契约、稳定错误、观测和测试 helper；skeleton 迁移现有 mini-app customer auth 到 framework adapter 装配层。生产环境以 PowerX Core 或平台身份源为权威，local/mock 仅用于开发、测试或显式 break-glass。

## Technical Context

**Language/Version**: Go 1.24  
**Primary Dependencies**: Gin middleware/context, existing skeleton customer auth, PowerX STS/delegated client patterns, framework runtime common logging/errors  
**Storage**: Framework 不新增生产 customer 持久化；skeleton local/dev 可复用现有 customer 表；生产 membership/customer 数据由 PowerX Core 或平台身份源权威维护  
**Testing**: `go test` unit/contract/integration tests; skeleton mini-app customer auth regression tests; static checks for member/customer boundary  
**Target Platform**: Plugin standalone backend + PowerX host/proxy runtime  
**Project Type**: backend framework + skeleton adapter migration  
**Performance Goals**: customer token/context 注入 p95 不超过现有 skeleton mini-app auth 基线 10%；membership 缓存不得超过 token 有效期；delegate 超时必须 fail-fast  
**Constraints**: 生产默认禁止 local/mock；Customer Auth 与 Member IAM 语义隔离；Tenant UUID only；禁止记录 raw token/password/secret；业务插件不得把行业 customer 模型写入 framework context  
**Scale/Scope**: 首批覆盖 framework Go runtime、skeleton go-gin customer auth 迁移、mini-app protected routes、test helper 与 OpenAPI 契约

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Gate

1. Host Contract First: PASS  
生产 customer 登录、校验、membership、bootstrap 以 PowerX Core/平台身份源为权威；插件通过委托契约接入，不耦合宿主内部实现。

2. Tenant Isolation & Zero Trust: PASS  
tenant-scoped 资源必须解析 Tenant UUID 并校验 customer membership；跨 token/request/bootstrap tenant mismatch 必须拒绝。

3. Service-Centric Architecture: PASS  
framework 提供契约与 middleware/helper，skeleton 只做 adapter 装配；业务插件只消费 context，不重复解析 token。

4. Observable & Testable Delivery: PASS  
要求稳定错误、审计/诊断字段、mock validator/resolver、membership disabled 与 tenant mismatch 测试。

5. Event Contracts & TaskBus Readiness: PASS（N/A）  
本特性不新增事件 topic。若未来加入 membership invalidation event，必须另行按事件契约声明。

6. Minimal Footprint & Versioned Releases: PASS  
不新增独立服务，不在 framework 中持久化生产 customer；以契约上提和 skeleton 迁移为主。

### Post-Design Gate

Phase 1 产物（research/data-model/contracts/quickstart）完成后复查：PASS。无宪章违例；生产 local/mock break-glass 需要显式审计，符合 Zero Trust。

## Project Structure

### Documentation (this feature)

```text
specs/023-framework-customer-auth/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── customer-auth.openapi.yaml
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
framework/backend/go/runtime/customerfw/
├── context.go          # CustomerContext getter/setter
├── errors.go           # stable error codes and mapping
├── middleware.go       # customer auth and membership middleware
├── validator.go        # token validator contract and adapters
├── membership.go       # membership resolver contract
├── bootstrap.go        # entry/bootstrap client contract
├── auth_client.go      # register/login/validate client contract
├── diagnostics.go      # audit/log fields and mode diagnostics
└── testing.go          # test token/context/mock helpers

skeleton/backend/go-gin/internal/
├── domain/customer/                 # compatibility types/adapters during migration
├── middleware/customer/             # wrapper to framework context during transition
├── middleware/customer_auth.go      # migrate to framework middleware wrapper
├── services/customer/               # local/delegate adapters implementing framework contracts
├── observability/customer/          # reuse/align audit and metrics
└── transport/http/mini-app/         # protected routes use framework customer auth

scaffold/templates/backend/go-gin/internal/
└── ...                              # mirror skeleton migration for generated plugins

tools/cli/internal/templates/data/backend/go-gin/internal/
└── ...                              # keep CLI template output aligned
```

**Structure Decision**: 新增 `framework/backend/go/runtime/customerfw` 承载通用契约；skeleton 和模板通过 adapter/wrapper 迁移，避免业务插件继续依赖 skeleton internal customer auth。

## Phase 0: Research Output

已生成 [research.md](./research.md)，覆盖：
1. Customer Auth 独立于 Member IAM；
2. 生产身份权威源为 PowerX Core/平台；
3. 全局 customer token + tenant-scoped membership 校验；
4. membership 短 TTL 缓存与失效策略；
5. local/mock 生产 break-glass 边界；
6. skeleton 迁移兼容策略。

## Phase 1: Design Output

已生成：
1. [data-model.md](./data-model.md)（核心实体、字段、关系、约束、状态）
2. [contracts/customer-auth.openapi.yaml](./contracts/customer-auth.openapi.yaml)（framework 级 API/adapter 契约骨架）
3. [quickstart.md](./quickstart.md)（迁移、测试与手工验证步骤）

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 无 | N/A | N/A |
