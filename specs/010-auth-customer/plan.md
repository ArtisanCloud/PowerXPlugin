# Implementation Plan: Customer Auth Modes

**Branch**: `010-auth-customer` | **Date**: 2025-12-29 | **Spec**: `/specs/010-auth-customer/spec.md`
**Input**: Feature specification from `/specs/010-auth-customer/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Provide dual-mode Customer authentication for mini-app endpoints. Skeleton mode supplies `/mini-app/auth/register` and `/mini-app/auth/login` plus JWT issuance backed by `customer.Customer`. Delegated mode accepts host-issued tokens by calling the PowerX CRM validation endpoint and never persists host customers. A `CustomerAuthenticator` interface selected via configuration feeds a reusable middleware that injects `CustomerContext` into request scopes. Success relies on tenant isolation (RLS + EnsureTenant) and measurable latency targets (<500 ms delegated validation, <10 s Skeleton token issuance).

## Technical Context

**Language/Version**: Go 1.24 (backend), TypeScript 5 / Node 20 (existing Nuxt admin)  
**Primary Dependencies**: Gin HTTP stack, existing PowerX middleware (`httpmw`, JWT helpers), bcrypt password hashing, internal HTTP client abstractions for delegated calls  
**Storage**: PostgreSQL (schema `powerx_plugin_base`, table `customer.Customer`; no additional `customer_account` table—extended columns on the same model handle Skeleton credentials)  
**Testing**: `go test ./...` (unit + integration around middleware/services), Playwright/axios stubs not required this iteration  
**Target Platform**: Linux containers hosting PowerXPlugin backend; mini-app consumers via HTTP  
**Project Type**: Multi-tier (backend service + existing admin UI; feature scope focused on backend mini-app router)  
**Performance Goals**: Delegated validation median < 500 ms, Skeleton login roundtrip < 10 s end-to-end, 100% of `/mini-app/*` protected routes enforce tenant + customer tokens  
**Constraints**: Zero Trust (JWT validation, tenant RLS), Constitution-mandated service/repository layering, STS for outbound host calls, no host customer data persisted locally  
**Scale/Scope**: Support thousands of customers per tenant, multiple tenants concurrently, coverage for all `/mini-app` endpoints without per-endpoint duplication

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Plan Alignment |
| --- | --- |
| Host Contract First | Delegated mode uses host CRM validation endpoint via configurable client; no custom host contract introduced. |
| Tenant Isolation & Zero Trust | Middleware chains `httpmw.EnsureTenant()` before customer auth, tokens embed tenant UUID, repo operations remain tenant-scoped via `BaseRepository`. |
| Service-Centric Architecture | Introduces `CustomerAuthenticator` services + repository; HTTP handlers stay thin, using shared middleware/context helpers. |
| Observable & Testable Delivery | Plan requires structured audit logs (`customer.auth.*`) and Go tests for Skeleton/Delegated success + failure scenarios. |
| Minimal Footprint & Versioned Releases | Reuses existing schema + JWT stack; no new runtime beyond config toggles; documentation & quickstart delivered. |

**Gate Status**: PASS — requirements above are addressed in design and tracked in tasks.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
```text
backend/
├── internal/domain/customer/
│   └── models.go
├── internal/entity/repository/customer/
│   └── customer_repository.go
├── internal/services/customer/
│   ├── authenticator_factory.go
│   ├── local_authenticator.go
│   └── delegate_authenticator.go
├── internal/middleware/customer/
│   └── context.go
├── internal/transport/http/mini-app/
│   ├── routes.go
│   └── customer_handler.go
├── internal/transport/http/middleware/
│   └── customer_auth.go
├── internal/observability/customer/
│   └── audit_logger.go
├── tests/
│   ├── integration/mini-app/customer_auth_test.go
│   └── unit/customer_authenticator_test.go
└── etc/
    └── config.dev.yaml (customerAuth section)

frontend/ (no new source files required for this feature; existing admin may surface settings documentation only)
```

**Structure Decision**: Use existing backend/service layered layout; customer-specific code lives under `backend/internal/*` directories shown above. No new top-level projects are introduced, keeping footprint within Constitution guidelines.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _None_ |  |  |
