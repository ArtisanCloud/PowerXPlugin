# Implementation Plan: Framework IAM 统一封装（Standalone/Delegated）

**Branch**: `018-framework-iam-unification` | **Date**: 2026-04-11 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/018-framework-iam-unification/spec.md`

## Summary

将 IAM 契约上提至 framework，形成统一接口与 adapter 模型：
1. framework 定义 IAM 领域契约（组织、成员、角色、权限、token、context）；
2. local/delegated 实现通过 adapter 挂载；
3. skeleton 从“契约定义者”收敛为“默认适配实现”；
4. 提供可回归的迁移路径与兼容策略。

## Technical Context

- **Language/Version**: Go 1.24  
- **Primary Dependencies**: framework middleware/rbac/context；skeleton IAM 与 authproxy  
- **Storage**: 复用现有 plugin DB（local）与宿主 IAM 接口（delegated）  
- **Testing**: Go unit + contract tests + local/delegated integration tests  
- **Target Platform**: Linux plugin runtime + 本地开发环境  
- **Project Type**: backend framework + skeleton adapter migration

## Constitution Check

- Host Contract First: PASS（不绕过宿主 IAM 契约）  
- Tenant Isolation & Zero Trust: PASS（统一 tenant/user context 解析与校验）  
- Service-Centric Architecture: PASS（业务只依赖 framework 接口）  
- Observable & Testable Delivery: PASS（模式与权限判定统一可观测）

## Project Structure

```text
specs/018-framework-iam-unification/
├── spec.md
├── plan.md
└── tasks.md
```

```text
framework/backend/go/
└── iam/
    ├── contracts/
    ├── adapters/
    ├── context/
    └── errors/

skeleton/backend/go-gin/internal/
└── services/iam/
    ├── adapters/local/
    └── adapters/delegated/
```

## Milestones

1. M1: framework IAM 契约与错误语义冻结。  
2. M2: local/delegated adapter 接入并完成契约测试。  
3. M3: skeleton 路由与服务迁移到 framework IAM 接口。  
4. M4: 双模式回归通过并输出迁移指南。

