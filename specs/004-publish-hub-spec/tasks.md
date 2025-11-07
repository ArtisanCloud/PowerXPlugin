# Tasks: PowerX Publish Hub Distribution Spec

**Input**: Design documents from `/specs/004-publish-hub-spec/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Align `docs/contracts/publish-hub.openapi.yaml` with `/specs/004-publish-hub-spec/contracts/publish-hub.openapi.yaml` 以确保契约单一真相。
- [X] T002 Update CLI workspace dependencies in `tools/cli/package.json` (Node 18, TS5, Playwright 1.48+) 并运行 `npm install`。
- [X] T003 Add feature flag defaults (`PX_PLUGIN_DEV_MODE`, `PX_PLUGIN_PUBLISH`, `PX_MARKET_OFFLINE_UPLOAD`, `PX_PLUGIN_HUB_ENABLED`) to `config/feature-flags.yaml` 并注明环境矩阵。
- [X] T004 [P] Refresh docs入口 (`docs/use_cases/_from_hub/SCN-PUBLISH-HUB-001/index.md`, `README.md`) 说明新的 CLI/审核/安装流程。

---

## Phase 2: Foundational (Blocking Prerequisites)

- [X] T005 Define `.pxp` artefact schema + integrity/encryption 字段到 `docs/contracts/artefacts/pxp-schema.yaml`（对齐数据模型）。
- [X] T006 Implement encryption & key-envelope helper at `tools/cli/src/lib/security/keyEnvelope.ts`（生成对称密钥、用 Marketplace 公钥封装）。
- [X] T007 Extend telemetry baseline `framework/observability/metrics/publish_metrics.go` 以覆盖 `dev.hotload.*`, `plugin.publish.*`, `plugin.offline.*`。
- [X] T008 [P] Add SLA +告警阈值模板到 `config/alerts/publish-hub.yaml`（在线 ≤4h、离线 ≤1d、安装回滚 ≤5min）。
- [X] T009 Harden RBAC guard + permission mapping `framework/backend/go/runtime/common/middleware/rbac_guard.go` & `framework/backend/go/runtime/admin/middleware/rbac_guard.go`（覆盖开发者/审核员/租户角色）。
- [X] T010 Implement audit log retention policy (≥180 天) in `framework/backend/go/runtime/common/audit/log_retention.go` + `config/logging.yaml` 并记录清理任务。

**Checkpoint**: Artefact契约、加密、Telemetry、SLA、权限与审计基建 ready。

---

## Phase 3: User Story 1 – Developer submits publish candidate (Priority P1) 🎯 MVP

**Goal**: CLI (`px-plugin dev/publish/dist`) 产出合规、签名、加密 artefact 并提交 Publish Hub。
**Independent Test**: 运行 `px-plugin dev --watch`, `px-plugin publish`, `px-plugin dist --target offline`；验证 manifest、签名、加密 `.pxp`、publish receipt、Telemetry。

### Implementation

- [X] T011 [P] [US1] Update `tools/cli/src/commands/publish.ts` 串联预检→签名→上传→回执输出，并写 `publish-receipt.json`。
- [X] T012 [P] [US1] Enhance publish pipeline `tools/cli/src/lib/publish/pipeline.ts`（版本递增、依赖/权限/灰度校验）。
- [X] T013 [US1] Extend `tools/cli/src/lib/publish/precheck.ts` 加入测试覆盖率、签名材料、渠道策略校验。
- [X] T014 [P] [US1] Implement `.pxp` 打包 & integrity 导出 `tools/cli/src/commands/dist.ts` + `tools/cli/src/lib/dist/offlinePackager.ts`（含 `integrity.txt`, `report.json`, `audit.log`）。
- [X] T015 [US1] Wire encryption+密钥封装模块 `tools/cli/src/lib/dist/encryptor.ts` 并向上传 payload 注入密钥元数据。
- [X] T016 [P] [US1] Update Telemetry emitter `tools/cli/src/lib/telemetry/emitter.ts` 输出 `plugin.publish.*` / `plugin.offline.*` 事件。
- [X] T017 [US1] 更新 CLI 文档 `docs/guides/publish/online.md`、`docs/guides/publish/offline.md`（命令示例、加密要求、回执说明）。

**Checkpoint**: CLI 提交/打包能力完整可演示。

### Parallel Opportunities
- T011/T012/T013 vs T014/T015 可分支并行；Telemetry (T016) 与文档 (T017) 可穿插。

---

## Phase 4: User Story 5 – Dev hotload workflow maintains rapid feedback (Priority P1)

**Goal**: `px-plugin dev --watch` + Dev API + Admin SSE 提供秒级反馈并可追溯。
**Independent Test**: 跑 `npm run e2e:dev-hotload` 或 quickstart Dev 流程，验证 register/reload/stop、SSE 日志、Telemetry。

### Implementation

- [X] T018 [P] [US5] Refactor CLI watch command `tools/cli/src/commands/dev/watch.ts`（增量构建、差异包限流、错误提示）。
- [X] T019 [US5] Update SessionClient `tools/cli/src/runtime/hotreload/session.ts`（mTLS、幂等 `x-reload-id`、自动重试、stop API）。
- [X] T020 [P] [US5] Implement Dev API handlers `framework/backend/go/runtime/devapi/handlers/dev_plugins.go` 覆盖 register/reload/delete + SSE 日志。
- [X] T021 [US5] Add telemetry & log piping `framework/backend/go/runtime/devapi/telemetry/hotload_metrics.go`（写入 Redis/Kafka）。
- [X] T022 [P] [US5] Update Admin SSE viewer `examples/starter/web-admin/app/pages/plugins/dev-hotload.vue`（session 状态、错误提示、7 天日志列表）。
- [X] T023 [US5] Extend quickstart `quickstart.md` Dev 章节，写入 stop/resume/故障排查步骤。

### Parallel Opportunities
- CLI (T018/T019) 与 Go Dev API (T020/T021) 并行；Admin 前端 (T022) 与文档 (T023) 随后补充。

---

## Phase 5: User Story 2 – Marketplace reviewer processes submissions (Priority P2)

**Goal**: 在线审核流完成签名/哈希校验、自动测试、审批、SLA 监控与通知。
**Independent Test**: 通过 Marketplace API 提交在线 publish，验证 4 小时内审批结果与 `plugin.publish.approved` 广播。

### Implementation

- [ ] T024 [US2] Implement publish submission handler `framework/backend/go/runtime/marketplace/handlers/publish.go`（入队、扫描、SLA 记录）。
- [ ] T025 [P] [US2] Add 自动化扫描 pipeline `framework/backend/go/runtime/marketplace/services/scanner.go`（签名/哈希/依赖）并输出报告。
- [ ] T026 [US2] Emit `plugin.publish.approved` +租户通知事件 `framework/backend/go/runtime/marketplace/events/publish_events.go`。
- [ ] T027 [P] [US2] Update reviewer console `sdk/workspace/packages/framework-admin/src/pages/marketplace/review.vue`（队列视图、SLA 计时器、审批动作）。
- [ ] T028 [US2] Document reviewer SOP `docs/guides/publish/marketplace-review.md`（含 SLA & checklist）。
- [ ] T029 [P] [US2] Implement online SLA tracker job `framework/backend/go/runtime/marketplace/services/online_sla_tracker.go`（统计在线审核耗时、触发告警）。
- [ ] T030 [US2] Add online Grafana/alert wiring `docs/operations/publish-hub-sla.md` + dashboards（映射 SC-002 指标）。
- [ ] T031 [P] [US4] Implement offline SLA tracker job `framework/backend/go/runtime/marketplace/services/offline_sla_tracker.go`（监控离线上传→审批 ≤1 工作日，超时告警）。
- [ ] T032 [US4] Add离线 SLA dashboard & playbook `docs/operations/publish-hub-sla.md`（新增离线 tab +告警处理流程）。

### Parallel Opportunities
- Handler/Scanner/Events (T024–T026)并行；UI (T027) 和文档 (T028) 可在 API 稳定后跟进；SLA tracker (T029) 与 dashboards (T030) 紧随。

---

## Phase 6: User Story 4 – Offline distribution steward handles `.pxp` packages (Priority P2)

**Goal**: 离线 `.pxp` 上传、密钥封装、签名/哈希校验、白名单租户流转。
**Independent Test**: 通过离线入口上传 `.pxp`，验证密钥解封、签名通过、1 个工作日内审批，白名单租户可见版本。

### Implementation

- [ ] T031 [US4] Build offline upload API `framework/backend/go/runtime/marketplace/handlers/offline_upload.go`（接收 `.pxp`、密钥封装、audit log）。
- [ ] T032 [P] [US4] Implement key unwrap adapter `framework/backend/go/runtime/marketplace/services/keyvault_adapter.go`（Marketplace 私钥、过期检测）。
- [ ] T033 [US4] Extend integrity & audit 校验 `framework/backend/go/runtime/marketplace/services/offline_validator.go`（比对 `integrity.txt`, `report.json`, 敏感字段）。
- [ ] T034 [P] [US4] Add离线审批 UI `sdk/workspace/packages/framework-admin/src/pages/marketplace/offline-review.vue`（白名单配置、SLA 提示）。
- [ ] T035 [US4] Update CLI docs `docs/guides/publish/offline.md` 的加密/密钥交付章节。

### Parallel Opportunities
- API handler (T031) 依赖 key unwrap (T032)；UI (T034) & 文档 (T035) 可在接口定型后执行。

---

## Phase 7: User Story 3 – Tenant admin installs and manages versions (Priority P3)

**Goal**: 租户在 Admin 中查看版本、执行在线/离线安装、灰度、回滚并查看日志。
**Independent Test**: 完成一次在线安装 & 一次离线导入，并在 5 分钟内回滚失败版本，日志/Telemetry 可查。

### Implementation

- [ ] T036 [US3] Implement install URL handler `framework/backend/go/runtime/admin/handlers/plugins_install_url.go`（下载 artefact、调用安装流水线、回滚超时）。
- [ ] T037 [P] [US3] Implement install local handler `framework/backend/go/runtime/admin/handlers/plugins_install_local.go`（上传 `.pxp`、验证签名+密钥封装）。
- [ ] T038 [US3] Enhance deployment orchestration `framework/backend/go/runtime/admin/services/plugin_deployer.go`（状态机、rollback link、日志聚合）。
- [ ] T039 [P] [US3] Update Admin UI `sdk/workspace/packages/framework-admin/src/pages/plugins/manage.vue`（版本列表、灰度批次、回滚按钮、日志）。
- [ ] T040 [US3] Add tenant notifications & telemetry wiring `framework/backend/go/runtime/admin/events/install_events.go`。
- [ ] T041 [US3] Extend quickstart `quickstart.md` 安装/回滚章节，明确“失败 5 分钟内回退”验证步骤。

### Parallel Opportunities
- URL/Local handlers (T036/T037) 可并行；UI (T039) 与事件 (T040) 可在 API stub 完成后跟进；文档 (T041) 收尾。

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T042 [P] Backfill unit/integration tests：`tests/cli/publish.spec.ts`, `tests/cli/dist.spec.ts`, `tests/devapi/dev_plugins_test.go`, `tests/admin/install_flow.spec.ts`。
- [ ] T043 Harden security docs `docs/security/publish-hub.md`（威胁模型、密钥旋转、RBAC 与审计策略）。
- [ ] T044 Performance & SLA验证脚本 `scripts/perf/publish-hub-bench.sh`（热加载 ≤2s、publish ≤10min、离线审核 ≤1d、安装回滚 ≤5min），输出报告。
- [ ] T045 Update `quickstart.md` 最终 checklist & 截图，确保操作人员按步骤复现。

---

## Dependencies & Execution Order

1. **Setup (Phase 1)** → 完成契约/依赖/flag/文档入口。
2. **Foundational (Phase 2)** → 提供 artefact 契约、加密、Telemetry、SLA 模板、RBAC、审计保留，所有用户故事依赖此阶段。
3. **User Stories**:
   - US1 + US5 (P1) 可在 Phase 2 完成后并行，构成 MVP。
   - US2 + US4 (P2) 依赖 US1 artefact & 加密输出，可在 P1 完成后并行；SLA tracker/dashboards (T029/T030) 确保 SC-002。
   - US3 (P3) 依赖 Marketplace 输出的版本与 `.pxp` 流程；UI 可提前基于 mock 开发。
4. **Polish (Phase 8)** → 全局测试、安全文档、性能基准、Quickstart 校验。

### Story Dependency Graph
```
US1 ─┬─> US2 ─┐
      │       ├─> US3
      ├─> US4 ┘
US5 ─┘  (独立，但 P1 与 US1 并行)
```

### Parallel Execution Examples
- CLI publish/dist (US1) vs Dev hotload (US5)；Marketplace 在线/离线 (US2/US4)；Admin 安装 (US3) 可与 UI/文档并进。
- Telemetry/SLA（T029–T030, T044）与安全/Quickstart (T043, T045) 可在主功能完成度 ≥80% 时穿插。

## Implementation Strategy

1. **MVP**: Setup → Foundational → US1 + US5；验证开发者端链路闭环。
2. **Increment 2**: 完成 US2 + US4（Marketplace 在线/离线审核 + SLA 监控）。
3. **Increment 3**: 完成 US3（租户安装/灰度/回滚）。
4. **Hardening**: Phase 8 处理测试、安全、性能、操作文档。
