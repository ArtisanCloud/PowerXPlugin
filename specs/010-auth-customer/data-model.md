# Data Model — Customer Auth Modes

## CustomerProfile
- **tenant_uuid** (UUID, required): owner tenant; immutable; enforced via RLS.
- **customer_uuid** (UUID, required): primary identifier exposed to mini-app.
- **contact** (object): `email`, `phone`, both optional but at least one must be present and unique per tenant.
- **password_hash** (string, Skeleton-only): bcrypt hash; nullable when Delegated.
- **status** (enum): `active`, `disabled`, `deleted`; determines login eligibility.
- **metadata** (jsonb): channel tags, locale, roles.
- **timestamps**: created_at, updated_at, deleted_at (soft delete).

### Validation & State
- Unique constraint `(tenant_uuid, email)` / `(tenant_uuid, phone)` when present.
- Only `active` customers receive tokens; `disabled` returns 403, `deleted` treated as not found.

## CustomerToken (logical entity)
- **tenant_uuid**: copied from profile or host claims.
- **customer_uuid**: subject identifier.
- **roles** (array): optional; used for downstream authorization.
- **issued_at / expires_at**: used to enforce success criteria.
- **mode** (enum): `local`, `delegate`; traced for observability.

### Rules
- Skeleton mode tokens signed with plugin issuer/secret; exp ≤ 2h.
- Delegated tokens are opaque but validator must return same fields; plugin never persists them.

## CustomerContext (runtime)
- **tenant_uuid**
- **customer_uuid**
- **roles**
- **source_mode**
- **attributes** (map)

Stored in Gin context + request context for service access.

## CustomerAuthConfig
- **mode** (`local`|`delegate`, required)
- **delegate_endpoint** (URL, required when mode=`delegate`)
- **delegate_timeout_ms**, **retry_policy**
- **jwt_issuer**, **jwt_audience**, **jwt_secret** (local mode)
- **cache_ttl_seconds** (optional)

Validation:
- Reject startup if required fields missing for chosen mode.
- TTL must be ≤ token expiration; default 0 (disabled).
