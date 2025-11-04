# Implementation Plan: Base Plugin Migration

**Branch**: `003-base-plugin-migration` · **Date**: 2025-11-01 · **Spec**: `specs/003-base-plugin-migration/spec.md`  
**Input**: Feature specification for migrating `com.powerx.plugin.base` CRUD capabilities into the PowerXPlugin skeleton/framework stack.

## Summary

Deliver end-to-end parity between the existing Base plugin example and the PowerXPlugin repository by (1) enhancing the Go framework so it natively supports CRUD routes with tenant-aware middleware and PowerX-formatted responses, (2) rebuilding the skeleton backend/frontend around in-memory Templates CRUD while keeping `.specify/memory/constitution.md` intact, and (3) aligning CLI templates/documentation so `px-plugin init` yields the same starter experience. The work is split into framework, backend skeleton, frontend/CLI, and documentation streams with explicit constitution checkpoints.

## Technical Context

- **Languages & Tooling**: Go 1.24+, Gin-compatible router abstractions, Node.js 18+, Nuxt 4.2, npm 9+.  
- **Framework Packages**: `framework/backend/go/router`, `bootstrap`, `middleware`, `manifest`, `@powerx-plugin/framework-client`, `@powerx-plugin/framework-admin`.  
- **Skeleton Targets**: `skeleton/backend` (Go module), `skeleton/web-admin` (Nuxt app).  
- **CLI & Templates**: `scaffold/templates/backend/go-gin`, `scaffold/templates/web/nuxt`, `tools/cli`.  
- **Docs & Contracts**: `docs/plan/001-init-project.md`, `docs/plan/002-plan-base-plugin-migration.md`, `docs/guide/quickstart.md`, `.specify/memory/constitution.md`.  
- **Testing Stack**: `go test`, integration curl suites, Nuxt lint/build, Playwright (optional smoke), CLI generation diff checks.

## Constitution Check

- **双重使命仓库** ✅ Framework and skeleton work progress in parallel; CLI templates are updated to keep generated artifacts aligned.  
- **契约优先兼容性** ✅ Response envelope schemas and tenant handling conform to PowerX conventions and reference `.specify/memory/constitution.md`.  
- **Go + Nuxt 基线** ✅ No deviation from mandated stack; assets remain within current Go/Nuxt layers.  
- **脚手架与 CLI 纪律** ✅ Skeleton changes are mirrored inside scaffold templates and exercised via CLI smoke tests.  
- **透明交付与一致性** ✅ Plan calls for documentation updates, examples, and smoke verification to demonstrate observable parity.

## Scope & Deliverables

```
framework/backend/go/
  router/                 # Path param, response helper, context upgrades
  middleware/             # RequestID, tenant context stubs
framework/frontend/nuxt/
  framework-client/       # add put/delete helpers
  framework-admin/        # starter page toggle, layout wiring
skeleton/backend/
  cmd/plugin/
  internal/
    templates/            # model, repository, service, handler (in-memory)
    routes/               # register CRUD routes (+ ping)
skeleton/web-admin/
  app/pages/              # intro.vue, templates/*
  app/components/         # TemplateFormModal, ConfirmDialog, Toast
  app/composables/        # useTemplateApi example
scaffold/templates/
  backend/go-gin/         # mirrored skeleton backend
  web/nuxt/               # mirrored skeleton frontend
docs/
  plan/002-plan-base-plugin-migration.md (link-back)
  guide/quickstart.md     # CRUD validation steps
  guide/standalone-mode.md# standalone CRUD walkthrough
.specify/
  memory/constitution.md  # reference only (no edits unless required)
```

## Phase Plan（修订）

### Phase 0 – Discovery & Baseline（完成）
- 对照 Base 插件整理 API/响应/中间件/Nuxt 配置，沉淀在 `research.md`。  
- 明确暂不迁移的能力（数据库、完整权限、Bridge 等），后续文档需说明。

