# Framework Customer Identity/Auth

`customerfw` provides generic C-end external customer identity and authorization primitives for plugin mini-app routes.

The package is intentionally limited to:

- validating customer tokens
- attaching a normalized `CustomerContext`
- resolving tenant context for customer requests
- enforcing customer-to-tenant membership
- defining bootstrap and delegated auth client contracts
- providing diagnostics fields that avoid raw token or secret values

It does not model SCRM or industry concepts such as customer profiles, tags, owners, follow-ups, timelines, players, guardians, learners, patients, fans, benefits, training plans, or reports. Those remain plugin domain models and should be exposed through plugin capabilities such as SCRM when other plugins need them.

It does carry generic PowerX Core customer display attributes through `CustomerContext.Profile`: `display_name`, `nickname`, `given_name`, `family_name`, `avatar_url`, `locale`, and `timezone`. Those fields are the base customer identity shape, not an SCRM or industry domain model.

Boundary rule:

- Framework IAM identifies back-office members/employees.
- `customerfw` identifies C-end external customers and their tenant membership.
- SCRM plugins model customer business data, tags, lifecycle, follow-up records, and member-customer business relationships.
- Other plugins call SCRM capabilities for customer profiles, tags, owners, follow-ups, lifecycle, leads, or timelines.

Typical usage:

```go
protected := group.Group(
	"",
	customerfw.Authenticate(validator, customerfw.RequireTenant()),
	customerfw.RequireMembership(resolver),
	tenantfw.EnsureTenant(),
)
```

Handlers should read only the framework context:

```go
cc := customerfw.MustContextFromGin(c)
tenantUUID := cc.TenantUUID
customerUUID := cc.CustomerUUID
```

Testing helpers:

```go
validator := customerfw.NewMockCustomerValidator(&customerfw.CustomerContext{
	TenantUUID:    "tenant-a",
	CustomerUUID:  "customer-a",
	Authenticated: true,
})
token := customerfw.TestToken("customer-a", "tenant-a")
ctx := customerfw.WithCustomerContext(context.Background(), &customerfw.CustomerContext{
	TenantUUID:    "tenant-a",
	CustomerUUID:  "customer-a",
	Authenticated: true,
})
```

Developer-facing docs:

- `docs/guides/develop/auth/customer.md`
- `docs/contracts/customer-auth.openapi.yaml`
