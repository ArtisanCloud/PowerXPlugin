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

- [X] T024 [US2] Implement publish submission handler `framework/backend/go/runtime/marketplace/handlers/publish.go`（入队、扫描、SLA 记录）。
- [X] T025 [P] [US2] Add 自动化扫描 pipeline `framework/backend/go/runtime/marketplace/services/scanner.go`（签名/哈希/依赖）并输出报告。
- [X] T026 [US2] Emit `plugin.publish.approved` +租户通知事件 `framework/backend/go/runtime/marketplace/events/publish_events.go`。
- [X] T027 [P] [US2] Update reviewer console `examples/starter/web-admin/app/pages/marketplace/review.vue`（队列视图、SLA 计时器、审批动作）。
- [X] T028 [US2] Document reviewer SOP `docs/guides/publish/marketplace-review.md`（含 SLA & checklist）。
- [X] T029 [P] [US2] Implement online SLA tracker job `framework/backend/go/runtime/marketplace/services/online_sla_tracker.go`（统计在线审核耗时、触发告警）。
- [X] T030 [US2] Add online Grafana/alert wiring `docs/operations/publish-hub-sla.md` + dashboards（映射 SC-002 指标）。
- [X] T031 [P] [US4] Implement offline SLA tracker job `framework/backend/go/runtime/marketplace/services/offline_sla_tracker.go`（监控离线上传→审批 ≤1 工作日，超时告警）。
- [X] T032 [US4] Add离线 SLA dashboard & playbook `docs/operations/publish-hub-sla.md`（新增离线 tab +告警处理流程）。

### Parallel Opportunities
- Handler/Scanner/Events (T024–T026)并行；UI (T027) 和文档 (T028) 可在 API 稳定后跟进；SLA tracker (T029) 与 dashboards (T030) 紧随。

---

## Phase 6: User Story 4 – Offline distribution steward handles `.pxp` packages (Priority P2)

**Goal**: 离线 `.pxp` 上传、密钥封装、签名/哈希校验、白名单租户流转。
**Independent Test**: 通过离线入口上传 `.pxp`，验证密钥解封、签名通过、1 个工作日内审批，白名单租户可见版本。

### Implementation

- [X] T031 [US4] Build offline upload API `framework/backend/go/runtime/marketplace/handlers/offline_upload.go`（接收 `.pxp`、密钥封装、audit log）。
- [X] T032 [P] [US4] Implement key unwrap adapter `framework/backend/go/runtime/marketplace/services/keyvault_adapter.go`（Marketplace 私钥、过期检测）。
- [X] T033 [US4] Extend integrity & audit 校验 `framework/backend/go/runtime/marketplace/services/offline_validator.go`（比对 `integrity.txt`, `report.json`, 敏感字段）。
- [X] T034 [P] [US4] Add离线审批 UI `examples/starter/web-admin/app/pages/marketplace/offline-review.vue`（白名单配置、SLA 提示）。
- [X] T035 [US4] Update CLI docs `docs/guides/publish/offline.md` 的加密/密钥交付章节。

### Parallel Opportunities
- API handler (T031) 依赖 key unwrap (T032)；UI (T034) & 文档 (T035) 可在接口定型后执行。

---

## Phase 7: User Story 3 – Tenant admin installs and manages versions (Priority P3)

**Goal**: 租户在 Admin 中查看版本、执行在线/离线安装、灰度、回滚并查看日志。
**Independent Test**: 完成一次在线安装 & 一次离线导入，并在 5 分钟内回滚失败版本，日志/Telemetry 可查。

### Implementation

- [X] T036 [US3] Implement install URL handler `framework/backend/go/runtime/admin/handlers/plugins_install_url.go`（下载 artefact、调用安装流水线、回滚超时）。
- [X] T037 [P] [US3] Implement install local handler `framework/backend/go/runtime/admin/handlers/plugins_install_local.go`（上传 `.pxp`、验证签名+密钥封装）。
- [X] T038 [US3] Enhance deployment orchestration `framework/backend/go/runtime/admin/services/plugin_deployer.go`（状态机、rollback link、日志聚合）。
- [X] T039 [P] [US3] Update Admin UI `examples/starter/web-admin/app/pages/plugins/manage.vue`（版本列表、灰度批次、回滚按钮、日志）。
- [X] T040 [US3] Add tenant notifications & telemetry wiring `framework/backend/go/runtime/admin/events/install_events.go`。
- [X] T041 [US3] Extend quickstart `quickstart.md` 安装/回滚章节，明确“失败 5 分钟内回退”验证步骤。

