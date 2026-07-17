# Implementation Plan: Standalone 模式 IAM & RBAC

**Branch**: `007-standalone-iam-rbac` | **Date**: 2025-12-13 | **Spec**: [/specs/007-standalone-iam-rbac/spec.md](spec.md)
**Input**: Feature specification from `/specs/007-standalone-iam-rbac/spec.md`

## Summary

构建 Standalone 模式下可独立运行的 IAM & RBAC 能力：本地数据库存储租户/部门/成员/角色/权限，Web Admin 提供组织管理 UI，后端暴露 `/api/v1/admin/iam/**` 和 Local Auth/STS 接口，并与 Manifest/指标/文档同步，保证与宿主 PowerX RBAC 三元模型一致；delegated provider 下正式菜单保持可见，由 RBAC 与 read-only 状态控制操作。

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

## 数据模型与 Runbook 对齐（2025-12-15 更新）

- `iam_users`/`iam_members` 的职责区分需要在数据模型文档中明确：Account 持有跨租户凭证，Member 负责租户内的组织/RBAC 状态，并通过 `iam_member_roles` 关联 `iam_roles`。
- 所有 IAM 相关表（tenants/users/members/roles/permissions/departments/member_roles/role_permissions/refresh_tokens/audit_logs）必须在迁移与模型层保持一致命名，并在 Runbook 中提供“同一账号加入多个租户”的验证步骤，方便运维复现问题。
- Phase 4 的文档任务需补充上述说明；若后续实现导致模型再次变更，需同时回传到 `data-model.md` 与 `docs/operations/runbooks/iam-rbac.md`。

## UI Parity Tasks（参考 PowerX settings 页面）

| Task | Description | Reference |
| ---- | ----------- | --------- |
| U1 | 在 `/admin/iam/overview` 建立“系统设置”式的卡片导航与快速设置区块，重用 `UCard + grid + quick settings` 模式，确保入口与 `settings/index.vue` 相似。 | `Core/PowerX/web-admin/app/pages/settings/index.vue` |
| U2 | 复制 `settings/users/index.vue` 的 Tab 行为（部门/用户/权限），并基于 Pinia store 判断 root/tenant-admin 显隐权限 Tab；部门/成员/权限三个子组件需要 Standalone 版本。 | `Core/PowerX/.../settings/users/index.vue` |
| U3 | 在角色列表中实现租户远程搜索、分页、过滤与克隆/编辑抽屉，逻辑对齐 `components/settings/users/RoleManager.vue`（含 `useOneShotAlert` 提示与 scope/scope_type 过滤）。 | `Core/PowerX/.../components/settings/users/RoleManager.vue` |
| U4 | 将租户配置表单（Plan Drawer、创建租户 Modal、功能开关）设计成 `settings/config` 的“快速设置”风格，复用 label/description/按钮布局与 `USelect`/`USwitch` 交互。 | `Core/PowerX/web-admin/app/pages/settings/config/index.vue` |

> 以上任务要在 skeleton + scaffold 模板同时实现，并在 PR 中附上对比截图；Quickstart 第 2.1 节已加入 UI 自检 checklist。
