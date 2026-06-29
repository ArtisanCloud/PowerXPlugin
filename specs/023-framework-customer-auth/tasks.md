# Tasks: Framework Customer Identity/Auth

**Input**: Design documents from `/specs/023-framework-customer-auth/`  
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/customer-auth.openapi.yaml`, `quickstart.md`  
**Tests**: This feature requires framework unit tests, skeleton integration/regression tests, config guard tests, and template parity checks because customer auth is a security boundary.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行，前提是任务改动不同文件且不依赖未完成任务。
- **[Story]**: 仅用户故事阶段使用，格式为 `[US1]`、`[US2]` 等。
- 每个任务必须包含明确文件路径。

## Phase 1: Setup

- [X] T001 Confirm existing customer auth baseline with `go test ./skeleton/backend/go-gin/tests/integration/mini-app ./skeleton/backend/go-gin/tests/unit` and record failures in `specs/023-framework-customer-auth/quickstart.md`.
- [X] T002 [P] Create framework customer auth package README in `framework/backend/go/runtime/customerfw/README.md`.
- [X] T003 [P] Add customer auth contract validation notes to `specs/023-framework-customer-auth/contracts/customer-auth.openapi.yaml`.
- [X] T004 [P] Inventory current skeleton customer auth dependencies with `rg "CustomerContext|CustomerAuth|Authenticator|customer_auth" skeleton/backend/go-gin scaffold/templates tools/cli/internal/templates/data`.

---

## Phase 2: Foundational Framework Contracts

**CRITICAL**: Complete this phase before user-story implementation. These files define the shared contract used by every story.

- [X] T005 Create `CustomerContext`, `CustomerAuthSource`, context setter/getter, and Gin helper contracts in `framework/backend/go/runtime/customerfw/context.go`.
- [X] T006 [P] Create stable error codes and HTTP/status mapping in `framework/backend/go/runtime/customerfw/errors.go`.
- [X] T007 [P] Define `CustomerTokenValidator`, `CustomerTokenValidationResult`, and multi-token validation contract in `framework/backend/go/runtime/customerfw/validator.go`.
- [X] T008 [P] Define `CustomerMembership`, `CustomerMembershipResolver`, status constants, and cache policy types in `framework/backend/go/runtime/customerfw/membership.go`.
- [X] T009 [P] Define `BootstrapInput`, `BootstrapContext`, and bootstrap resolver contract in `framework/backend/go/runtime/customerfw/bootstrap.go`.
- [X] T010 [P] Define `CustomerAuthClient`, register/login inputs, and auth result contract in `framework/backend/go/runtime/customerfw/auth_client.go`.
- [X] T011 [P] Define diagnostics/audit field helpers and source-mode diagnostics in `framework/backend/go/runtime/customerfw/diagnostics.go`.
- [X] T012 [P] Add unit tests for context helper behavior in `framework/backend/go/runtime/customerfw/context_test.go`.
- [X] T013 [P] Add unit tests for error mapping and secret redaction in `framework/backend/go/runtime/customerfw/errors_test.go`.

**Checkpoint**: Framework has customer auth public contracts without skeleton dependencies.

---

## Phase 3: User Story 1 - 插件统一获取 C 端身份上下文 (Priority: P1)

**Goal**: Protected C-end handlers and services can read one normalized customer context without re-parsing request tokens.

**Independent Test**: A request with a valid customer token reaches a protected route, framework injects context, and downstream service reads the same context from request context.

### Tests

- [X] T014 [P] [US1] Add middleware success/missing-token tests and audit field assertions in `framework/backend/go/runtime/customerfw/middleware_test.go`.
- [X] T015 [P] [US1] Add skeleton route regression test for framework context injection in `skeleton/backend/go-gin/tests/integration/mini-app/customer_framework_context_test.go`.

### Implementation

- [X] T016 [US1] Implement customer authenticate middleware token extraction, context injection, and auth decision audit emission in `framework/backend/go/runtime/customerfw/middleware.go`.
- [X] T017 [US1] Add skeleton adapter from existing customer authenticator to framework validator in `skeleton/backend/go-gin/internal/services/customer/framework_adapter.go`.
- [X] T018 [US1] Replace skeleton Gin customer context wrapper to delegate to framework context in `skeleton/backend/go-gin/internal/middleware/customer/context.go`.
- [X] T019 [US1] Update mini-app customerhttp wrapper to call framework middleware in `skeleton/backend/go-gin/internal/transport/http/mini-app/customerhttp/authenticate.go`.
- [X] T020 [US1] Update mini-app protected route context probe to use framework context in `skeleton/backend/go-gin/internal/transport/http/mini-app/routes.go`.

**Checkpoint**: Existing protected mini-app requests can read framework customer context.

---

## Phase 4: User Story 2 - 阻止 customer token 跨租户使用 (Priority: P1)

**Goal**: Tenant-scoped resources require a current tenant resolved from token, request, or bootstrap; mismatches and missing tenant are rejected before business logic.

**Independent Test**: A globally scoped token without tenant is rejected for tenant-scoped access; the same token with active membership and resolved tenant succeeds.

### Tests

- [X] T021 [P] [US2] Add global-token tenant-required tests in `framework/backend/go/runtime/customerfw/middleware_tenant_test.go`.
- [X] T022 [P] [US2] Add multi-token conflict tests in `framework/backend/go/runtime/customerfw/validator_test.go`.
- [X] T023 [P] [US2] Add skeleton tenant-required/mismatch integration tests in `skeleton/backend/go-gin/tests/integration/mini-app/customer_tenant_scope_test.go`.

### Implementation

- [X] T024 [US2] Implement tenant resolution from request, token result, and bootstrap context in `framework/backend/go/runtime/customerfw/tenant.go`.
- [X] T025 [US2] Implement tenant mismatch comparison across request/token/bootstrap in `framework/backend/go/runtime/customerfw/middleware.go`.
- [X] T026 [US2] Implement multi-token consistency validation in `framework/backend/go/runtime/customerfw/validator.go`.
- [X] T027 [US2] Update skeleton middleware adapter to preserve existing tenant injection behavior through framework tenant resolver in `skeleton/backend/go-gin/internal/middleware/customer_auth.go`.
- [X] T028 [US2] Update contract examples for tenant-required and token conflict errors in `specs/023-framework-customer-auth/contracts/customer-auth.openapi.yaml`.

**Checkpoint**: Tenant-scoped C-end access cannot proceed without resolved tenant and cannot cross tenant contexts.

---

## Phase 5: User Story 3 - 校验 customer 与 tenant membership (Priority: P1)

**Goal**: Framework resolves customer membership for current tenant and rejects missing/inactive membership before tenant-scoped business logic.

**Independent Test**: Active membership succeeds; missing, suspended, disabled, deleted, or expired membership fails with stable errors.

### Tests

- [X] T029 [P] [US3] Add membership resolver tests for resolve/list behavior and active/inactive statuses in `framework/backend/go/runtime/customerfw/membership_test.go`.
- [X] T030 [P] [US3] Add membership cache TTL and token-expiry tests in `framework/backend/go/runtime/customerfw/membership_cache_test.go`.
- [X] T031 [P] [US3] Add skeleton membership rejection integration tests in `skeleton/backend/go-gin/tests/integration/mini-app/customer_membership_test.go`.

### Implementation

- [X] T032 [US3] Implement `RequireMembership` middleware and authenticated current-customer membership listing contract in `framework/backend/go/runtime/customerfw/membership.go`.
- [X] T033 [US3] Implement optional short-TTL membership cache with token validity cap in `framework/backend/go/runtime/customerfw/membership_cache.go`.
- [X] T034 [US3] Add no-op/local/mock membership resolvers for development and tests in `framework/backend/go/runtime/customerfw/membership_mock.go`.
- [X] T035 [US3] Add skeleton membership resolver adapter for existing local/delegate customer auth paths including current-customer list support in `skeleton/backend/go-gin/internal/services/customer/membership_adapter.go`.
- [X] T036 [US3] Wire membership middleware into mini-app protected routes in `skeleton/backend/go-gin/internal/transport/http/mini-app/routes.go`.

**Checkpoint**: Framework can enforce customer-tenant membership independently of plugin business models.

---

## Phase 6: User Story 4 - 统一 C 端入口解析到租户上下文 (Priority: P2)

**Goal**: Framework supports bootstrap entry resolution and prevents bootstrap/token tenant conflicts.

**Independent Test**: Valid entry hints resolve tenant context; expired/invalid entry fails; bootstrap tenant mismatch with token is rejected.

### Tests

- [X] T037 [P] [US4] Add bootstrap resolver contract tests in `framework/backend/go/runtime/customerfw/bootstrap_test.go`.
- [X] T038 [P] [US4] Add bootstrap-token mismatch tests in `framework/backend/go/runtime/customerfw/middleware_bootstrap_test.go`.
- [X] T039 [P] [US4] Add skeleton bootstrap route adapter tests in `skeleton/backend/go-gin/tests/integration/mini-app/customer_bootstrap_test.go`.

### Implementation

- [X] T040 [US4] Implement bootstrap resolver option and context attachment in `framework/backend/go/runtime/customerfw/bootstrap.go`.
- [X] T041 [US4] Implement bootstrap tenant conflict enforcement in `framework/backend/go/runtime/customerfw/middleware.go`.
- [X] T042 [US4] Add skeleton bootstrap adapter stub for delegated/platform entry resolution in `skeleton/backend/go-gin/internal/services/customer/bootstrap_adapter.go`.
- [X] T043 [US4] Add bootstrap resolve handler wiring for the contract path, backed by the framework bootstrap adapter, in `skeleton/backend/go-gin/internal/transport/http/mini-app/customer_handler.go`.
- [X] T044 [US4] Mirror bootstrap adapter changes in `scaffold/templates/backend/go-gin/internal/services/customer/bootstrap_adapter.go.tmpl`.
- [X] T045 [US4] Mirror bootstrap handler changes in `tools/cli/internal/templates/data/backend/go-gin/internal/transport/http/mini-app/customer_handler.go.tmpl`.

**Checkpoint**: C-end entry hints can provide tenant context without plugin-specific parsing rules.

---

## Phase 7: User Story 5 - 委托 Core 完成 customer 注册、登录和校验 (Priority: P2)

**Goal**: Production customer auth delegates to PowerX Core/platform identity source; local/mock are blocked by default in production except audited break-glass.

**Independent Test**: Delegated source unavailable returns stable 503-style error; production local/mock startup fails unless explicit break-glass is configured and diagnosable.

### Tests

- [X] T046 [P] [US5] Add CustomerAuthClient contract tests covering register/login/validate, STS-backed delegated/platform calls, and third-party source normalization in `framework/backend/go/runtime/customerfw/auth_client_test.go`.
- [X] T047 [P] [US5] Add production local/mock guard tests in `framework/backend/go/runtime/customerfw/source_policy_test.go`.
- [X] T048 [P] [US5] Add skeleton config validation tests for break-glass behavior in `skeleton/backend/go-gin/internal/config/customer_auth_test.go`.
- [X] T049 [P] [US5] Add delegated unavailable, register/login/validate handler regression, and abuse-protection tests in `skeleton/backend/go-gin/tests/integration/mini-app/customer_delegate_framework_test.go`.

### Implementation

- [X] T050 [US5] Implement source policy validator for production, platform, delegated, third_party, local_dev, mock, and break-glass in `framework/backend/go/runtime/customerfw/source_policy.go`.
- [X] T051 [US5] Implement STS-backed delegated/platform/third-party CustomerAuthClient register/login/validate adapter using existing gateway/delegate patterns in `framework/backend/go/runtime/customerfw/delegated_client.go`.
- [X] T052 [US5] Update skeleton `CustomerAuthConfig` with framework source policy fields in `skeleton/backend/go-gin/internal/config/config.go`.
- [X] T053 [US5] Update skeleton customer authenticator factory to implement framework validator/client contracts in `skeleton/backend/go-gin/internal/services/customer/authenticator_factory.go`.
- [X] T054 [US5] Update delegate authenticator STS usage, error mapping, timeout mapping, and third-party source normalization to framework errors in `skeleton/backend/go-gin/internal/services/customer/delegate_authenticator.go`.
- [X] T055 [US5] Wire skeleton register/login/validate handlers to framework CustomerAuthClient contract with public-endpoint abuse protection in `skeleton/backend/go-gin/internal/transport/http/mini-app/customer_handler.go`.
- [X] T056 [US5] Mirror config, delegate adapter, and auth handler changes in `scaffold/templates/backend/go-gin/internal/config/config.go.tmpl` and `tools/cli/internal/templates/data/backend/go-gin/internal/config/config.go.tmpl`.

**Checkpoint**: Production customer auth authority is platform/delegated and unsafe local/mock modes cannot silently run.

---

## Phase 8: User Story 6 - 提供标准测试工具 (Priority: P3)

**Goal**: Plugin tests can use framework helpers for customer context, token, validator, resolver, and middleware failure scenarios.

**Independent Test**: Handler/service tests can simulate authenticated customer, tenant mismatch, membership disabled, and delegate unavailable without a real identity source.

### Tests

- [X] T057 [P] [US6] Add test helper unit tests in `framework/backend/go/runtime/customerfw/testing_test.go`.
- [X] T058 [P] [US6] Add skeleton example tests that use framework mock validator/resolver in `skeleton/backend/go-gin/internal/transport/http/mini-app/customer_framework_helpers_test.go`.

### Implementation

- [X] T059 [US6] Implement `WithCustomerContext`, `TestToken`, `MockCustomerValidator`, and `MockMembershipResolver` helpers in `framework/backend/go/runtime/customerfw/testing.go`.
- [X] T060 [US6] Replace duplicated skeleton test token helpers where practical in `skeleton/backend/go-gin/tests/integration/mini-app/customer_auth_test.go`.
- [X] T061 [US6] Add testing helper documentation to `framework/backend/go/runtime/customerfw/README.md`.
- [X] T062 [US6] Add plugin developer testing notes to `docs/guides/develop/auth/customer.md`.

**Checkpoint**: New plugin tests no longer need to hand-roll customer token/context plumbing.

---

## Final Phase: Polish & Cross-Cutting Concerns

- [X] T063 [P] Update customer auth planning docs with final framework migration status in `docs/plan/023-framework-customer-auth.md`.
- [X] T064 [P] Update scaffold customer auth templates to import framework customer auth contracts in `scaffold/templates/backend/go-gin/internal/transport/http/mini-app/customerhttp/authenticate.go.tmpl`.
- [X] T065 [P] Update CLI embedded customer auth templates to import framework customer auth contracts in `tools/cli/internal/templates/data/backend/go-gin/internal/transport/http/mini-app/customerhttp/authenticate.go.tmpl`.
- [X] T066 Add static boundary check for customer/member IAM semantic drift in `scripts/contracts/validate-customer-auth-boundary.sh`.
- [X] T067 Add CI or make validation hook for `scripts/contracts/validate-customer-auth-boundary.sh` in `package.json`.
- [X] T068 Run `go test ./framework/backend/go/runtime/customerfw ./skeleton/backend/go-gin/internal/... ./skeleton/backend/go-gin/tests/...` and record results in `specs/023-framework-customer-auth/quickstart.md`.
- [X] T069 Run template parity checks for scaffold and CLI embedded templates with `npm test` and record results in `specs/023-framework-customer-auth/quickstart.md`.
- [X] T070 Review logs/diagnostics to ensure no raw token/password/secret is emitted in `framework/backend/go/runtime/customerfw/diagnostics.go`.

---

## Dependencies & Execution Order

### Phase Dependencies

1. Phase 1 Setup has no dependencies.
2. Phase 2 Foundational Framework Contracts blocks all user stories.
3. US1, US2, and US3 are P1 and should be completed before P2 stories.
4. US2 depends on US1 middleware/context foundation.
5. US3 depends on US1 context foundation and US2 tenant resolution.
6. US4 depends on US2 tenant conflict enforcement.
7. US5 depends on Phase 2 contracts and should be integrated before production rollout.
8. US6 depends on finalized contracts from US1-US5.
9. Final Phase depends on all story phases.

### User Story Completion Order

1. US1 - unified customer context
2. US2 - tenant-scoped access enforcement
3. US3 - membership enforcement
4. US4 - bootstrap tenant resolution
5. US5 - delegated/platform auth and production source policy
6. US6 - testing helpers

## Parallel Execution Examples

### US1

```text
T014 middleware tests
T015 skeleton context regression test
T017 skeleton adapter
```

### US2

```text
T021 tenant-required tests
T022 multi-token tests
T024 tenant resolver
```

### US3

```text
T029 membership status tests
T030 cache TTL tests
T034 mock resolver implementation
```

### US4

```text
T037 bootstrap resolver tests
T042 skeleton bootstrap adapter
T044 scaffold bootstrap template
```

### US5

```text
T046 auth client tests
T047 source policy tests
T051 delegated client adapter
```

### US6

```text
T057 helper tests
T059 helper implementation
T062 developer testing docs
```

## Implementation Strategy

### MVP First

Complete Phase 1, Phase 2, US1, US2, and US3. This yields the minimum secure framework customer auth path: unified context, tenant enforcement, and membership enforcement.

### Incremental Delivery

1. Framework contracts and context helpers.
2. Customer auth middleware and skeleton wrapper migration.
3. Tenant mismatch and global-token enforcement.
4. Membership resolver and cache policy.
5. Bootstrap resolver and Core/delegated auth client.
6. Test helper and template parity.

### Validation Criteria

- Every protected C-end path obtains customer context from framework.
- Tenant-scoped paths reject missing tenant and tenant mismatch before business logic.
- Inactive membership states are rejected.
- Production local/mock modes fail unless explicit break-glass is configured.
- Skeleton local/delegate customer auth regressions remain green.
