# Development Tasks: Customer Auth Modes

**Branch**: `010-auth-customer` | **Spec**: `/specs/010-auth-customer/spec.md`

> Checklist format: `- [ ] T001 [P] [StoryLabel] Description`

## Phase 1 — Setup

- [x] T001 Verify `customer_auth` config section exists in `skeleton/backend/go-gin/etc/config.example.yaml` and add `mode`, `delegate_endpoint`, `jwt_*`, `cache_ttl_seconds` defaults
- [x] T002 Ensure `skeleton/backend/go-gin/internal/domain/customer/models.go` includes `CustomerContext` and `CustomerAuthConfig` structs per data-model.md
- [x] T003 Update `skeleton/backend/go-gin/cmd/database/migrate/migrate.go` to include `customer_accounts` table for Skeleton mode usage
- [x] T004 Document mode selection in `docs/guides/develop/auth/customer.md` referencing the new `customer_auth` config keys

## Phase 2 — Foundational Infrastructure

- [x] T005 Implement `backend/internal/services/customer/authenticator_factory.go` returning local vs delegate implementations based on config
- [x] T006 Create `backend/internal/middleware/customer/context.go` with setter/getter helpers storing `CustomerContext` in Gin + request contexts
- [x] T007 Wire `backend/internal/transport/http/mini-app/routes.go` to run `customerhttp.Authenticate` before `httpmw.EnsureTenant()` for all `/mini-app/*` routes
- [x] T008 Add observability helpers in `backend/internal/observability/customer/audit_logger.go` to log register/login/validation outcomes with tenant + customer identifiers

## Phase 3 — User Story 1 (P1)

_Goal_: Customer access to protected mini-app APIs with tenant-isolated tokens  
_Independent Test_: Start backend, run `/mini-app/orders` (or sample endpoint) with: (a) missing token -> 401, (b) mismatched tenant -> 403, (c) valid token -> 200

- [x] T009 [US1] Enhance `backend/internal/middleware/customer_auth.go` to extract headers (`Authorization`, `X-Customer-Token`), call authenticator, enforce tenant match, and inject `CustomerContext`
- [x] T010 [US1] Add unit tests in `backend/tests/unit/customer_authenticator_test.go` covering happy path, missing token, tenant mismatch, and disabled account cases
- [x] T011 [US1] Create integration tests under `backend/tests/integration/mini-app/*` simulating `/mini-app/ping` calls with valid/invalid tokens
- [x] T012 [US1] Update `backend/internal/transport/http/mini-app/*` handlers to read `CustomerContext` via helper instead of parsing headers directly (no business logic change)
- [x] T026 [US1] Review all `/mini-app` handlers and middleware responses to ensure success and error paths use `contracts.Response*` envelopes consistently

## Phase 4 — User Story 2 (P2)

_Goal_: Skeleton mode login & token issuance  
_Independent Test_: Register a customer, log in to receive JWT, call `/mini-app/*` using that JWT and observe access granted

- [x] T013 [US2] Extend `backend/internal/entity/repository/customer/customer_repository.go` with CRUD helpers (`FindByEmailOrPhone`, `CreateCustomer`, `UpdateStatus`) scoped by tenant
- [x] T014 [US2] Implement `backend/internal/services/customer/local_authenticator.go` handling registration (bcrypt hash) and token issuance with JWT helpers
- [x] T015 [US2] Add mini-app handlers in `backend/internal/transport/http/mini-app/customer_handler.go` for `/mini-app/auth/register` and `/mini-app/auth/login` following `contracts/mini-app/auth.openapi.yaml`
- [x] T016 [US2] Register routes and RBAC for the new endpoints within `backend/internal/transport/http/mini-app/routes.go` (guarded by Skeleton mode)
- [x] T017 [US2] Add API tests (Go integration or Postman collection) covering register/login success + failure, ensuring quickstart steps remain valid
- [x] T027 [US2] Extend login tests to cover token expiration/replay attempts and disabled customer states (reject with 401/423) in `backend/tests/integration/mini-app/*`

## Phase 5 — User Story 3 (P3)

_Goal_: Delegated mode validation using host CRM endpoint  
_Independent Test_: Configure `customerAuth.mode=delegate` with mock host endpoint; valid host token yields 200, invalid token returns 401, tenant mismatch returns 403

- [x] T018 [US3] Implement `backend/internal/services/customer/delegate_authenticator.go` to call configured host validation endpoint (HTTP client, timeout, optional cache)
- [x] T019 [P] [US3] Add retries + structured error mapping in delegate authenticator to produce meaningful unauthorized responses without persisting data
- [x] T020 [US3] Introduce configuration loader changes to ensure delegate-specific fields (endpoint, timeout) are required when mode=delegate
- [x] T021 [US3] Write integration test using stub host server to verify success, host error, and tenant mismatch flows
- [x] T028 [US3] Add negative integration tests simulating unreachable delegate endpoint, misconfigured URLs, and cache TTL violations to ensure graceful degradation

## Phase 6 — Polish & Cross-cutting

- [x] T022 Update `docs/guides/develop/auth/customer.md` with step-by-step instructions for both modes, referencing quickstart commands
- [x] T023 Add monitoring/alerting guidance to `docs/plan/010-auth-customer.md` or README (log keys, metrics)
- [x] T024 Ensure `quickstart.md` commands are linked from `README.md` or relevant docs
- [x] T025 Verify configuration examples in `plugin.yaml` / manifests list the new settings where appropriate
- [x] T029 Instrument latency metrics for Skeleton login and Delegated validation (e.g., log timings or expose Prometheus metrics) to verify SC-003 thresholds

## Dependencies

| From Story | To Story | Reason |
| --- | --- | --- |
| Setup → Foundational | Setup config + models needed before authenticator work |
| Foundational → Story 1 | US1 middleware requires factory/context groundwork |
| Story 1 → Story 2 | Skeleton endpoints depend on core middleware/token flow |
| Story 1 → Story 3 | Delegated mode plugs into same middleware/context as US1 |

## Parallel Execution Opportunities

- Setup tasks touch different files (config vs models) and can be parallelized after initial planning.  
- Foundational tasks T005–T008 operate on separate directories; each can be owned independently.  
- Story 2 splits between repository/service (T013–T014) and handlers/tests (T015–T017).  
- Story 3’s retry/error mapping (T019) can proceed concurrently with initial delegate client (T018) once interfaces are defined.

## Implementation Strategy

- **MVP**: Complete through User Story 1 to ensure all `/mini-app/*` endpoints require tenant + customer tokens.  
- **Increment**: Deliver Skeleton login/register (Story 2) to unblock standalone testing.  
- **Final**: Implement Delegated validation (Story 3) before production rollout.  
- Add observability and documentation polish in Phase 6, then proceed to `/speckit.tasks` for execution tracking.
