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
2. **Given** two tenants create templates, **When** tenant 100 requests tenant 200's template ID, **Then** the repository enforces `.specify/memory/constitution.md` rules and returns a 404 Not Found without leaking the record.

---

### User Story 3 - Admin starter & CLI alignment (Priority: P3)

As a front-end/CLI maintainer, I can use the framework Layer starter pages and CLI templates to produce an admin UI showing intro + templates CRUD operations that communicate with the skeleton backend using `@artisan-cloud/plugin-framework-client`.

**Why this priority**: Completes the end-to-end example and ensures downstream plugin authors接收的骨架与 Base 插件体验一致。

**Independent Test**: Launch the skeleton web-admin dev server并执行 CRUD；随后用 CLI 生成项目并验证其按相同配置启动。

**Acceptance Scenarios**：

1. **Given** skeleton 前端启动，**When** 访问 `/templates/crud`，**Then** 可完成创建/编辑/删除模板并收到 Toast 提示，页面无控制台错误。
2. **Given** 使用 `px-plugin init` 生成新项目，**When** 启动其前端，**Then** 首页 `/` 与 `/_p/{pluginId}/admin/templates/crud` 呈现与 Skeleton 一致的页面，并持有正确的菜单/i18n/运行时配置。

---

### User Story 4 – Nuxt 配置与运行时对齐 (Priority: P3)

作为模板维护者，我需要 Skeleton 与 CLI 输出的 Nuxt 工程复刻 Base 插件的关键配置（`baseURL`、`runtimeConfig`、Nitro headers、HMR 代理、`@nuxt/icon`、`@pinia/nuxt` 等），以便开发者能够在 Standalone 与宿主双场景下开箱运行。

**Independent Test**: 对比 `com.powerx.plugin.base/web-admin/nuxt.config.ts` 与 Skeleton/CLI 模板的 Diff；手动/脚本验证 Standalone 下 `/`、`/_p/{pluginId}/admin` 访问路径、语言包加载、HMR、代理行为。

**Acceptance Scenarios**：

1. **Given** Skeleton 在 Standalone 模式运行，**When** 访问 `/` 与 `/_p/com.powerx.sample/admin`，**Then** 页面均可渲染，语言包正确加载，`runtimeConfig.public` 反映 Standalone API 基址。
2. **Given** 通过环境变量切换 `POWERX_PROXY=1`，**When** 重新启动 Skeleton，**Then** `app.baseURL`、`apiBaseUrl`、Nitro headers 与 Base 插件一致，CLI 生成项目亦保持同样开关。

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
- **FR-004**: `@artisan-cloud/plugin-framework-client` MUST expose `get`, `post`, `put`, and `delete` helpers that forward tenant headers automatically.
- **FR-005**: Skeleton backend MUST implement a Templates repository/service pair embedding `repository.BaseRepository[Template]`, providing `NewTemplateRepository`, and honoring tenant isolation per `.specify/memory/constitution.md`.
- **FR-006**: Skeleton backend MUST expose `/api/v1/templates` CRUD endpoints with in-memory storage and seed data for demo purposes.
- **FR-007**: Skeleton handlers MUST remain thin (validation + serialization) and delegate business logic to services reusable by future HTTP/gRPC transports.
- **FR-008**: Skeleton frontend MUST提供 Intro、主页、Templates CRUD 页面，消费 framework-client 并复刻 Base 插件导航/菜单结构。
- **FR-009**: Skeleton Nuxt 配置 MUST 复刻 Base 插件关键项：`compatibilityDate`、`ssr`、`baseURL`、`runtimeConfig`、Nitro headers、`@nuxt/icon`、`@pinia/nuxt`、`@nuxtjs/color-mode`、HMR/代理设置及语言包目录（`i18n/locales`）。
- **FR-010**: CLI 模板 MUST 同步 Skeleton 的 Nuxt/Go 结构、依赖与配置，占位符包含插件 ID、菜单、RBAC、API 基址。
- **FR-011**: 文档 (`docs/guide/quickstart.md`,`docs/guide/standalone-mode.md`,`docs/plan/002-plan-base-plugin-migration.md`) MUST 更新差异说明、运行步骤、局限性。

### Key Entities *(include if feature involves data)*

- **TemplateRecord**: Tenant-scoped reusable template with attributes `id`, `tenant_id`, `name`, `description`, `content`, `created_at`, `updated_at`.
- **TenantContext**: Middleware-derived context capturing effective tenant ID and request metadata consumed by repositories for isolation checks.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Framework unit/integration tests covering Router path parameters, response helper, and middleware achieve ≥90% statement coverage.
- **SC-002**: Standalone `curl` smoke suite across two tenant IDs completes full CRUD cycle with average latency ≤1s per request.
- **SC-003**: Skeleton frontend manual QA completes create/edit/delete flows without console errors and reflects persisted data immediately，且首页、Admin 路由、语言切换均正常。
- **SC-004**: CLI-rendered project passes `go test ./...` and `npm run lint` on first run with no manual code adjustments，并在 Standalone 下 / 与 Admin 路由可访问。
- **SC-005**: Skeleton 与 Base 插件 `nuxt.config.ts` 的关键信息差异需在 `research.md` 中记录并得到确认（要么对齐要么注明不迁）。

## Clarifications

### Session 2025-11-01

- Q: 当租户 A 请求租户 B 的模板 ID 时，后端 `/api/v1/templates/:id` 应返回哪种 HTTP 状态码？ → A: 404 Not Found (隐藏资源是否存在)
