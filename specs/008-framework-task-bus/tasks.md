# Tasks: Framework TaskBus Event Bridge

**Input**: Design documents from `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/008-framework-task-bus/`  
**Prerequisites**: `specs/008-framework-task-bus/plan.md`, `specs/008-framework-task-bus/spec.md`, `specs/008-framework-task-bus/research.md`, `specs/008-framework-task-bus/data-model.md`, `specs/008-framework-task-bus/contracts/channel-events.yaml`

## Phase 1: Setup (Shared Infrastructure)

- [x] T001 Confirm baseline contracts exist in `specs/008-framework-task-bus/contracts/channel-events.yaml`
- [x] T002 [P] Add a CI check hook placeholder for contracts validation in `.github/workflows/ci.yml`
- [x] T003 [P] Add a local validation entrypoint for contracts validation in `Makefile` or `make-files/test.mk`

---

## Phase 2: Foundational (Blocking Prerequisites)

- [x] T004 Define `event_bridge` config shape and defaults in `skeleton/backend/go-gin/internal/config/config.go`
- [x] T005 Implement `Event` meta builder helper (tenant_uuid/request_id/source_plugin/trace_id/occurred_at/payload_version) in `framework/backend/go/event/meta.go`
- [x] T006 Implement core event types (`Event`, `Topic`, `Subscription` DTOs) in `framework/backend/go/event/models.go`
- [x] T007 Implement an emitter interface and factory (local vs taskbus) in `framework/backend/go/eventbridge/emitter.go`
- [x] T008 Implement a local emitter (in-memory) with fallback semantics in `framework/backend/go/eventbridge/local_emitter.go`
- [x] T009 Implement a contracts validator (topic unique + required meta keys) in `tools/contracts/validate-taskbus-contracts.go`
- [x] T010 Wire the contracts validator into CI (run on PR) in `.github/workflows/ci.yml`

**Checkpoint**: Foundational ready — Emitter API exists; local emitter works; contracts validator enforced.

---

## Phase 3: User Story 1 — 业务侧统一发事件 (Priority: P1) 🎯 MVP

**Goal**: 业务层通过统一 Emitter 发出 Channel 事件（不改业务对外 API）。

**Independent Test**: 在本地（TaskBus 关闭）触发一次示例业务动作（或直接调用 service），确认事件被发出且包含必填 meta 字段。

- [x] T011 [P] [US1] Add unit tests for meta builder in `skeleton/backend/go-gin/tests/unit/event_meta_test.go`
- [x] T012 [P] [US1] Add unit tests for local emitter fallback (no panic + logs) in `skeleton/backend/go-gin/tests/unit/event_local_emitter_test.go`
- [x] T013 [US1] Create a channel event emitter adapter implementing `ChannelEventEmitter` in `skeleton/backend/go-gin/internal/observability/channel/event_emitter.go`
- [x] T014 [US1] Update one representative path to emit `powerx.channel.master.credential_inspection.v1` in `skeleton/backend/go-gin/internal/jobs/channel/master/*`
- [x] T015 [US1] Ensure emitted events include `topic + tenant_uuid + trace_id` idempotency key material in `skeleton/backend/go-gin/internal/observability/channel/event_emitter.go`

---

## Phase 4: User Story 2 — TaskBus 接入与双实现切换 (Priority: P2)

**Goal**: 通过 `event_bridge.enabled` 切换 local/TaskBus，并在 TaskBus 不可用时自动降级到 local。

**Independent Test**: 在开启 TaskBus 模式时注入一个“模拟不可用”的 TaskBus emitter，验证会自动降级到 local 且打点/告警。

- [x] T016 [US2] Implement taskbus emitter adapter skeleton (interface only; no external deps) in `framework/backend/go/eventbridge/taskbus_provider.go`
- [x] T017 [US2] Implement fallback logic: taskbus failure → local emitter in `framework/backend/go/eventbridge/emitter.go`
- [x] T018 [P] [US2] Add integration test for fallback behavior in `skeleton/backend/go-gin/tests/integration/event_bridge_fallback_test.go`
- [x] T019 [US2] Wire emitter into `app.Deps` and service construction in `skeleton/backend/go-gin/internal/shared/app/deps.go`
- [x] T020 [US2] Document config + fallback in `specs/008-framework-task-bus/quickstart.md`
- [x] T030 [P] [US2] Declare publish/subscribe permissions (least-privilege topics with versions) in `skeleton/plugin.yaml`
- [x] T031 [US2] Enforce event publish/subscribe permissions at runtime (deny + log + metric) in `skeleton/backend/go-gin/internal/security/event_permissions.go`
- [x] T032 [P] [US2] Add unit tests for permission enforcement in `skeleton/backend/go-gin/tests/unit/event_permissions_test.go`
- [x] T033 [P] [US2] Implement an in-process TaskBus stub for integration tests (publish → dispatch → consumer) in `framework/backend/go/eventbridge/taskbus_stub.go`
- [x] T034 [P] [US2] Add an integration test for “TaskBus mode E2E” using the stub in `skeleton/backend/go-gin/tests/integration/event_bridge_taskbus_e2e_test.go`
- [x] T035 [US2] Add a staging validation checklist (SC-002) in `specs/008-framework-task-bus/quickstart.md`

---

## Phase 5: User Story 3 — 事件契约与任务/观测迁移 (Priority: P3)

