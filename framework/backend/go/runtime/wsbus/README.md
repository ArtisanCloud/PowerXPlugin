
## Delegated realtime service

`NewRealtimeService(gatewayClient)` uses the fixed `realtime.service.v1` Core
capabilities. `NewRemoteHub(ctx, service, namespace)` implements `LocalHub.Publish`
through that service and `ScopedSubscriber` for authorized return subscriptions.
It never mirrors a Host publication to an in-memory Hub or connects plugin Redis.
`SubscribeScoped` selects this declared scoped interface, or the existing local
Hub subscription in explicit Local mode.

Core derives the tenant from the API Key or STS service credential, never from
caller identity headers or URL parameters. `StreamInput.TenantUUID` is optional
caller context validated before credential acquisition; neither public streaming
client sends it as an HTTP header or query parameter. Realtime requests retain
`preferred_protocol=core_internal` and the `realtime.service.v1` body, including
`tenant_key` (an equality assertion against the credential tenant), namespace and
cursor. Event tenant validation remains mandatory. Other streaming capabilities
retain their own protocol selection, including the default `rest` protocol.

A tenant feed starts only after an authorized subscriber supplies its trusted
scope. Each replica captures its own cursor before subscribing, consumes retained
frames serially, suppresses duplicate/out-of-order cursors and remembers at most
2048 event IDs. Limits: 32 active tenant feeds, 512 subscriptions per feed. The
last unsubscribe cancels its upstream; `Close` cancels and waits for workers.
Transient stream failures retry three times at 100/200/400 ms with the same cursor.
Auth/contract/expired-cursor failures stop and notify subscribers. Lease reconnect
is a declared stream operation, not a polling or transport fallback.

All replicas must use the same service identity and namespace for fanout. API Key
and STS subjects are deliberately distinct. Payload bytes use canonical Base64
through the Gateway and `json.Number` inside the Hub to preserve integer IDs.
User/member/topic authorization remains the plugin consumer's responsibility;
`MemberUUID` is a destination constraint and must be checked before delivery.
A publish ACK is not a browser delivery or business completion acknowledgement.

Core retains at most 2048 entries and expires an idle stream after one hour.
Expired history is explicit failure; consumers must expose it and read current
persisted state after a fresh authorized subscription. They must not re-run
business tasks or claim historical progress was replayed beyond that window.

## Tenant transport regression

From `framework/backend/go`, run:

```sh
GOWORK=off go test -race ./gateway ./internal/integration/gateway ./runtime/powerx/hostcontract ./runtime/powerx/capability ./runtime/wsbus
```

With a matching Core checkout, run from the repository root:

```sh
python3 scripts/contracts/verify-realtime-stream-core.py /path/to/PowerX/backend
```

The Core check uses temporary module/overlay files, synthetic API Keys, in-memory
SQLite and miniredis. Both SDK clients call Core's authentication middleware and
real realtime invoker without altering request headers. It covers ready, invalid
credentials, cross-tenant body assertions, missing grants, live revocation and
cancellation. It does not validate STS issuance, a published SDK or an installed
plugin/browser. Core currently maps the typed cross-tenant
`REALTIME_INVALID_ARGUMENT` rejection (400) to HTTP 502; the check separately
verifies the typed rejection and that HTTP fails without delivering frames.
