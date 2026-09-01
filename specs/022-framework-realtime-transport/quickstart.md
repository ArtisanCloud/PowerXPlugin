# Quickstart: Framework Realtime Transport

## 1. Static Drift Check

```bash
npm run check:realtime-transport
```

Expected: descriptor keys, protocol/action/scope values and duplicates are valid; skeleton business code has no direct browser transport primitives or `gin-contrib/sse` imports.

## 2. Backend Tests

```bash
cd framework/backend/go
go test ./runtime/realtime ./runtime/ssebus ./runtime/wsbus

cd ../../../skeleton/backend/go-gin
go test ./internal/transport/http/mcp ./internal/transport/http/plugin/agent ./internal/transport/http/wsbus ./internal/transport/http/admin/runtime_ops
```

## 3. Frontend Tests

```bash
cd skeleton/web-admin/nuxt
npm run test:unit -- framework-realtime.spec.ts
```

If existing unrelated tests fail, record them in the implementation report and keep realtime-specific tests isolated.

Current baseline: `framework-realtime.spec.ts` passes. `npm run lint` exits successfully
with the repository's pending-lint-configuration notice. The full Nuxt suite has an
unrelated existing failure in `tests/unit/useAuth.fallback.spec.ts`: Vitest does not
resolve the Nuxt `#app` alias imported by `app/utils/tenant-context.ts`.

## 4. Standalone Manual Validation

1. Start plugin backend on `8078` with `go run ./cmd/plugin`.
2. Start Nuxt admin.
3. Register an MCP session through the authenticated runtime session endpoint.
4. Open `/api/v1/mcp/sse?session_id=<session_uuid>` with the same tenant-scoped Bearer token.
5. ACK the session and confirm the stream receives `event: session.ready`.
6. Close the session after the probe, and remove only its explicitly identified test records if the probe used development storage.
7. Open Agent Skill Bridge and send a prompt that triggers PowerX Agent SSE.
8. Confirm the page receives `start/token/final/end` without direct page-level SSE parsing.

## 5. Host/Proxy Manual Validation

1. Open the plugin through PowerX host `/_p/{plugin_id}`.
2. Subscribe to a declared WS topic and an SSE channel.
3. Confirm diagnostics show URL source as `host` or `proxy`.
4. Switch tenant/member context.
5. Confirm old connections close and new connections use the new scope.

## 6. Manifest/RBAC Validation

1. Declare every WS topic or SSE channel in `skeleton/plugin.d/events.yaml` with `protocols`, `actions`, and `scope`.
2. Run `npm run check:realtime-transport`.
3. Confirm an invalid protocol/action/scope or duplicate key fails the check.
4. Confirm a declared topic/channel is accepted by the framework descriptor loader.

### Topic naming and scope rules

- Keys must start with `_topic.`, `_channel.`, or `powerx.`. Old free-form topic aliases are not supported.
- A descriptor is deny-by-default for protocol, action, event type, and scope. Publishers should use `realtime.NewAuthorizedWSPublisher`; subscribers use `realtime.Decide` before registering with the bus.
- A tenant-scoped dynamic key must use the exact placeholder `{{tenant_uuid}}`, for example `_topic.notify.tenant.{{tenant_uuid}}`. A member-scoped key may additionally use `{{member_uuid}}`.
- Placeholders are substituted only from the validated runtime scope. They are not wildcards: a request for another tenant/member key is rejected.

Current skeleton examples:

- `_topic.notify.tenant.{{tenant_uuid}}` for tenant notifications;
- `_topic.iam.<provider>.sync.progress` for IAM channel sync progress;
- `_channel.mcp.session` for MCP SSE/WS session lifecycle.

## 7. Agent SSE Probe

Use PowerX Core probe endpoint through plugin backend:

```bash
curl -N "http://127.0.0.1:8078/api/v1/plugin/agent/stream/sse?agent_id=<agent>&session_id=<session>&trace_id=<trace>&q=hello"
```

Expected:

- HTTP 200 for valid configuration.
- Raw Agent SSE event names preserved.
- Errors mapped to stable framework error events with trace/request IDs.

## 8. Verified commands

The backend contract checks for this feature are:

```bash
cd framework/backend/go
go test ./runtime/realtime/...

cd ../../../skeleton/backend/go-gin
go test ./cmd/plugin ./internal/bootstrap ./internal/transport/http/mcp ./internal/transport/http/admin/runtime_ops ./internal/transport/http/admin/iam ./internal/transport/http/wsbus ./tests/integration

cd ../../..
npm run check:realtime-transport
```
