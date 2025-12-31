# Tasks: Framework TaskBus Event Bridge

**Input**: Design documents from `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/`  
**Prerequisites**: `specs/008-framework-task-bus/plan.md`, `specs/008-framework-task-bus/spec.md`, `specs/008-framework-task-bus/research.md`, `specs/008-framework-task-bus/data-model.md`, `specs/008-framework-task-bus/contracts/channel-events.yaml`

## Phase 1: Setup (Shared Infrastructure)

- [ ] T001 Confirm baseline contracts exist in `specs/008-framework-task-bus/contracts/channel-events.yaml`
- [ ] T002 [P] Add a CI check hook placeholder for contracts validation in `.github/workflows/ci.yml`
- [ ] T003 [P] Add a local validation entrypoint for contracts validation in `Makefile` or `make-files/test.mk`

---

## Phase 2: Foundational (Blocking Prerequisites)

- [ ] T004 Define `event_bridge` config shape and defaults in `skeleton/backend/internal/config/config.go`
- [ ] T005 Implement `Event` meta builder helper (tenant_uuid/request_id/source_plugin/trace_id/occurred_at/payload_version) in `skeleton/backend/internal/domain/event/meta.go`
- [ ] T006 Implement core event types (`Event`, `Topic`, `Subscription` DTOs) in `skeleton/backend/internal/domain/event/models.go`
- [ ] T007 Implement an emitter interface and factory (local vs taskbus) in `skeleton/backend/internal/services/event_bridge/emitter.go`
- [ ] T008 Implement a local emitter (log/in-memory) with fallback semantics in `skeleton/backend/internal/services/event_bridge/local_emitter.go`
- [ ] T009 Implement a contracts validator (topic unique + required meta keys) in `tools/contracts/validate-taskbus-contracts.go`
- [ ] T010 Wire the contracts validator into CI (run on PR) in `.github/workflows/ci.yml`

**Checkpoint**: Foundational ready — Emitter API exists; local emitter works; contracts validator enforced.

---

## Phase 3: User Story 1 — 业务侧统一发事件 (Priority: P1) 🎯 MVP

**Goal**: 业务层通过统一 Emitter 发出 Channel 事件（不改业务对外 API）。

**Independent Test**: 在本地（TaskBus 关闭）触发一次示例业务动作（或直接调用 service），确认事件被发出且包含必填 meta 字段。

- [ ] T011 [P] [US1] Add unit tests for meta builder in `skeleton/backend/tests/unit/event_meta_test.go`
- [ ] T012 [P] [US1] Add unit tests for local emitter fallback (no panic + logs) in `skeleton/backend/tests/unit/event_local_emitter_test.go`
- [ ] T013 [US1] Create a channel event emitter adapter implementing `ChannelEventEmitter` in `skeleton/backend/internal/observability/channel/event_emitter.go`
- [ ] T014 [US1] Update one representative path to emit `powerx.channel.master.credential_inspection.v1` in `skeleton/backend/internal/jobs/channel/master/*`
- [ ] T015 [US1] Ensure emitted events include `topic + tenant_uuid + trace_id` idempotency key material in `skeleton/backend/internal/observability/channel/event_emitter.go`

---

## Phase 4: User Story 2 — TaskBus 接入与双实现切换 (Priority: P2)

**Goal**: 通过 `event_bridge.enabled` 切换 local/TaskBus，并在 TaskBus 不可用时自动降级到 local。

**Independent Test**: 在开启 TaskBus 模式时注入一个“模拟不可用”的 TaskBus emitter，验证会自动降级到 local 且打点/告警。

