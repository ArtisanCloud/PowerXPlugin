# Tasks: Framework Realtime Transport

**Input**: Design documents from `/specs/022-framework-realtime-transport/`  
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/realtime-events.schema.yaml`

## Phase 1: Setup

- [x] T001 Confirm current direct realtime usage inventory with `rg "new EventSource|new WebSocket|gin-contrib/sse|body\\.getReader\\("`.
- [x] T002 Add realtime static scan script `scripts/contracts/validate-realtime-transport.sh`.
- [x] T003 Add CI hook for realtime static scan.
- [x] T004 Add realtime contracts validation placeholder for `plugin.d/events.yaml`.

---

## Phase 2: Framework Backend Foundation

- [x] T005 Create `framework/backend/go/runtime/realtime` package with scope, descriptor, envelope and error types.
- [x] T006 Implement topic/channel builder for global/tenant/member scopes.
- [x] T007 Implement descriptor loader/validator against `plugin.d/events.yaml` data shape.
- [x] T008 Implement publish/subscribe permission decision helper.
- [x] T009 Extend or wrap `runtime/ssebus` with standardized error events and optional descriptor validation.
- [x] T010 Add stream-through helper for upstream SSE proxy preserving raw event/data.
- [x] T011 Add backend unit tests for scope builder, descriptor validation and permission decisions.
- [x] T012 Add backend unit tests for SSE stream-through preserving raw event names.

**Checkpoint**: Framework backend can validate and serve managed SSE and stream-through SSE without skeleton-specific code.

---

## Phase 3: Framework Frontend Client Foundation

- [x] T013 Create `framework/frontend/nuxt/framework-client/realtime.ts` facade.
- [x] T014 Extend `sse.ts` with managed EventSource mode lifecycle state.
- [x] T015 Add fetch-based SSE client supporting headers, AbortController and typed event callbacks.
- [x] T016 Align `ws.ts` diagnostics with realtime state fields.
- [x] T017 Add context-change handling that closes old connections when token/tenant/member changes.
- [x] T018 Add frontend unit tests for URL resolution in standalone/host/proxy modes.
- [x] T019 Add frontend unit tests for cleanup/reconnect behavior.

**Checkpoint**: Frontend has one documented entrypoint for WS/SSE clients.

---

## Phase 4: Skeleton Backend Migration

- [x] T020 Migrate `skeleton/backend/go-gin/internal/transport/http/mcp/handler.go` from `gin-contrib/sse` and `internal/mcp/stream` to framework SSE bus.
- [x] T021 Remove direct `github.com/gin-contrib/sse` dependency from `skeleton/backend/go-gin/go.mod` and update sums.
- [x] T022 Migrate `/plugin/agent/stream/sse` proxy to framework stream-through helper.
- [x] T023 Add stable error mapping for Agent stream-through failures.
- [x] T024 Ensure WSBus subscribe path validates declared topics and tenant/member scope.
- [x] T025 Add integration tests for MCP SSE migrated path.
- [x] T026 Add integration tests for Agent SSE stream-through path.

---

## Phase 5: Skeleton Frontend Migration

- [x] T027 Migrate `skeleton/web-admin/nuxt/app/composables/api/useStream.ts` to framework realtime client.
- [x] T028 Migrate `skeleton/web-admin/nuxt/app/pages/capabilities/RegisterForm.vue` MCP stream to framework SSE client.
- [x] T029 Migrate Agent Skill Bridge page to framework fetch-based Agent SSE client.
- [x] T030 Ensure page unload/HMR cleanup closes MCP and Agent stream connections.
- [x] T031 Add or update E2E tests for capability MCP stream.
- [ ] T032 Add or update E2E tests for Agent Chat SSE stream.

---

## Phase 6: Manifest/RBAC Governance

- [x] T033 Extend `plugin.d/events.yaml` with SSE channels and scope metadata.
- [x] T034 Extend manifest check to validate realtime descriptors, protocols, actions and duplicate keys.
- [x] T035 Wire runtime descriptor allowlist into backend publish/subscribe checks.
- [x] T036 Add tests proving undeclared publish/subscribe is rejected.
- [x] T037 Add documentation for topic/channel naming and scope builder usage.

---

## Phase 7: CI Enforcement & Cleanup

- [x] T038 Enforce no direct `new EventSource` in skeleton business code.
- [x] T039 Enforce no direct `new WebSocket` in skeleton business code.
- [x] T040 Enforce no direct `gin-contrib/sse` in skeleton backend.
- [x] T041 Enforce no page-level hand-written SSE reader except framework client internals/tests.
- [x] T042 Update scaffold templates and CLI embedded templates to use framework realtime client/server.
- [x] T043 Run Go tests for framework realtime, ssebus, wsbus, mcp, plugin agent.
- [x] T044 Run Nuxt tests/lint and record unrelated existing failures if any.
- [x] T045 Update quickstart with final verified commands and results.

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → Phase 3 → Phase 4/5 → Phase 6 → Phase 7.
- Backend framework stream-through (T010) blocks Agent backend migration (T022).
- Frontend fetch SSE client (T015) blocks Agent page migration (T029).
- Descriptor validation (T007/T034) blocks runtime enforcement (T035/T036).

## Parallel Opportunities

- T005-T008 can run in parallel with T013-T016.
- MCP migration (T020) and Agent backend stream-through (T022) can run after T009/T010.
- Frontend MCP migration (T028) and Agent page migration (T029) can run after T015.
- CI cleanup tasks T038-T041 can be implemented after migrations are complete.
