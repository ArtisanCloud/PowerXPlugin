# Data Model: Framework Customer Identity/Auth

## Authority Boundary

PowerXPlugin framework does not own the production customer database schema.
Production customer identity, auth identities, tenant memberships, shared app
entries, sessions, and login audit records are owned by PowerX Core.

Framework data models in this spec are runtime contracts only:

- `CustomerContext` maps from PowerX Core `customer_accounts` plus the current token/membership decision.
- `CustomerMembership` maps from PowerX Core `customer_tenant_memberships`.
- `BootstrapContext` maps from PowerX Core `mini_app_entries`.
- `CustomerAuthResult` maps from PowerX Core customer auth/session issuance.

Plugin local mode may persist a local mirror for development/debugging, but that
mirror must remain compatible with the PowerX Core customer schema and must not
be treated as a production source of truth. Parent, player, guardian, athlete,
student, patient, fan, membership-benefit, training-profile, and similar
industry concepts belong to business plugins, not to customerfw. The framework
contract does include the generic PowerX customer display attributes carried by
PowerX Core `customer_accounts`.

## CustomerContext

Runtime C 端身份上下文，注入到受保护请求并供 handler/service 读取。

- `tenant_uuid` (string, optional until tenant-scoped access; required before tenant-scoped business logic)
- `customer_uuid` (string, required)
- `membership_uuid` (string, optional for global-only context; required after membership resolution)
- `profile` (CustomerAttributes, optional): basic display attributes mirrored from PowerX Core `customer_accounts`
- `roles` (array<string>, customer-side roles only)
- `scopes` (array<string>, customer-side scopes only)
- `source` (string, required): `platform`, `delegated`, `third_party`, `local_dev`, `mock`
- `authenticated` (bool, required)
- `attributes` (object, optional, generic claims only)
- `raw_claims` (object, optional, sanitized; must not include raw token or secrets)
- `trace_id` (string, optional)

### Validation Rules

- `customer_uuid` is required for authenticated contexts.
- `tenant_uuid` is required before tenant-scoped data access.
- `membership_uuid` is required after membership middleware succeeds.
- `roles` and `scopes` are customer-side only and must not reuse后台 member IAM permission semantics.
- `profile` must contain only generic display attributes, not SCRM or industry fields.
- `attributes` and `raw_claims` must not contain industry business model fields or secrets.

## CustomerAttributes

Generic customer display attributes that are safe to expose through runtime
identity context. These fields are part of the PowerX Core customer base CRM
shape, not plugin industry models.

- `display_name` (string, optional): preferred display name for admin lists, details, and C-end greetings.
- `nickname` (string, optional): social or channel nickname.
- `given_name` (string, optional): personal/given name.
- `family_name` (string, optional): family/surname.
- `avatar_url` (string, optional): public avatar URL.
- `locale` (string, optional): preferred locale.
- `timezone` (string, optional): preferred timezone.

Recommended display fallback:

```text
display_name -> nickname -> family_name + given_name -> email -> phone -> customer_uuid
```

## CustomerMembership

Relationship between a customer and a tenant.

- `tenant_uuid` (string, required)
- `customer_uuid` (string, required)
- `membership_uuid` (string, required)
- `status` (string, required): `active`, `pending`, `suspended`, `disabled`, `deleted`
- `roles` (array<string>, optional)
- `scopes` (array<string>, optional)
- `source` (string, required): platform/delegated/local/mock source that produced the decision
- `expires_at` (string, optional, RFC3339)
- `policy_version` (string, optional)

### State Transitions

```text
pending -> active
active -> suspended
active -> disabled
suspended -> active
suspended -> disabled
disabled -> active（仅平台权威源允许）
active|pending|suspended|disabled -> deleted
```

### Validation Rules

- Only `active` allows tenant-scoped protected access.
- `pending`, `suspended`, `disabled`, and `deleted` must be rejected.
- Production membership authority is platform/delegated source, even if framework caches the decision.
- Cached membership lifetime must not exceed token validity.

## CustomerTokenValidationResult

Normalized result of validating a customer token.

- `valid` (bool, required)
- `customer_context` (CustomerContext, optional when invalid)
- `token_expires_at` (string, optional, RFC3339)
- `token_tenant_uuid` (string, optional)
- `source` (string, required)
- `error_code` (string, optional)
- `error_message` (string, optional)

### Validation Rules

- Invalid, expired, malformed, or unauthenticated token results must not produce authenticated context.
- Globally scoped token may omit `token_tenant_uuid`.
- If multiple tokens are supplied, all valid results must resolve to the same customer and compatible tenant context.

## BootstrapContext

Tenant context resolved from C 端入口 hints.

- `tenant_uuid` (string, required when resolution succeeds)
- `org_uuid` (string, optional)
- `entry_type` (string, required): `scene`, `invite_code`, `org_code`, `tenant_hint`, `direct`
- `campaign` (string, optional)
- `channel` (string, optional)
- `expires_at` (string, optional, RFC3339)
- `metadata` (object, optional; generic entry metadata only)

### Validation Rules

- Expired or invalid bootstrap entries must fail without creating customer login state.
- Bootstrap tenant must match token/request tenant when both are present.
- Bootstrap metadata must not become plugin business profile data.

## CustomerAuthResult

Result of customer registration or login through platform/delegated identity source.

- `access_token` (string, required for successful login)
- `refresh_token` (string, optional)
- `expires_in` (integer, required, seconds)
- `customer_context` (CustomerContext, required)
- `available_memberships` (array<CustomerMembership>, optional when tenant selection is required)
- `trace_id` (string, optional)

### Validation Rules

- Tokens and refresh tokens are sensitive and must not be logged.
- If login maps to multiple memberships and no tenant is selected, result must require tenant selection instead of auto-picking.

## CustomerAuthError

Stable error outcome for customer auth failures.

- `code` (string, required)
- `message` (string, required)
- `details` (object, optional, sanitized)
- `trace_id` (string, optional)

### Error Codes

- `CUSTOMER_TOKEN_MISSING`
- `CUSTOMER_TOKEN_INVALID`
- `CUSTOMER_UNAUTHENTICATED`
- `CUSTOMER_TENANT_MISMATCH`
- `CUSTOMER_TENANT_REQUIRED`
- `CUSTOMER_MEMBERSHIP_REQUIRED`
- `CUSTOMER_MEMBERSHIP_DISABLED`
- `CUSTOMER_DELEGATE_UNAVAILABLE`
- `CUSTOMER_BOOTSTRAP_FAILED`
- `CUSTOMER_CONTEXT_MISSING`
- `CUSTOMER_IDENTITY_SOURCE_FORBIDDEN`

## CustomerAuthSource

Identity source mode used for diagnostics and startup validation.

- `mode` (string): `platform`, `delegated`, `third_party`, `local_dev`, `mock`
- `production_allowed` (bool)
- `break_glass` (bool)
- `diagnostic_label` (string)

### Validation Rules

- `local_dev` and `mock` are production-forbidden by default.
- Production break-glass mode must be explicit, audited, and visible in diagnostics.

## Relationships

- `CustomerTokenValidationResult` may produce a `CustomerContext`.
- `BootstrapContext` may provide the current `tenant_uuid` for a globally scoped customer token.
- `CustomerMembership` completes `CustomerContext` for tenant-scoped access.
- `CustomerAuthResult` contains the issued credentials and normalized `CustomerContext`.
- `CustomerAuthError` is returned for token, tenant, membership, bootstrap, source, or context failures.
