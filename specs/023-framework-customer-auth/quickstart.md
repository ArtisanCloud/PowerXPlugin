# Quickstart: Framework Customer Identity/Auth

## 1. Verify Feature Branch

```bash
git branch --show-current
```

Expected:

```text
023-framework-customer-auth
```

## 2. Run Current Customer Auth Regression Tests

Use existing skeleton tests as the compatibility baseline before migration:

```bash
go test ./skeleton/backend/go-gin/tests/integration/mini-app ./skeleton/backend/go-gin/tests/unit
```

Expected:

- Existing local customer auth tests pass.
- Existing delegated customer auth tests pass.
- Tenant mismatch behavior remains unchanged unless the new global-token requirement explicitly changes the test expectation.

Recorded result:

```text
go test ./skeleton/backend/go-gin/tests/integration/mini-app ./skeleton/backend/go-gin/tests/unit
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/tests/integration/mini-app 1.540s
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/tests/unit 0.537s
```

## 3. Framework Package Tests

After implementation, run framework customer auth tests:

```bash
go test ./framework/backend/go/runtime/customerfw
```

Expected coverage:

- customer context setter/getter
- token validator mock/local/delegated adapters
- multiple token conflict rejection
- tenant required for tenant-scoped routes
- membership active/disabled/missing handling
- production local/mock startup rejection

Recorded result:

```text
go test ./runtime/customerfw
ok github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw 0.416s
```

## 4. Skeleton Migration Tests

After skeleton adapters are migrated to framework contracts:

```bash
go test ./skeleton/backend/go-gin/internal/middleware ./skeleton/backend/go-gin/internal/transport/http/mini-app ./skeleton/backend/go-gin/internal/services/customer
```

Expected:

- protected mini-app routes read framework customer context
- service code does not re-parse raw token
- local/dev customer auth remains available outside production
- delegated unavailable returns stable service-unavailable error

Recorded result:

```text
go test ./tests/integration/mini-app ./tests/unit
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/tests/integration/mini-app 1.181s
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/tests/unit 0.895s
```

## 5. Security Boundary Checks

Run static checks to catch forbidden model and logging drift:

```bash
rg -n "log\\.(Info|Warn|Error|Debug)|slog\\.|logrus\\." \
  framework/backend/go/runtime/customerfw skeleton/backend/go-gin/internal \
  -g '!**/*test*' | rg -n "token|password|secret|refresh_token|access_token"
```

Expected:

- no new `tenant_id` usage
- no log statements containing raw token/password/secret fields
- no customer context treated as member IAM context

## 6. Manual Tenant/Membership Validation

1. Configure delegated/platform customer auth.
2. Start plugin backend in standalone mode.
3. Send protected C-end request with global customer token but no tenant hint.
4. Confirm response asks for tenant context.
5. Repeat with a tenant where membership is active.
6. Confirm request succeeds and context contains `tenant_uuid`, `customer_uuid`, and `membership_uuid`.
7. Repeat with a disabled membership.
8. Confirm request is rejected before business handler executes.

## 7. Production Source Guard

Start backend with production mode and local/mock identity source.

Expected:

- startup fails by default with `CUSTOMER_IDENTITY_SOURCE_FORBIDDEN`
- startup succeeds only when explicit break-glass is configured
- break-glass mode is visible in audit or diagnostics output

## 8. Multi-Token Conflict Probe

Send a protected request containing two customer token credentials that resolve to different customers or tenants.

Expected:

- request is rejected before business logic executes
- error is stable and includes trace/request identifier when available

## 9. Final Implementation Validation

Customer auth focused tests:

```text
cd framework/backend/go && go test ./runtime/customerfw
ok github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw 1.086s

cd skeleton/backend/go-gin && go test ./internal/config ./internal/transport/http/mini-app ./tests/integration/mini-app ./tests/unit
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config 0.404s
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/mini-app
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/tests/integration/mini-app
ok github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/tests/unit

bash scripts/contracts/validate-customer-auth-boundary.sh
customer auth boundary check passed
```

Full requested Go sweep was run:

```text
go test ./framework/backend/go/runtime/customerfw ./skeleton/backend/go-gin/internal/... ./skeleton/backend/go-gin/tests/...
```

Result: failed in pre-existing non-customer-auth packages:

- `skeleton/backend/go-gin/internal/transport/http/plugin/agent`: gateway test fakes do not implement newly required `RegisterCatalog`.
- `skeleton/backend/go-gin/internal/transport/http/integration`: gateway test fake does not implement `ArchiveAgentSession`.
- `skeleton/backend/go-gin/tests/integration`: scheduler test still uses removed `config.GatewayConfig.TenantUUID`.

Template parity command was run:

```text
npm test
```

Result: failed before customer auth boundary check because `scripts/capabilities/run-from-package.mjs` reports `mock is not defined`. The customer auth boundary check passes when run directly.