### Parallel Opportunities
- URL/Local handlers (T036/T037) 可并行；UI (T039) 与事件 (T040) 可在 API stub 完成后跟进；文档 (T041) 收尾。

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T042 [P] Backfill unit/integration tests：`tests/cli/publish.spec.ts`, `tests/cli/dist.spec.ts`, `tests/devapi/dev_plugins_test.go`, `tests/admin/install_flow.spec.ts`。
- [X] T043 Harden security docs `docs/security/publish-hub.md`（威胁模型、密钥旋转、RBAC 与审计策略）。
- [X] T044 Performance & SLA验证脚本 `scripts/perf/publish-hub-bench.sh`（热加载 ≤2s、publish ≤10min、离线审核 ≤1d、安装回滚 ≤5min），输出报告。
- [X] T045 Update `quickstart.md` 最终 checklist & 截图，确保操作人员按步骤复现。

---

## Phase 9: Security & Reliability Enhancements (P1→P2)

**Goal**: 补齐安全层、回滚机制、SLA监控、测试覆盖与权限控制，提升生产就绪度

### P1 Priority: Security & Rollback

- [X] T046 [P1] Implement mTLS for SessionClient：`tools/cli/src/runtime/hotreload/session.ts` 添加证书加载、mTLS配置、重试与Backoff机制
- [X] T047 [P1] Add mTLS verification in Dev API：`framework/backend/go/runtime/devapi/handlers/dev_plugins.go` 集成 mTLS 中间件，校验客户端证书
- [X] T048 [P1] Implement 5-min auto rollback：`framework/backend/go/runtime/admin/services/plugin_deployer.go` 引入状态机、定时器与自动回滚逻辑
- [X] T049 [P1] Wire rollback telemetry & events：`framework/backend/go/runtime/admin/events/install_events.go` 记录回滚事件，更新 SC-004 指标
- [X] T050 [P1] Offline signature verification：`framework/backend/go/runtime/marketplace/services/offline_validator.go` 实现 manifest.signature 验证、证书链校验、CRL检查

### P2 Priority: Monitoring & Testing

- [X] T051 [P2] SLA Dashboard wiring：落地 `config/alerts/publish-hub.yaml` 到 Grafana/Prometheus，在线/离线 tracker 与实际指标 wiring
- [X] T052 [P2] Playwright E2E tests：补全 `tests/admin/install_flow.spec.ts` 真实页面交互测试（安装/回滚流程、SSE日志）
- [X] T053 [P2] RBAC integration：`framework/backend/go/runtime/common/middleware/rbac_guard.go` 接入真实 auth service/权限表与审计日志

### P3 Priority: Documentation & Polish

- [X] T054 [P3] Update `quickstart.md` with mTLS cert setup、rollback verification steps
- [X] T055 [P3] Add mTLS cert rotation guide to `docs/security/publish-hub.md`
- [X] T056 [P3] Create SLA runbook in `docs/operations/publish-hub-sla.md`

### Execution Order (Phase 9)

1. **mTLS** (T046→T047) → 补齐安全基础
2. **Rollback** (T048→T049) → 提升可靠性
3. **Offline Verification** (T050) → 补齐安全链
4. **SLA Dashboard** (T051) → 可观测性
5. **E2E Tests** (T052) → 质量保证
6. **RBAC** (T053) → 访问控制
7. **Docs** (T054→T056) → 文档完善

---

## Phase 10: User Story 0 – Plugin scaffolding & compliance bootstrap (Priority P1)

**Goal**: 让 `px-plugin init`、模板仓、合规扫描与 Git 注册形成 1 分钟交付链，并通过 `px-plugin doctor` / 第三方导入策略支撑团队协作（SCN-DEV-PLUGIN-INIT-001）。
**Independent Test**: 执行 `px-plugin init demo --template react-dashboard` → 自动安装依赖 → 上传 SBOM → 完成 Git 注册；团队成员克隆仓库后运行 `px-plugin doctor --fix` 并导入一个第三方源码包，观察审计与豁免流程。

### Implementation

