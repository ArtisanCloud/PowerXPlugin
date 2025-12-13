# Implementation Plan: Standalone 模式 IAM & RBAC

**Branch**: `007-standalone-iam-rbac` | **Date**: 2025-12-13 | **Spec**: [/specs/007-standalone-iam-rbac/spec.md](spec.md)
**Input**: Feature specification from `/specs/007-standalone-iam-rbac/spec.md`

## Summary

构建 Standalone 模式下可独立运行的 IAM & RBAC 能力：本地数据库存储租户/部门/成员/角色/权限，Web Admin 提供组织管理 UI，后端暴露 `/api/v1/admin/iam/**` 和 Local Auth/STS 接口，并与 Manifest/指标/文档同步，保证与宿主 PowerX RBAC 三元模型一致且在 Delegated 模式自动隐藏本地菜单。

## Technical Context

**Language/Version**: Go 1.24（后端），TypeScript 5 + Nuxt 4.2（前端），Node.js 20  
**Primary Dependencies**: Gin、Gorm、Pinia、`@nuxt/ui`、PowerX 插件框架、px-plugin CLI  
**Storage**: PostgreSQL（生产）/SQLite（本地），schema `powerx_plugin_base` 启用 IAM 表  
**Testing**: Go `go test ./...`、Playwright (`auth-local`, `iam-local`)、Nuxt 单测、CLI 集成测  
**Target Platform**: 插件后端服务 + Nuxt Web Admin，宿主 PowerX 可反代  
**Project Type**: Web（Go backend + Nuxt admin 前端）  
**Performance Goals**: 初始化 <2 min；RBAC 判定 <50ms；`iam export` 单租户 <10s  
**Constraints**: 遵循 `plugin/resource/action` 三元模型、菜单仅在 Standalone 模式显示、RLS/tenant_uuid 字段完整、STS 短期令牌 60s TTL  
**Scale/Scope**: 单实例 1–5 租户、每租户 1k 成员；未来可扩展 10k 角色绑定

## Constitution Check

- ✅ Host Contract：所有 API 均在 `/api/v1/**`，遵循 Manifest/RBAC schema；STS/CTX 兼容宿主。
- ✅ Tenant Isolation：模型全部携带 `tenant_uuid`，本地与 Delegated 模式均坚持零信任。
- ✅ Service-Centric：Handler→Service→Repo 分层，RBAC/审计/指标集中在 Service/observability。
- ✅ Observable/Testable：指标、审计、Playwright 及 Go 测试在范围内。

结论：无阻塞，进入 Phase 0。

## Project Structure

### Documentation (this feature)

```text
specs/007-standalone-iam-rbac/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md (由 /speckit.tasks 生成)
```

### Source Code (repository root)

```text
backend/
├── cmd/
├── internal/
│   ├── bootstrap/
│   ├── services/
│   ├── transport/http/admin/iam/
│   ├── entity/models/iam/
│   ├── middleware/
│   └── observability/
├── manifestx/
└── etc/

frontend/web-admin/
├── app/pages/admin/iam/
├── app/components/iam/
├── app/stores/
├── app/composables/
└── tests/

scripts/ & tools/
└── px-plugin CLI（`iam export/seed`）
```

**Structure Decision**: 采用现有 backend & web-admin 双仓布局，在 `internal/transport/http/admin/iam` 引入新 Handler，前端新增 `app/pages/admin/iam`，CLI 位于 `tools/cli`。

## Complexity Tracking

无偏离宪章的额外复杂度。
