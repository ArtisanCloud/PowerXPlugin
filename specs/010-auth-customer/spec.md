# Feature Specification: Customer Auth Modes

**Feature Branch**: `010-auth-customer`  
**Created**: 2025-12-29  
**Status**: Draft  
**Input**: User description: "Implement Customer authentication per docs/plan/010-auth-customer and docs/guides/develop/auth/customer.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Customer access to protected mini-app APIs (Priority: P1)

As a mini-app customer, I can call protected endpoints (e.g., browse or submit orders) only after presenting a valid token that matches my tenant, so that my personal data stays isolated and unauthorized tenants cannot impersonate me.

**Why this priority**: This is the fundamental reason to build Customer auth; without it, the plugin cannot safely expose any 2C endpoints.

**Independent Test**: Spin up the mini-app router with Customer middleware, send requests with/without valid tokens across different tenants, and verify that only correct combinations succeed while others receive `401/403`.

**Acceptance Scenarios**:

1. **Given** a request contains `Authorization: Bearer <valid customer token>` for tenant T1, **When** it hits `/mini-app/*` endpoints, **Then** the middleware injects `CustomerContext` and the handler processes the request successfully.
2. **Given** a token issued for tenant T1, **When** it is used against tenant T2’s endpoint, **Then** the middleware rejects it with an unauthorized response and logs the mismatch.

---

### User Story 2 - Skeleton mode login & token issuance (Priority: P2)

As a plugin operator running in standalone/Skeleton mode, I can offer `/mini-app/auth/register` and `/mini-app/auth/login` so customers can create accounts, obtain JWTs, and reuse those JWTs for subsequent mini-app requests.

**Why this priority**: Enables local development, demos, and deployments where the plugin is not fronted by PowerX; once implemented it also provides a fallback credential flow.

**Independent Test**: Use API tests to register a customer, log in, receive a JWT, and consume a protected mini-app API with that JWT—all without touching Delegated flows.

**Acceptance Scenarios**:

1. **Given** a new tenant in Skeleton mode, **When** a customer registers with unique email/phone + password, **Then** the system creates a tenant-scoped record and returns a confirmation.
2. **Given** a registered customer with correct credentials, **When** they call `/mini-app/auth/login`, **Then** the response includes a JWT containing `tenant_uuid` and `customer_uuid` plus expiry metadata.

---

### User Story 3 - Delegated mode validation (Priority: P3)

As a plugin operator deployed behind PowerX, I can configure the plugin to call the host CRM/IAM validation endpoint so customer tokens issued by the host are accepted without duplicating customer data.

**Why this priority**: Ensures production deployments integrate with host identity, reducing duplication and aligning with PowerX security requirements.

**Independent Test**: Simulate host-issued tokens, stub the delegate validation endpoint, and verify the plugin accepts valid responses, rejects invalid ones, and never persists customer credentials locally.

**Acceptance Scenarios**:

1. **Given** Delegated mode is enabled with a reachable host validation endpoint, **When** a request presents a host-issued token, **Then** the authenticator forwards it, validates the response, and injects claims without creating local customer records.
2. **Given** the host endpoint returns errors or mismatched tenants, **When** the plugin forwards the token, **Then** it responds with unauthorized and surfaces diagnostic info for operators.

---

### Edge Cases

- Concurrent logins or replays: what happens if a customer reuses an expired or revoked token?
- Tenant mismatch: how does the system respond when a valid host token references a tenant the plugin instance does not serve?
- Missing headers: how are requests handled when `Authorization` is missing or the token is malformed?
- Mode misconfiguration: what error does the system report if Delegated mode is enabled but the host endpoint is unreachable?
- Account lifecycle: how does Skeleton mode behave when a customer is disabled or soft-deleted but still holds a valid token?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Mini-app HTTP groups MUST enforce `httpmw.EnsureTenant()` before invoking Customer authentication.
- **FR-002**: System MUST provide a configurable `CustomerAuthenticator` factory that instantiates either Local (Skeleton) or Delegated implementations based on runtime configuration.
- **FR-003**: Customer middleware MUST extract tokens from headers (`Authorization` or `X-Customer-Token`), validate them, reject invalid/mismatched tenants, and attach `CustomerContext` to request scopes.
- **FR-004**: Skeleton mode MUST expose `/mini-app/auth/register` and `/mini-app/auth/login` REST endpoints that operate against tenant-scoped `customer.Customer` records, enforce secure password hashing, and issue JWTs with expiry metadata.
- **FR-005**: Delegated mode MUST forward incoming tokens to the configured host validation endpoint, map host claims into `CustomerContext`, and avoid persisting host-managed customers in plugin storage.
- **FR-006**: Configuration MUST allow switching modes at runtime via environment/config files (e.g., `customerAuth.mode=local|delegate`) and define delegate endpoint, JWT issuer/audience, and secrets.
- **FR-007**: All customer-authenticated responses MUST leverage `contracts.Response*` wrappers, including clear unauthorized messages when validation fails.
- **FR-008**: Services and handlers MUST be able to retrieve `CustomerContext` (customer UUID, tenant UUID, roles, metadata) via helper functions without re-reading headers.
- **FR-009**: The system MUST log structured audit entries for key lifecycle events (register, login, token validation failures) with tenant and customer identifiers.
- **FR-010**: Automated tests MUST cover Skeleton registration/login flows, Delegated validation success/failure, tenant mismatch handling, and middleware injection behaviour.

### Key Entities *(include if feature involves data)*

- **Customer Profile**: Represents a tenant-scoped customer identity (UUID, contact info, status, password hash for Skeleton mode).
- **Customer Token**: JWT or host-issued token that encodes tenant UUID, customer UUID, expiry, and optional roles/attributes.
- **CustomerContext**: Runtime structure added to request contexts containing tenant UUID, customer UUID, roles, and source mode (Skeleton/Delegated) for downstream services.
- **CustomerAuthConfig**: Configuration blob describing mode, endpoints, issuers, secrets, and optional cache settings.

### Assumptions

- PowerX host exposes a synchronous validation endpoint returning tenant and customer identifiers plus token freshness indicators.
- Skeleton mode databases already include `customer.Customer` (or equivalent) schema with tenant UUID column and soft-delete semantics.
- Mini-app APIs already exist and simply need the middleware + context enforcement; no new business flows are introduced beyond authentication.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of requests to `/mini-app/*` endpoints are blocked unless both tenant UUID and customer token are present and validated.
- **SC-002**: In Skeleton mode, a customer can register, log in, and obtain a usable token in under 10 seconds end-to-end under nominal load.
- **SC-003**: Delegated mode validation completes within 500 ms for 95% of requests when the host endpoint responds normally (excluding network faults), ensuring mini-app UX is not degraded.
- **SC-004**: Less than 1% of authentication failures are due to plugin misconfiguration after rollout, as measured by structured logs and alerting.
- **SC-005**: Automated test suite covers at least one happy-path and one failure-path scenario for each mode (Skeleton register/login, Skeleton tenant mismatch, Delegated success, Delegated tenant mismatch, missing token).