### Phase 1 – Framework Enhancements（完成，US1）
- Router Param/Query/BindJSON、响应助手、RequestID/TenantContext 中间件。  
- `@powerx-plugin/framework-client` PUT/DELETE，`framework-admin` StarterPages toggle。  
- 覆盖率 ≥90%，回归 `go test ./framework/backend/go/...`。

### Phase 2 – Skeleton Backend Parity（进行中，US2）
- **已交付**：内存仓储 + Service/Handler + 单测 + Tenant 隔离。  
- **待补强**：迁移 Base `dto`/错误码、分页响应；补足 Handler 校验路径；文档说明与数据库版本差异。

### Phase 3 – Nuxt 配置 & 页面对齐（进行中，US3/US4）
- 复制 Base `nuxt.config.ts` 的关键项：`compatibilityDate`、`ssr`、`baseURL`、`runtimeConfig`、Nitro headers、`@nuxt/icon`、`@pinia/nuxt`、`@nuxtjs/color-mode`、HMR/代理。  
- Skeleton 前端提供 Home + Intro + Templates CRUD；语言包存于 `i18n/`；Bridge/Dev Console 视需求列入差异说明。  
- Layer/CLI 模板同步上述配置，确保 `POWERX_PROXY` 切换可用；`npm run lint`、手动 CRUD 验证通过。

### Phase 4 – CLI 模板同步与生成验证（进行中）
- 将 Skeleton Go/Nuxt 结构复制到 `scaffold/templates/**` 与 CLI 内嵌模板，完善占位符。  
- `./bin/px-plugin init` smoke：生成项目 `go test ./...`、`npm run lint`、页面访问与 Skeleton 一致；补充 CLI README。  
- 若发现差异，回写 `research.md` 与本计划。

### Phase 5 – 文档与差异透明化（进行中）
- 更新 Quickstart、Standalone、Plan（本文）等文档，列出运行步骤、环境变量、延迟测试示例、当前限制。  
- 在 `tasks.md` 维护「未迁能力」列表供后续排期。

### Phase 6 – 余项评估（待启动）
- 评估是否需要 Stub Bridge/Operations/Stores；若不迁移，编写 ADR 或 README 说明。  
- output 升级指南：如何从内存仓储切换到数据库、如何接入权限服务。

## Testing & Validation

- `go test ./framework/backend/go/... -coverprofile` with coverage ≥90% (Phase 1 completion gate).  
- `go test ./skeleton/backend/...` with coverage of repository/service (Phase 2 gate).  
- `curl` or Postman collection hitting `/api/v1/templates` for two tenant IDs (Phase 2 gate).  
- `curl -w` latency check confirming Standalone CRUD ≤1s 平均时延（SC-002）。  
- `npm run lint` / `npm run build` for skeleton web-admin and framework workspace (Phase 3).  
- Playwright smoke for Templates CRUD (Phase 3 optional, recommended).  
- `px-plugin init` smoke plus diff check/`npm run lint` on generated project (Phase 4)。  
- Document manual verification steps to satisfy Success Criteria SC-001..SC-005.

## Risks & Mitigations

- **Router regression risk**: Add comprehensive unit tests and retain backward compatibility (no breaking changes to existing handlers).  
- **Tenant isolation drift**: Mirror constitution rules exactly; include comments linking to `.specify/memory/constitution.md`.  
- **CLI template divergence**: Automate diff checks between skeleton and scaffold outputs.  
- **Frontend dependency bloat**: Keep imported libraries minimal; reuse existing Nuxt dependencies only.  
- **Documentation lag**: Track doc updates in acceptance checklist to avoid stale instructions.

## Next Steps

1. Phase 2 补强（DTO/错误码/README）；完成后进入 Phase 3 Nuxt 配置 parity。  
2. Phase 3 同步完成后立即执行 Phase 4 CLI smoke 并更新模板依赖。  
3. Phase 5 文档回填，整理 Phase 6 未迁任务并排期。