- [ ] T057 [P] [US0] Introduce CLI scaffold entry `tools/cli/src/commands/plugin/init.ts` + executor（`tools/cli/src/executors/scaffold.ts`）以处理参数解析、模板下载、变量渲染、依赖安装、SBOM 生成与 `publish.yml`/`manifest` 写入。
- [ ] T058 [US0] Build template registry metadata `packages/template-registry/index.yaml`（或 `scaffold/templates/index.yaml`）与版本锁策略，并支持 per-template hooks / language 要求；补充示例模板与回滚策略。
- [ ] T059 [US0] Extend bootstrap + compliance services：`framework/backend/go/runtime/bootstrap/service/bootstrap_handler.go`、`internal/compliance/scanner/license_scanner.go` 支持 CLI 请求校验、Git 注册、许可证/漏洞扫描以及 `plugin-import-audit` Webhook。
- [ ] T060 [US0] Implement `px-plugin doctor`（`tools/cli/src/commands/plugin/doctor.ts` + `tools/cli/src/executors/doctor.ts`）完成环境/依赖/flag 诊断、自动修复与 `.doctor/report.json` 生成。
- [ ] T061 [US0] Add第三方源码导入守护：`tools/cli/src/commands/plugin/import.ts` + `config/compliance/external_source_policy.yaml` + `docs/standards/powerx-plugin/integration/04_security_and_compliance/Plugin_Security_Checklist.md` 更新审批/豁免流程。
- [ ] T062 [US0] Refresh developer docs `docs/guides/bootstrap-context.md`, `docs/guides/cli-plugin-tutorial.md`, `quickstart.md` 加入模板选择、依赖镜像、`plugin doctor`、第三方导入与 Git 注册截图。

### Parallel Opportunities
- T057/T058 可并行（CLI vs 模板仓）；T059 依赖 CLI 协议确定；T060/T061 可在模板完成后推进；T062 收尾。

---

## Phase 11: User Story 6 – Host simulator & sandbox validation close the debug loop (Priority P1)

**Goal**: 提供 `px-plugin host start --mock` + `px-plugin debug attach` + 沙箱验证/错误诊断的端到端工具链（SCN-DEV-PLUGIN-DEBUG-001）。
**Independent Test**: 启动宿主模拟器 → 使用 `px-plugin dev --watch` 重载 → `px-plugin debug attach` 推送断点 → 执行 `plugin-sandbox-suite` → 生成 `POST /internal/debug/report` 并同步工单。

### Implementation

- [ ] T063 [P] [US6] Add host simulator CLI: `tools/cli/src/commands/dev/host.ts`（start/stop/status/attach）串联镜像校验、权限隔离、日志挂载，并输出 sessionId + host endpoint。
- [ ] T064 [US6] Extend SessionClient `tools/cli/src/runtime/hotreload/session.ts` 与调试配置以复用 host session、分发断点、自动重试、`debug-observability-v2` 指标。
- [ ] T065 [P] [US6] Implement Go-side controllers：`framework/backend/go/runtime/devapi/handlers/host_simulator.go`（宿主生命周期、版本守护）与 `sandbox_validation.go`（`POST /internal/sandbox/deploy` orchestration、脱敏数据加载、权限模板）。
- [ ] T066 [US6] Add error diagnostics pipeline：`framework/backend/go/runtime/devapi/telemetry/debug_reports.go`、`framework/backend/go/runtime/devapi/handlers/debug_report.go` 生成脱敏报告、推送工单、记录 `debug.report.generate_ms`。
- [ ] T067 [US6] Update docs & runbooks：`docs/guides/publish/marketplace-review.md`, `docs/guides/bootstrap-context.md`, `docs/guides/cli-plugin-tutorial.md` 添加宿主模拟器、沙箱验证、故障排查与审计流程。

### Parallel Opportunities
- CLI (T063/T064) 与 Go handlers (T065/T066) 并行；文档 (T067) 收尾但需依赖 API 输出。

---

## Phase 12: User Story 7 – Release pipeline & Marketplace orchestration (Priority P1)

**Goal**: 让 `px-plugin publish create/deploy`, `px-plugin pack`, `px-plugin import --offline`、Marketplace 审核和 canary/回滚策略形成统一发布体验（SCN-DEV-PLUGIN-PUBLISH-001）。
**Independent Test**: CLI 创建版本 → 流水线自动门禁 → `px-plugin publish deploy --strategy canary` → `px-plugin pack` 生成 artefact → `px-plugin import --offline` → Marketplace 审核/上架 → 指标/告警合格。

### Implementation

