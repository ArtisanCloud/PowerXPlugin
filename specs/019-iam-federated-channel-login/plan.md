# Implementation Plan: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

**Branch**: `019-iam-federated-channel-login` | **Date**: 2026-04-12 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/019-iam-federated-channel-login/spec.md`

## Summary

实现可版本化复用的联邦扫码登录能力，重点是将 provider registry/factory 与默认渠道实现（企微/钉钉/飞书）上提到 framework，skeleton 仅负责装配与路由接线。  
本次计划将交付 challenge/risk 校验、identity binding/JIT、映射策略、双模式上下文一致性与审计风控闭环。

## Technical Context

**Language/Version**: Go 1.24, TypeScript 4.x (Nuxt 4.2)  
**Primary Dependencies**: framework IAM contracts/context/errors, skeleton IAM/auth service, RBAC, observability/audit  
**Storage**: PostgreSQL/SQLite（复用 IAM 表并新增 external_identity/binding/challenge/risk_event）  
**Testing**: Go unit/integration tests, callback security regression tests, admin/public flow tests  
**Target Platform**: Linux plugin runtime + Web admin/public endpoints  
**Project Type**: framework reusable capability + skeleton integration  
**Performance Goals**: 联邦回调判定 p95 < 200ms（不含外部 provider 网络耗时）；主链路成功率灰度期 >= 98%  
**Constraints**: Host Contract First、Tenant UUID、Zero Trust；delegated 以宿主会话为权威；渠道故障不影响密码登录；目录分层遵循 `internal/domain/{models,repository}`；新增持久化模型必须注册到 migrate 入口  
**Scale/Scope**: 首批支持 3 渠道（企微/钉钉/飞书），覆盖 2 种运行模式（standalone/delegated）

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Gate

1. Host Contract First: PASS  
delegated 模式下以宿主令牌为权威，插件仅适配上下文。
2. Tenant Isolation & Zero Trust: PASS  
challenge/risk 模型覆盖 replay/expired/cross-tenant/signature 风险校验。
3. Service-Centric Architecture: PASS  
framework 提供复用能力，skeleton 仅做装配与接线，不重复实现底层 provider。
4. Observable & Testable Delivery: PASS  
要求审计事件字段标准化并配套安全回归。
5. Event Contracts & TaskBus Readiness: PASS（N/A）  
本特性不引入新的事件总线协议，复用现有观测管道。
6. Minimal Footprint & Versioned Releases: PASS  
抽象上提到 framework，降低后续插件接入重复成本。

### Post-Design Gate

Phase 1 产物（research/data-model/contracts/quickstart）完成后复查：PASS。无新增违例。

## Project Structure

### Documentation (this feature)

```text
specs/019-iam-federated-channel-login/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── federated-login.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
framework/backend/go/
└── iam/federated/
    ├── contracts/
    ├── providers/
    │   ├── registry.go
    │   ├── wecom/
    │   ├── dingtalk/
    │   └── lark/
    ├── challenge/
    └── risk/

skeleton/backend/go-gin/internal/
├── bootstrap/
├── domain/
│   ├── models/
│   └── repository/
├── services/iam/federated/
├── transport/http/public/auth/
├── transport/http/admin/iam/
└── observability/auth/
```

**Structure Decision**: 采用“framework 复用能力 + skeleton 装配接线”结构。任何 provider factory 与默认渠道实现均放 framework，skeleton 仅保留插件运行时依赖注入与路由对接。

## Phase 0: Research Output

已生成 [research.md](./research.md)，覆盖：
1. framework 承载 provider factory 与默认实现的边界；
2. JIT 默认策略（唯一匹配自动绑定）；
3. 映射策略默认生效时机（版本变化才重算）；
4. 风控拒绝反馈策略（错误码可区分，前端文案统一）；
5. delegated 权威会话策略（宿主权威，插件最小缓存）。

## Phase 1: Design Output

已生成：
1. [data-model.md](./data-model.md)（关键实体、关系、约束、状态）  
2. [contracts/federated-login.openapi.yaml](./contracts/federated-login.openapi.yaml)（联邦登录契约草案）  
3. [quickstart.md](./quickstart.md)（接入与验证流程）

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 规格中包含架构约束（framework vs skeleton） | 该边界是业务方明确冻结要求 | 仅写业务需求会导致实现走偏，无法保障复用目标 |
