# Implementation Plan: 插件能力注册与暴露治理闭环

**Branch**: `006-plugin-capability` | **Date**: 2025-12-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-plugin-capability/spec.md`

## Summary

为插件生态交付统一的能力注册与多协议暴露管线：开发者在插件侧声明原子能力、协议矩阵与复合 Workflow/Agent 模板，Capabilities Manager 负责解析 `capabilities/*.yaml`、生成 OpenAPI/Proto/MCP/Workflow 契约，并在插件安装或升级时 3 分钟内同步至 PowerX Core（`capability_registry`、Workflow Builder、Agent Hub）。方案要求所有能力默认同步执行，必要时显式声明 `async`，并在目录同步失败时立即阻断安装回滚，确保宿主获取到一致的节点/工具目录。

## Technical Context

**Language/Version**: Backend Go 1.24、Frontend/CLI Node.js 20 + TypeScript 5 + Nuxt 4.2、脚本 Bash/Node 18 兼容。  
**Primary Dependencies**: Gin HTTP、Gorm ORM、PowerX Plugin Framework（admin/client layer）、px-plugin CLI、OpenAPI/Buf（proto）、Nuxt UI 3.3.x。  
**Storage**: PostgreSQL ≥13，单 schema（`powerx_plugin_base`）+ RLS；目录/契约文件存放在仓库内。  
**Testing**: Go `make test` + service/unit tests、Playwright/NUXT E2E、CLI 集成测试、Schema/contract lint（OpenAPI/Buf/capabilities lint）。  
**Target Platform**: PowerX 插件容器（Linux amd64）+ Web Admin（SSR/SPA via Nuxt）、PowerX Workflow/Agent Runtime。  
**Project Type**: Multi-surface（backend services + CLI tooling + web-admin 文档）。  
**Performance Goals**: 能力注册自动校验 ≤5 分钟、能力目录同步 ≤3 分钟、审核 SLA ≤2 工作日、暴露同步 100% 成功、Workflow/Agent 工具加载失败率 <1%。  
**Constraints**: Go/Nuxt 栈、STS 凭证、租户隔离（UUID/RLS）、manifest 与运行态保持一致、禁止未声明的通道、默认同步执行；能力目录或协议同步失败必须阻断回滚。  
**Scale/Scope**: 100+ 原子能力、几十个复合 Workflow/Agent 节点，支持千级租户并发查询，目标集成 API Gateway + Agent Hub + Workflow Builder。

## Constitution Check

- ✅ 语言/框架符合宪章（Go 1.24、Nuxt 4、Node 20）。
- ✅ 目录分层沿用 `internal/{services,transport}/`、`capabilities/*.yaml`，manifest 宣告符合 Host Contract。
- ✅ 数据存储限 Postgres + RLS；无跨 schema 行为。
- ✅ 依赖容器注入、Handler 轻薄、Service 中心化；无裸 `*gorm.DB` 暴露。
- ✅ 可观测 / 测试要求（审计、指标、Automation）在 spec 中已有指标/回滚策略。

> Gate 通过，可进入 Phase 0。

## Project Structure

```text
specs/006-plugin-capability/
├── spec.md
├── plan.md
├── research.md              # 本次 /speckit.plan 生成
├── data-model.md            # 本次 /speckit.plan 生成
├── quickstart.md            # 本次 /speckit.plan 生成
├── contracts/               # 本次 /speckit.plan 生成 (OpenAPI/Proto/MCP/Workflow 摘要)
└── tasks.md                 # /speckit.tasks 阶段生成
```

```text
skeleton/backend/
├── internal/
│   ├── transport/
│   │   ├── http/
│   │   │   └── admin/capability/**
│   │   └── grpc/capability/**
│   ├── services/
│   │   ├── capability/**
│   │   └── capability_registry/**
│   ├── handlers/
│   │   └── capabilities/<domain>/<action>_handler.go
│   ├── domain/
│   │   ├── models/
│   │   └── repository/
│   └── observability/**
├── cmd/
│   └── server
skeleton/scripts/
└── capabilities/** (lint/export/compose)

skeleton/web-admin/
├── app/
│   ├── pages/capabilities/**
│   ├── components/capabilities/**
│   ├── stores/
│   └── types/
└── tests/

contracts/
├── exposure/openapi.yaml
├── exposure/proto/
├── exposure/workflow/
└── exposure/mcp-tools.json
```

**Structure Decision**: 采用「后端（Go）+ CLI（Node）+ Web-admin（Nuxt）」三层结构，全部落在 `skeleton/` 目录树：`skeleton/backend/internal/{transport,services,domain}`、`skeleton/scripts/capabilities`、`skeleton/web-admin/app/**`，并结合仓库根下的 `contracts/` 与 `capabilities/` 目录用于声明、生成与同步。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _None_ | | |