- [ ] T068 [P] [US7] Implement release CLI commands：`tools/cli/src/commands/publish/create.ts` & `publish/deploy.ts`（窗口、批次、审批、灰度/回滚、事件上报）。
- [ ] T069 [US7] Add `px-plugin pack` + offline import wiring：扩展 `tools/cli/src/commands/dist.ts` / 新 `pack.ts` + `tools/cli/src/commands/plugin/import.ts`（pack metadata、signing、`px-plugin import --offline` API 调用、Integrity 校验）。
- [ ] T070 [P] [US7] Build release orchestrator：`framework/backend/go/runtime/publish/pipeline_handler.go`, `config/publish/approval_flows.yaml`、`framework/backend/go/runtime/marketplace/services/offline_validator.go` 扩展 canary/灰度/回滚、SLA 计时与事件。
- [ ] T071 [US7] Update Marketplace/Admin UI：`examples/starter/web-admin/app/pages/publish/pipelines.vue`, `docs/guides/publish/marketplace-review.md`，展示发布计划、灰度状态、Marketplace 审核详情与订阅通知。
- [ ] T072 [US7] Wire metrics & alerts：`framework/observability/metrics/publish_metrics.go`, `config/alerts/publish-hub.yaml`, `workflow-metrics.mjs` 记录 `publish.local.iteration_cycle_time`, `publish.gray.error_rate`, `marketplace.listing.sla_hours` 等新增指标。

### Parallel Opportunities
- CLI (T068/T069) 可与 Go orchestrator (T070) 协同设计协议；UI (T071) 与 Telemetry (T072) 根据 API stub 并行。

---

## Dependencies & Execution Order

1. **Setup (Phase 1)** → 完成契约/依赖/flag/文档入口。
2. **Foundational (Phase 2)** → 提供 artefact 契约、加密、Telemetry、SLA 模板、RBAC、审计保留，所有用户故事依赖此阶段。
3. **User Stories**:
   - US0（Phase 10）先于所有 CLI/发布故事，确保 artefact/manifest/模板一致；完成后可启动 US1/US5。
   - US1 + US5 (P1) 可在 Phase 2 + US0 完成后并行，构成开发者端 MVP。
   - US6（Phase 11）与 US5 紧耦合，需在 Dev API 稳定后补齐宿主/沙箱验证，再反哺调试 SLA。
   - US2 + US4 (P2) 依赖 US1 artefact & 加密输出，可在 P1 完成后并行；SLA tracker/dashboards (T029/T030) 确保 SC-002。
   - US7（Phase 12）构建 `px-plugin publish create/deploy` 与 Marketplace 编排，依赖 US2/US4 的审核能力与 US3 的安装链路。
   - US3 (P3) 依赖 Marketplace 输出的版本与 `.pxp` 流程；UI 可提前基于 mock 开发。
4. **Polish (Phase 8)** → 全局测试、安全文档、性能基准、Quickstart 校验。
5. **Hardening (Phase 9)** → 安全/可靠性收尾，为新增 Workstream 奠定 mTLS、回滚与 SLA 监控。

### Story Dependency Graph
```
US0 ─┐
     ├─> US1 ─┬─> US2 ─┐
US5 ─┘        │       ├─> US3
US6 ──────────┘       └─> US4
US7 <─────────────── US2/US4 输出
```

### Parallel Execution Examples
- US0 模板/CLI（T057/T058/T059）与 doctor/import（T060/T061）可交错，结束后并行推进 US1/US5。
- CLI publish/dist (US1) vs Dev hotload (US5) & Host simulator (US6)；Marketplace 在线/离线 (US2/US4)；Admin 安装 (US3) 可与 UI/文档并进。
- Release orchestrator (US7) 可与 Marketplace UI/Telemetry (T071/T072) 并行，只需共享协议；Telemetry/SLA（T029–T030, T044, T072）与安全/Quickstart (T043, T045, T062) 可在主功能完成度 ≥80% 时穿插。

## Implementation Strategy

- **MVP-A (Scaffold)**: Phase 10（US0）让 CLI 初始化/合规守住 artefact 基线。
- **MVP-B (Dev)**: Phase 2 → US1 + US5 + US6，确保开发/调试闭环。
- **Increment 2 (Marketplace)**: 完成 US2 + US4 → 允许在线/离线审核 + SLA 监控。
- **Increment 3 (Tenant Deploy & Release)**: 完成 US3 + US7 → 覆盖安装/灰度/回滚 + 发布/Marketplace 编排。
- **Polish & Hardening**: Phase 8–9 处理测试、安全、性能、操作文档并维持 SLA。