- [ ] T016 [US2] Implement taskbus emitter adapter skeleton (interface only; no external deps) in `skeleton/backend/internal/services/event_bridge/taskbus_emitter.go`
- [ ] T017 [US2] Implement fallback logic: taskbus failure → local emitter in `skeleton/backend/internal/services/event_bridge/emitter.go`
- [ ] T018 [P] [US2] Add integration test for fallback behavior in `skeleton/backend/tests/integration/event_bridge_fallback_test.go`
- [ ] T019 [US2] Wire emitter into `app.Deps` and service construction in `skeleton/backend/internal/shared/app/deps.go`
- [ ] T020 [US2] Document config + fallback in `specs/008-framework-task-bus/quickstart.md`
- [ ] T030 [P] [US2] Declare publish/subscribe permissions (least-privilege topics with versions) in `skeleton/plugin.yaml`
- [ ] T031 [US2] Enforce event publish/subscribe permissions at runtime (deny + log + metric) in `skeleton/backend/internal/security/event_permissions.go`
- [ ] T032 [P] [US2] Add unit tests for permission enforcement in `skeleton/backend/tests/unit/event_permissions_test.go`
- [ ] T033 [P] [US2] Implement an in-process TaskBus stub for integration tests (publish → dispatch → consumer) in `skeleton/backend/internal/services/event_bridge/taskbus_stub.go`
- [ ] T034 [P] [US2] Add an integration test for “TaskBus mode E2E” using the stub in `skeleton/backend/tests/integration/event_bridge_taskbus_e2e_test.go`
- [ ] T035 [US2] Add a staging validation checklist (SC-002) in `specs/008-framework-task-bus/quickstart.md`

---

## Phase 5: User Story 3 — 事件契约与任务/观测迁移 (Priority: P3)

**Goal**: 契约治理落地（PR+CI 校验）并推进至少一个 job → consumer 的迁移示例（可先以本地 consumer stub 演示）。

**Independent Test**: 改动 `contracts/channel-events.yaml`（例如新增 topic）会触发 CI 校验；本地可运行 validator；示例 consumer 能处理事件并具备幂等逻辑。

- [ ] T021 [US3] Add a lightweight contracts CI check script wrapper in `scripts/contracts/validate-taskbus-contracts.sh`
- [ ] T022 [US3] Ensure CI runs the wrapper and fails on invalid contracts in `.github/workflows/ci.yml`
- [ ] T023 [US3] Implement a sample consumer interface + local dispatcher (in-memory) in `skeleton/backend/internal/services/event_bridge/consumer.go`
- [ ] T024 [US3] Implement idempotency filter using `topic + tenant_uuid + trace_id` in `skeleton/backend/internal/services/event_bridge/idempotency.go`
- [ ] T025 [P] [US3] Add unit tests for idempotency filter in `skeleton/backend/tests/unit/event_idempotency_test.go`
- [ ] T026 [US3] Migrate one job result write path to consumer handler (local dispatcher) in `skeleton/backend/internal/jobs/channel/master/*`
- [ ] T036 [US3] Add dual-write config (`event_bridge.mode=local|taskbus|dual`) and defaults in `skeleton/backend/internal/config/config.go`
- [ ] T037 [US3] Implement dual-write behavior (emit to TaskBus + local) with clear error semantics in `skeleton/backend/internal/services/event_bridge/emitter.go`
- [ ] T038 [US3] Extend contracts validator to fail on obvious sensitive payload fields (e.g. `password`, `secret`, `token`, `access_key`) in `tools/contracts/validate-taskbus-contracts.go`

---

## Phase 6: Polish & Cross-Cutting

- [ ] T027 [P] Update documentation links and examples in `docs/plan/008-framework-task-bus.md`
- [ ] T028 [P] Add minimal metrics hooks (emit/consume latency & error count) in `skeleton/backend/internal/observability/event_bridge/metrics.go`
- [ ] T039 [P] Add unit/integration tests verifying metrics counters/histograms are updated on emit/consume success/failure in `skeleton/backend/tests/*`
- [ ] T040 [P] Finalize “ops signals” section (alerts/dashboards/rollback) and align with real metric names in `specs/008-framework-task-bus/quickstart.md`
- [ ] T029 Run `go test ./skeleton/backend/...` and ensure no regressions (exclude unrelated flaky tests)

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → US1 (MVP) → US2 → US3 → Polish
- US2 depends on the foundational emitter factory + config.
- US3 depends on the contracts validator + CI hook.

## Parallel Opportunities

- T002 + T003 + T004 can be parallel once T001 confirms scope.
- T011 + T012 can be parallel.
- Validator tasks (T009/T010/T021/T022) can be parallel with emitter tasks (T007/T008) as long as file paths do not overlap.
