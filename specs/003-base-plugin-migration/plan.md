# Implementation Plan: Base Plugin Migration

**Branch**: `003-base-plugin-migration` · **Date**: 2025-11-01 · **Spec**: `specs/003-base-plugin-migration/spec.md`  
**Input**: Feature specification for migrating `com.powerx.plugin.base` CRUD capabilities into the PowerXPlugin skeleton/framework stack.

## Summary

Deliver end-to-end parity between the existing Base plugin example and the PowerXPlugin repository by (1) enhancing the Go framework so it natively supports CRUD routes with tenant-aware middleware and PowerX-formatted responses, (2) rebuilding the skeleton backend/frontend around in-memory Templates CRUD while keeping `.specify/memory/constitution.md` intact, and (3) aligning CLI templates/documentation so `px-plugin init` yields the same starter experience. The work is split into framework, backend skeleton, frontend/CLI, and documentation streams with explicit constitution checkpoints.

## Technical Context

- **Languages & Tooling**: Go 1.21+, Gin-compatible router abstractions, Node.js 18+, Nuxt 4.2, npm 9+.  
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
sdk/workspace/frontend/nuxt/
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

## Phase Plan

### Phase 0 – Discovery & Alignment
- Review `com.powerx.plugin.base` implementation to confirm API shapes, response envelopes, middleware behaviour, and front-end menu/i18n expectations.
- Inventory reusable assets (components, composables) and record deviations requiring simplification (e.g., Redis license cache).
- Validate outstanding clarifications (e.g., 404 response) are reflected in spec; note remaining open items (data scale, availability) for later documentation if needed.

### Phase 1 – Framework Enhancements (Spec US1)
- Extend `framework/backend/go/router`:
  - Implement path parameter parsing, query helpers, JSON binding on `bootstrap.Context`.
  - Add response helper producing `{success,data,error,timestamp,request_id}`.
- Add middleware stubs:
  - RequestID generator (UUID/ULID) attached to context.
  - Tenant resolver reading `X-Tenant-ID` or fallback to Standalone default, storing value for repository consumption.
- Update `@powerx-plugin/framework-client` to expose `get/post/put/delete`, injecting tenant header defaults.
- Document helper usage via package comments/tests (`*_test.go`) covering param parsing and envelope output.
- Capture coverage report (`go test -coverprofile`) ensuring framework router/middleware packages reach ≥90% statement coverage (SC-001 baseline).
- Regression: run `go test ./framework/backend/go/...`.

### Phase 2 – Skeleton Backend Migration (Spec US2)
- Create `skeleton/backend/internal/templates/` with:
  - `model.go` mirroring Base template fields.
  - Repository embedding `BaseRepository[Template]`, providing `NewTemplateRepository`, maintaining tenant isolation in memory (map keyed by tenant).
- Service orchestrating list/get/create/update/delete with validation hooks.
- HTTP handler using new framework context + response helper.
- Ensure repository `BeginTenantTx` executes `SET LOCAL app.tenant_id` per constitution and document behaviour.
- Seed demo data (optional) within in-memory store during bootstrap.
- Update router registration to mount `/api/v1/templates` CRUD + existing ping.
- Add unit tests for repository/service covering tenant isolation and CRUD flows.
- Add edge-case tests covering missing/invalid tenant headers and non-numeric template IDs returning structured errors.
- Update skeleton README explaining in-memory persistence, tenant defaults, and extension points.

### Phase 3 – Frontend & Framework Layers (Spec US3)
- Port Base plugin starter pages to `skeleton/web-admin/app`:
  - `intro.vue`, `templates/index.vue`, `templates/crud.vue`, supporting components.
  - Provide `app/composables/api/useTemplateApi.ts` referencing `framework-client`.
- Enhance `@powerx-plugin/framework-admin` Layer to:
  - Offer `starterPages` toggle that registers default menus/pages.
  - Expose hooks for registering menus (if not already).
- Ensure runtime config handles Standalone vs host base URLs, removing unused Bridge/debug artifacts.
- Write front-end smoke tests (Playwright or component tests) verifying create/edit/delete flows.
- `npm run lint` in skeleton and workspace.

### Phase 4 – CLI Templates & Documentation
- Sync skeleton backend/frontend into scaffold templates with placeholders for plugin IDs, names, menus, RBAC keys.
- Update CLI logic (if required) to ensure `px-plugin init` fills placeholders and optionally toggles starter pages.
- Add smoke script (Makefile target) that runs `px-plugin init com.powerx.demo --ui=starter`, diff-checks result vs skeleton expectation.
- Update `docs/guide/quickstart.md` and `docs/guide/standalone-mode.md` to include CRUD verification steps and limitations (in-memory store, AuthGuard 501).
- Append summary of changes to `docs/plan/002-plan-base-plugin-migration.md` if deviations arise.

## Testing & Validation

- `go test ./framework/backend/go/... -coverprofile` with coverage ≥90% (Phase 1 completion gate).  
- `go test ./skeleton/backend/...` with coverage of repository/service (Phase 2 gate).  
- `curl` or Postman collection hitting `/api/v1/templates` for two tenant IDs (Phase 2 gate).  
- `curl -w` latency check confirming Standalone CRUD ≤1s 平均时延（SC-002）。  
- `npm run lint` / `npm run build` for skeleton web-admin and framework workspace (Phase 3).  
- Playwright smoke for Templates CRUD (Phase 3 optional, recommended).  
- `px-plugin init` smoke plus diff check/`npm run lint` on generated project (Phase 4).  
- Document manual verification steps to satisfy Success Criteria SC-001..SC-004.

## Risks & Mitigations

- **Router regression risk**: Add comprehensive unit tests and retain backward compatibility (no breaking changes to existing handlers).  
- **Tenant isolation drift**: Mirror constitution rules exactly; include comments linking to `.specify/memory/constitution.md`.  
- **CLI template divergence**: Automate diff checks between skeleton and scaffold outputs.  
- **Frontend dependency bloat**: Keep imported libraries minimal; reuse existing Nuxt dependencies only.  
- **Documentation lag**: Track doc updates in acceptance checklist to avoid stale instructions.

## Next Steps

1. Execute Phase 0 discovery and confirm no additional clarifications required.  
2. Begin Phase 1 implementation on `framework/backend/go` and `framework-client`.  
3. Upon Phase 1 completion, proceed with Phase 2 skeleton backend work, following with Phase 3/4 sequentially.
