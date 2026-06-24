# Quickstart: Framework Realtime Transport

## 1. Static Drift Check

```bash
rg -n "new EventSource|new WebSocket|gin-contrib/sse|body\\.getReader\\(" \
  skeleton framework \
  -g '!**/*test*' -g '!framework/frontend/nuxt/framework-client/**'
```

Expected: no business-code violations after migration. Framework client internals and tests may contain protocol primitives.

## 2. Backend Tests

```bash
go test ./framework/backend/go/runtime/ssebus ./framework/backend/go/runtime/wsbus
go test ./skeleton/backend/go-gin/internal/transport/http/mcp ./skeleton/backend/go-gin/internal/transport/http/plugin/agent ./skeleton/backend/go-gin/internal/transport/http/wsbus
```

## 3. Frontend Tests

```bash
npm --prefix skeleton/web-admin/nuxt run test
npm --prefix skeleton/web-admin/nuxt run lint
```

If existing unrelated tests fail, record them in the implementation report and keep realtime-specific tests isolated.

## 4. Standalone Manual Validation

1. Start plugin backend on `8078`.
2. Start Nuxt admin.
3. Open Agent Skill Bridge.
4. Send a prompt that triggers PowerX Agent SSE.
5. Confirm the page receives `start/token/final/end` without direct page-level SSE parsing.

## 5. Host/Proxy Manual Validation

1. Open the plugin through PowerX host `/_p/{plugin_id}`.
2. Subscribe to a declared WS topic and an SSE channel.
3. Confirm diagnostics show URL source as `host` or `proxy`.
4. Switch tenant/member context.
5. Confirm old connections close and new connections use the new scope.

## 6. Manifest/RBAC Validation

1. Add a temporary undeclared topic in a test publish path.
2. Run manifest alignment checks.
3. Confirm CI/local check fails.
4. Remove the temporary topic and confirm checks pass.

## 7. Agent SSE Probe

Use PowerX Core probe endpoint through plugin backend:

```bash
curl -N "http://127.0.0.1:8078/api/v1/plugin/agent/stream/sse?agent_id=<agent>&session_id=<session>&trace_id=<trace>&q=hello"
```

Expected:

- HTTP 200 for valid configuration.
- Raw Agent SSE event names preserved.
- Errors mapped to stable framework error events with trace/request IDs.
