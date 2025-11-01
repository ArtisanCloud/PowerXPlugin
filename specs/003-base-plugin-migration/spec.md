# Feature Specification: Base Plugin Migration

**Feature Branch**: `003-base-plugin-migration`  
**Created**: 2025-11-01  
**Status**: Draft  
**Input**: Derived from `docs/plan/002-plan-base-plugin-migration.md`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Framework groundwork ready (Priority: P1)

As a framework maintainer, I can expose Router, response helper, and middleware primitives that support CRUD routes with path parameters, consistent response envelopes, and tenant-aware context so downstream skeletons can rely on them without reimplementing infrastructure.

**Why this priority**: Without these primitives the migration is blocked; all downstream skeleton and CLI outputs depend on the framework enhancements.

**Independent Test**: Run framework unit tests that mount mock handlers using the new Router features and assert response envelope/middleware behavior without involving skeleton code.

**Acceptance Scenarios**:

1. **Given** a handler registered at `/api/v1/templates/:id`, **When** an HTTP request hits `/api/v1/templates/42`, **Then** `bootstrap.Context.Param("id")` returns `"42"` and the tenant middleware resolves a default tenant ID when the header is absent.
2. **Given** a handler writes via the response helper, **When** it returns `{data: {...}}`, **Then** the JSON envelope includes `success`, `data`, optional `error`, `timestamp`, and `request_id` fields conforming to the documented schema.

---

### User Story 2 - Skeleton backend CRUD sample (Priority: P2)

As a skeleton maintainer, I can run `go run ./skeleton/backend/cmd/plugin` and interact with an in-memory Templates CRUD implementation that respects tenant isolation conventions so developers have a reference backend.

**Why this priority**: Provides the minimum viable example proving the migration is successful for backend consumers and unblocks front-end + CLI work.

**Independent Test**: Issue HTTP requests directly against the skeleton backend to exercise CRUD operations and tenant isolation without needing the front-end.

**Acceptance Scenarios**:

1. **Given** the skeleton backend running, **When** a POST to `/api/v1/templates` is made with JSON payload and header `X-Tenant-ID: 100`, **Then** the resource is stored in-memory, tagged with tenant 100, and GET `/api/v1/templates` scoped to tenant 100 returns it.
2. **Given** two tenants create templates, **When** tenant 100 requests tenant 200's template ID, **Then** the repository enforces `.specify/memory/constitution.md` rules and returns a 404 (or equivalent denial) without leaking the record.

---

### User Story 3 - Admin starter & CLI alignment (Priority: P3)

As a front-end/CLI maintainer, I can use the framework Layer starter pages and CLI templates to produce an admin UI showing intro + templates CRUD operations that communicate with the skeleton backend using `@powerx-plugin/framework-client`.

**Why this priority**: Completes the end-to-end example and ensures downstream plugin authors receive a fully functional starter.

**Independent Test**: Launch the skeleton web-admin dev server and perform CRUD actions against the backend, then render CLI templates and verify the generated project boots without manual fixes.

**Acceptance Scenarios**:

1. **Given** the skeleton frontend dev server running, **When** a user navigates to `/templates/crud`, **Then** they can create, edit, and delete templates with toast feedback using framework-client `get/post/put/delete`.
2. **Given** CLI templates render a new project, **When** the generated project is started, **Then** it contains the same starter pages and CRUD wiring as the skeleton with configurable plugin ID placeholders.

---

### Edge Cases

- Missing or malformed `X-Tenant-ID` header triggers middleware to apply Standalone defaults (tenant `1`) while logging a warning.
- Path parameters containing non-numeric template IDs return a structured error envelope with `success=false` and HTTP 400.
- Response helper handles handlers returning `nil` data or explicit errors without panicking and always produces the full envelope.
- CLI template rendering fails fast with a descriptive error if required placeholders (plugin ID, menus, permissions) are absent.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Framework Router MUST support path parameters, query strings, and JSON body binding through `bootstrap.Context`.
- **FR-002**: Framework Router MUST expose a response helper that emits envelopes containing `success`, `data`, `error`, `timestamp`, and `request_id`.
- **FR-003**: Framework middleware MUST populate request IDs and tenant context (`X-Tenant-ID` header or Standalone default) for every request.
- **FR-004**: `@powerx-plugin/framework-client` MUST expose `get`, `post`, `put`, and `delete` helpers that forward tenant headers automatically.
- **FR-005**: Skeleton backend MUST implement a Templates repository/service pair embedding `repository.BaseRepository[Template]`, providing `NewTemplateRepository`, and honoring tenant isolation per `.specify/memory/constitution.md`.
- **FR-006**: Skeleton backend MUST expose `/api/v1/templates` CRUD endpoints with in-memory storage and seed data for demo purposes.
- **FR-007**: Skeleton handlers MUST remain thin (validation + serialization) and delegate business logic to services reusable by future HTTP/gRPC transports.
- **FR-008**: Skeleton frontend MUST provide Intro and Templates CRUD pages that consume the framework client for API access and display toast/confirm feedback.
- **FR-009**: CLI templates MUST render backend and frontend skeleton assets with placeholder substitution for plugin IDs, menu labels, and RBAC keys.
- **FR-010**: Documentation (`docs/init-project.md`, quickstart, standalone guide) MUST outline new CRUD verification steps and clearly state current limitations (in-memory storage, AuthGuard 501, etc.).

### Key Entities *(include if feature involves data)*

- **TemplateRecord**: Tenant-scoped reusable template with attributes `id`, `tenant_id`, `name`, `description`, `content`, `created_at`, `updated_at`.
- **TenantContext**: Middleware-derived context capturing effective tenant ID and request metadata consumed by repositories for isolation checks.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Framework unit/integration tests covering Router path parameters, response helper, and middleware achieve ≥90% statement coverage.
- **SC-002**: Standalone `curl` smoke suite across two tenant IDs completes full CRUD cycle with average latency ≤1s per request.
- **SC-003**: Skeleton frontend manual QA completes create/edit/delete flows without console errors and reflects persisted data immediately.
- **SC-004**: CLI-rendered project passes `go test ./...` and `npm run lint` on first run with no manual code adjustments.