**Goal**: 契约治理落地（PR+CI 校验）并推进至少一个 job → consumer 的迁移示例（可先以本地 consumer stub 演示）。

**Independent Test**: 改动 `contracts/channel-events.yaml`（例如新增 topic）会触发 CI 校验；本地可运行 validator；示例 consumer 能处理事件并具备幂等逻辑。
- [x] T021 [US3] Add a lightweight contracts CI check script wrapper in `scripts/contracts/validate-taskbus-contracts.sh`
- [x] T022 [US3] Ensure CI runs the wrapper and fails on invalid contracts in `.github/workflows/ci.yml`
- [x] T023 [US3] Implement a sample consumer interface + local dispatcher (in-memory) in `framework/backend/go/eventbridge/consumer.go`
- [x] T024 [US3] Implement idempotency filter using `topic + tenant_uuid + trace_id` in `framework/backend/go/eventbridge/idempotency.go`
- [x] T025 [P] [US3] Add unit tests for idempotency filter in `skeleton/backend/go-gin/tests/unit/event_idempotency_test.go`
- [x] T026 [US3] Migrate one job result write path to consumer handler (local dispatcher) in `skeleton/backend/go-gin/internal/jobs/channel/master/*`
- [x] T036 [US3] Add dual-write config (`event_bridge.mode=local|taskbus|dual`) and defaults in `skeleton/backend/go-gin/internal/config/config.go`
- [x] T037 [US3] Implement dual-write behavior (emit to TaskBus + local) with clear error semantics in `framework/backend/go/eventbridge/emitter.go`
- [x] T038 [US3] Extend contracts validator to fail on obvious sensitive payload fields (e.g. `password`, `secret`, `token`, `access_key`) in `tools/contracts/validate-taskbus-contracts.go`

---

## Phase 6: Polish & Cross-Cutting

- [x] T027 [P] Update documentation links and examples in `docs/plan/008-framework-task-bus.md`
- [x] T028 [P] Add minimal metrics hooks (emit/consume latency & error count) in `skeleton/backend/go-gin/internal/observability/event_bridge/metrics.go`
- [x] T039 [P] Add unit/integration tests verifying metrics counters/histograms are updated on emit/consume success/failure in `skeleton/backend/go-gin/tests/*`
- [x] T040 [P] Finalize “ops signals” section (alerts/dashboards/rollback) and align with real metric names in `specs/008-framework-task-bus/quickstart.md`
- [x] T029 Run `go test ./skeleton/backend/go-gin/...` and ensure no regressions (exclude unrelated flaky tests)

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → US1 (MVP) → US2 → US3 → Polish
- US2 depends on the foundational emitter factory + config.
- US3 depends on the contracts validator + CI hook.

## Parallel Opportunities

- T002 + T003 + T004 can be parallel once T001 confirms scope.
- T011 + T012 can be parallel.
- Validator tasks (T009/T010/T021/T022) can be parallel with emitter tasks (T007/T008) as long as file paths do not overlap.

---

## Phase 7: Host Provider 落地与版本发布（Next）

**Goal**: 将 `TaskBusProvider` 从“接口占位”升级为“宿主可用实现”，并完成 framework 新版本发布与插件迁移闭环。

**Independent Test**: 在 `mode=taskbus` 且 provider 可用时，事件经宿主链路成功投递；provider 不可用且 `fallback_to_local=true` 时可自动回落。

- [x] T041 [US4] Implement a real host taskbus provider in `framework/backend/go/runtime/taskbus/provider.go` (or equivalent runtime package).
- [x] T042 [US4] Inject provider via `Factory.WithTaskBusProvider(...)` in `skeleton/backend/go-gin/cmd/plugin/main.go`.
- [x] T043 [P] [US4] Add integration test for "taskbus mode with real provider wiring" in `skeleton/backend/go-gin/tests/integration/event_bridge_taskbus_provider_test.go`.
- [x] T044 [US4] Add drop metrics/log fields for local queue full in `framework/backend/go/eventbridge/local_emitter.go` and `skeleton/backend/go-gin/internal/observability/event_bridge/metrics.go`.
- [x] T045 [P] [US4] Add tests for local-queue-full behavior and metrics update in `skeleton/backend/go-gin/tests/unit/event_local_queue_full_test.go`.
- [x] T046 [US4] Publish framework prerelease (`v0.0.3-alpha` or newer) and update dependency notes in `docs/plan/008-framework-task-bus.md`.
- [x] T047 [US4] Add migration guide for external plugins (adapter mapping checklist) in `docs/guides/async_runtime/event_fabric/integration_playbook.md`.

**Checkpoint**: Host provider wired + fallback verified + release/migration docs ready.

---

## Updated Dependencies & Execution Order (with Phase 7)

- Phase 1 → Phase 2 → US1 (MVP) → US2 → US3 → Polish → **Phase 7 (Host Provider & Release)**
- Phase 7 depends on stable contracts + emitter factory + integration baseline from earlier phases.
- Release/migration tasks (T046/T047) should run after provider wiring and integration validation (T041~T045).

## Phase 7 Entry Gate（进入 Host Provider 开发前）

- [x] G1 已阅读并确认 `specs/008-framework-task-bus/readiness.md` 的未完成项。
- [x] G2 已完成 Day-0 校验命令并保留执行记录。
- [x] G3 目标 topic 权限已在 `skeleton/plugin.yaml` 明确声明。
- [x] G4 已确认迁移目标插件的当前 framework 版本与升级路径。
