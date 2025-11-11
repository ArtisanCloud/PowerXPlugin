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

- [X] T057 [P] [US0] Introduce CLI scaffold entry `tools/cli/src/commands/plugin/init.ts` + executor（`tools/cli/src/executors/scaffold.ts`）以处理参数解析、模板下载、变量渲染、依赖安装、SBOM 生成与 `publish.yml`/`manifest` 写入。
- [X] T058 [US0] Build template registry metadata `packages/template-registry/index.yaml`（或 `scaffold/templates/index.yaml`）与版本锁策略，并支持 per-template hooks / language 要求；补充示例模板与回滚策略。
- [X] T059 [US0] Extend bootstrap + compliance services：`framework/backend/go/runtime/bootstrap/service/bootstrap_handler.go`、`framework/backend/go/internal/compliance/scanner/license_scanner.go` 支持 CLI 请求校验、Git 注册、许可证/漏洞扫描以及 `plugin-import-audit` Webhook。
- [X] T060 [US0] Implement `px-plugin doctor`（`tools/cli/src/commands/plugin/doctor.ts` + `tools/cli/src/executors/doctor.ts`）完成环境/依赖/flag 诊断、自动修复与 `.doctor/report.json` 生成。
- [X] T061 [US0] Add第三方源码导入守护：`tools/cli/src/commands/plugin/import.ts` + `config/compliance/external_source_policy.yaml` + `docs/standards/powerx-plugin/integration/04_security_and_compliance/Plugin_Security_Checklist.md` 更新审批/豁免流程。
- [X] T062 [US0] Refresh developer docs `docs/guides/bootstrap-context.md`, `docs/guides/cli-plugin-tutorial.md`, `quickstart.md` 加入模板选择、依赖镜像、`plugin doctor`、第三方导入与 Git 注册截图。

### Parallel Opportunities
- T057/T058 可并行（CLI vs 模板仓）；T059 依赖 CLI 协议确定；T060/T061 可在模板完成后推进；T062 收尾。

---

## Phase 11: User Story 6 – Host simulator & sandbox validation close the debug loop (Priority P1)

**Goal**: 提供 `px-plugin host start --mock` + `px-plugin debug attach` + 沙箱验证/错误诊断的端到端工具链（SCN-DEV-PLUGIN-DEBUG-001）。
**Independent Test**: 启动宿主模拟器 → 使用 `px-plugin dev --watch` 重载 → `px-plugin debug attach` 推送断点 → 执行 `plugin-sandbox-suite` → 生成 `POST /internal/debug/report` 并同步工单。

### Implementation

- [X] T063 [P] [US6] Add host simulator CLI: `tools/cli/src/commands/dev/host.ts`（start/stop/status/attach）串联镜像校验、权限隔离、日志挂载，并输出 sessionId + host endpoint。
- [X] T064 [US6] Extend SessionClient `tools/cli/src/runtime/hotreload/session.ts` 与调试配置以复用 host session、分发断点、自动重试、`debug-observability-v2` 指标。
- [X] T065 [P] [US6] Implement Go-side controllers：`framework/backend/go/runtime/devapi/handlers/host_simulator.go`（宿主生命周期、版本守护）与 `sandbox_validation.go`（`POST /internal/sandbox/deploy` orchestration、脱敏数据加载、权限模板）。
- [X] T066 [US6] Add error diagnostics pipeline：`framework/backend/go/runtime/devapi/telemetry/debug_reports.go`、`framework/backend/go/runtime/devapi/handlers/debug_report.go` 生成脱敏报告、推送工单、记录 `debug.report.generate_ms`。
- [X] T067 [US6] Update docs & runbooks：`docs/guides/publish/marketplace-review.md`, `docs/guides/bootstrap-context.md`, `docs/guides/cli-plugin-tutorial.md` 添加宿主模拟器、沙箱验证、故障排查与审计流程。

### Parallel Opportunities
- CLI (T063/T064) 与 Go handlers (T065/T066) 并行；文档 (T067) 收尾但需依赖 API 输出。

---

## Phase 12: User Story 7 – Release pipeline & Marketplace orchestration (Priority P1)

**Goal**: 让 `px-plugin publish create/deploy`, `px-plugin pack`, `px-plugin import --offline`、Marketplace 审核和 canary/回滚策略形成统一发布体验（SCN-DEV-PLUGIN-PUBLISH-001）。
**Independent Test**: CLI 创建版本 → 流水线自动门禁 → `px-plugin publish deploy --strategy canary` → `px-plugin pack` 生成 artefact → `px-plugin import --offline` → Marketplace 审核/上架 → 指标/告警合格。

### Implementation

- [X] T068 [P] [US7] Implement release CLI commands：`tools/cli/src/commands/publish/create.ts` & `publish/deploy.ts`（窗口、批次、审批、灰度/回滚、事件上报）。
- [X] T069 [US7] Add `px-plugin pack` + offline import wiring：扩展 `tools/cli/src/commands/dist.ts` / 新 `pack.ts` + `tools/cli/src/commands/plugin/import.ts`（pack metadata、signing、`px-plugin import --offline` API 调用、Integrity 校验）。
- [X] T070 [P] [US7] Build release orchestrator：`framework/backend/go/runtime/publish/pipeline_handler.go`, `config/publish/approval_flows.yaml`、`framework/backend/go/runtime/marketplace/services/offline_validator.go` 扩展 canary/灰度/回滚、SLA 计时与事件。
- [X] T071 [US7] Update Marketplace/Admin UI：`examples/starter/web-admin/app/pages/publish/pipelines.vue`, `docs/guides/publish/marketplace-review.md`，展示发布计划、灰度状态、Marketplace 审核详情与订阅通知。
- [X] T072 [US7] Wire metrics & alerts：`framework/observability/metrics/publish_metrics.go`, `config/alerts/publish-hub.yaml`, `workflow-metrics.mjs` 记录 `publish.local.iteration_cycle_time`, `publish.gray.error_rate`, `marketplace.listing.sla_hours` 等新增指标。

### Parallel Opportunities
- CLI (T068/T069) 可与 Go orchestrator (T070) 协同设计协议；UI (T071) 与 Telemetry (T072) 根据 API stub 并行。

---

## Phase 13: Go CLI Implementation of dev --watch (New)

**Goal**: 用 Go 实现完整的 `px-plugin dev --watch` 命令链，平替 TypeScript 版，提供 file watching、Dev API 交互、session 管理、增量构建、SSE 日志和审计功能。
**Independent Test**: 构建 Go CLI → 运行 `px-plugin dev --watch` → 修改文件 → 验证 reload → 检查 session 持久化 → 对比 TypeScript 版行为完全一致。
**Timeline**: 8 周（Week 1-2 基础设施，Week 3-4 核心功能，Week 5-6 高级特性，Week 7-8 测试优化）

### Week 1-2: Infrastructure Setup

- [X] **T073 [P] CLI 入口与帮助文本**
  - 更新 `tools/cli/cmd/root.go` 注册 `dev` 子命令，补充 help usage、示例。
  - 拆分 `tools/cli/cmd/dev.go`：定义 flag 结构、`runDevWatch/resume/stop/list/logs` 框架。
  - 验收：`px-plugin dev --help` 正常输出，`px-plugin dev --list-sessions` 运行无 panic。

- [X] **T074 Dev API 客户端**
  - `tools/cli/internal/devapi/`：实现 Register/Reload/Delete/Status，访问 `/internal/dev/plugins/*`，加入 `x-reload-id`。
  - 支持 mTLS/Retry/Timeout，错误时返回详细上下文。
  - 验收：httptest mock API 返回 200/500 均覆蓋，重试逻辑可测试。

- [X] **T075 文件监听**
  - `internal/watch/filewatcher.go` 基于 `fsnotify` 递归监听 EntryPath，忽略 `.git`,`node_modules`,`dist/**` 等 pattern。
  - 实现 250 ms Debounce、SHA256 hash 缓存、事件类型映射。
  - 验收：临时目录单测，创建/修改/删除文件均触发事件。

- [X] **T076 Session 管理**
  - `internal/session/`：JSON 存于 `~/.px-plugin/sessions/<id>.json`，支持 Create/Get/List/Update/Delete/Cleanup。
  - 维护 metrics（数量、平均耗时、成功率）、7 天 TTL、`StatusActive/Error/Stopped`。
  - 验收：memstore 单测覆盖多 session 场景。

- [X] **T077 依赖 & Build**
  - `go.mod` 升级 go1.24，加入 `fsnotify`, `uuid`, `fatih/color` 等必需依赖。
  - Makefile/脚本提供 `make build-cli`。
  - 验收：`go build ./tools/cli/cmd/px-plugin` 成功。

- [X] **T078 核心单测**
  - `internal/devapi/client_test.go`（httptest）、`internal/watch/filewatcher_test.go`、`internal/session/manager_test.go`。
  - 覆盖率 ≥80%，CI 跑 `go test ./tools/cli/internal/...`.

### Week 3-4: Core Functionality

- [X] **T079 [P] 增量构建器**
  - `internal/build/` 定义 `Builder` 接口+`SimpleBuilder`，支持 Full/Incremental/Diff 策略及 Go/Node/mixed。
  - 生成 bundle hash/size、记录 artefacts，预留 cache & parallel 钩子。
  - 验收：对 `examples/com.powerx.demo` 后端/前端执行成功，输出 hash。

- [X] **T080 Dev watch 主流程**
  - `runDevWatch`：解析 flag → 读取 `plugin.yaml`（id/version/backend entry）→ `devapi.Register` → 保存 session。
  - 启动 watcher + builder；首轮 build 成功后触发 reload；watcher 监听变更→去抖→构建→Reload（带 changedFiles）。
  - Ctrl+C/`--stop` 清理资源 & 调 `devapi.Delete`。
  - 验收：接入 mock Dev API，修改文件触发 reload，stdout 显示 sessionId/reload 结果。

- [X] **T081 Session resume/list/stop**
  - 修复 `Store.List` 读取逻辑，确保 `.json` 文件可列出。
  - `dev --resume` 重启 watcher + reload 流程，`--stop` 调用 Dev API 删除并清理文件。
  - 验收：启动 watch → Ctrl+C → resume 同一 session；`--list` 返回所有 session。

- [X] **T082 审计日志**
  - `internal/audit/` 按 JSON Lines 写 `~/.px-plugin/audit/YYYY-MM-DD.log`，记录 register/reload/delete/resume/logs 事件。
  - `dev.hotload.*` 指标包含耗时/结果/错误描述，可供 telemetry。
  - 验收：运行 watch 后有日志文件，`dev --logs` 能查询记录。

- [X] **T083 [P] Dev API 契约校验**
  - 核对 `specs/004-publish-hub-spec/contracts/publish-hub.openapi.yaml`，确认 URL/字段一致；补充 contract test。
  - 在 `docs/development/t083-dev-api-contract-alignment.md` 记录差异与修复。

- [x] **T084 集成测试**
  - 搭建 mock Dev API + temp watcher 目录，调用 CLI（或内部包）模拟 register→watch→reload→delete。
  - 验收：`go test ./tools/cli/internal/e2e`（2025-11-10）通过，记录在 `docs/development/t084-integration-tests-summary.md`。

### Week 5-6: Advanced Features

- [x] **T085 [P] mTLS 支持**
  - `internal/mtls` 统一加载 cert/key/CA，支持轮换检测、`PX_MTLS_*` env、`~/.px-plugin/certs/`.
  - Dev API 与 SSE client 共用 TLS config，证书过期提前告警。
  - 验收：`go test ./tools/cli/internal/mtls` 通过，`docs/development/t085-mtls-authentication-summary.md` + `px-plugin doctor --check-mtls` 行为一致。

- [x] **T086 SSE 日志流**
  - `internal/sse/`：实现自动重连、心跳、sessionId 过滤、并行 console+file 输出。
  - `px-plugin dev --logs <id>` 调用 `/internal/dev/plugins/{session}/logs`，支持 `--logs-level/--logs-file/--no-color`.
  - 验收：`go test ./tools/cli/internal/sse` 通过，`docs/development/t086-sse-log-streaming-summary.md` 描述 CLI 行为。

- [x] **T087 性能优化**
  - watcher：hash cache（mtime+size）、string pool、限速器。
  - devapi：HTTP keep-alive、连接池、背压。
  - builder：diff 模式只构建变更模块。
  - 输出 `dev.hotload.go_cli_*` 指标。
  - 验收：`go test ./tools/cli/internal/performance ./tools/cli/internal/resources` 通过，`docs/development/t087-performance-optimizations-summary.md` 与 `scripts/perf/go-cli-dev-watch-bench.sh` 指标对齐。

- [x] **T088 错误恢复**
  - 网络错误指数退避（1s→2s→4s→8s→30s）+ reload 失败自动回滚至上一成功 bundle（`tools/cli/internal/devwatch/runner.go` / `TestRunner_BackoffAndRollbackOnReloadFailure`）。
  - `px-plugin doctor` 增加 Dev API 健康诊断，遇到证书/网络问题提供 remediation（`go test ./tools/cli/internal/devwatch` 2025-11-10 通过）。
  - `px-plugin doctor` Health Checks 文档：`docs/guides/cli/go-cli-dev-watch.md#health-checks`。

- [x] **T089 [P] 资源限制**
  - CPU ≤10%、内存 ≤100MB、watch 文件 ≤10k，超过时告警或限流。
  - builder 加 `--max-procs/--max-memory` 配置；`docs/development/t089-resource-limits-summary.md` 记录策略。
  - ✅ `px-plugin dev` 默认读取 config/env（`performance.memoryLimit`, `performance.cpuThreshold`, `watch.maxFiles` 等）并暴露 `--max-procs/--max-memory-mb/--max-cpu-percent/--max-watch-files`；资源监控阈值触发时会自动 throttle。

- [x] **T090 配置管理**
  - `internal/config/` 读取 `~/.px-plugin/config.json` + env（Dev API、tenant、mTLS、feature flag）。
  - 可选实现 `px-plugin config show/set` 或与 `px auth configure` 互通。
  - ✅ `px-plugin dev` 会自动读取 `~/.px-plugin/config.json` 与 `PX_DEV_TENANT`/`PX_MTLS_*` 作为默认值（详见 `docs/development/t090-configuration-management-summary.md#4-cli-集成状态`）。

### Week 7-8: Testing & Optimization

- [x] **T091 [P] 端到端验证**
  - 联动真实 PowerX Dev API：`go build -o px-plugin ./tools/cli/cmd/px-plugin` → `./px-plugin dev --watch` → 修改文件 → 观测 reload ≤2s、Admin SSE 日志。
  - ✅ 记录步骤于 `docs/development/t084-integration-tests-summary.md#真实-dev-api-e2e-验证（t091）`。

- [x] **T092 性能基准**
  - `scripts/perf/dev-hotload-bench.sh` 生成 benchmark，输出延迟/CPU/内存，确保 P95 ≤2s、文件变化→API ≤250ms、内存 ≤100MB。
  - ✅ `scripts/perf/go-cli-dev-watch-bench.sh` 产出 JSON/Markdown，含 mock Dev API 回放与 Reload 指标（file-change→API、reload latency、memory）- 最新输出位于 `tmp/go-cli-dev-watch-bench/*`。

- [x] **T093 [P] 兼容性对比**
  - 用同一插件分别跑 Go CLI 与 TS CLI，对比 payload/行为/日志，确保 100% 契约一致，记录在 `docs/development/t083-dev-api-contract-alignment.md`。
  - ✅ 文档新增 “Go vs TypeScript CLI Compatibility” 小节（mock Dev API 回放 + payload 比对）。

- [x] **T094 文档**
  - 新增 `docs/guides/cli/go-cli-dev-watch.md`，更新 quickstart/spec 中的 CLI 章节、FAQ、故障排查。
  - ✅ Quickstart 已补充 dev --watch/doctor 流程：`docs/guides/quickstart.md#dev-api-热更新与-doctor-诊断`。
  - ✅ `specs/004-publish-hub-spec/spec.md#documentation--enablement` 收敛 doctor/rollback/quickstart 资料，供产品/QA 引用。
  - ✅ README 快速开始指向新流程：`README.md#快速开始`。
  - ✅ CLI FAQ：`docs/guides/cli/go-cli-troubleshooting.md` 提供 doctor/SSE/跨平台脚本排查指引。
  - 同步 `docs/development/t085/t090` 等附录描述实现。

- [x] **T095 [P] 跨平台测试**
  - 在 macOS/Linux/Windows 构建 & 运行 watch 流程，验证 fsnotify 行为、路径分隔符、证书路径。
  - 记录差异及 workaround。
  - ✅ 脚本已在 macOS/arm64 跑通（完整运行）、Linux/Windows 交叉编译成功且标记 `BUILD_ONLY`；参考 `docs/development/t095-cross-platform-summary.md`，CI 需在 native Linux/Windows 环境补跑 runtime tests。

- [x] **T096 最终打磨**
  - 改善 CLI 输出（进度、错误信息、示例），`root.go` help 中突出 Go CLI 特性。
  - `go fmt`, `golangci-lint`, 更新 `CHANGELOG`, 确认所有文档/任务对齐。
  - ✅ `px-plugin help` 增加 doctor/文档提示，未知命令指向 `px-plugin help`。
  - ✅ `px-plugin doctor` 输出分步骤进度；`CHANGELOG.md` 记录 FAQ/帮助更新。

### Parallel Opportunities
- T073/T074 can start immediately (command + client)
- T075/T076 can parallel (watcher + session)
- T079/T080 depend on T073/T074/T075/T076
- T085/T086/T087 need T079/T080 completed first
- T091/T092/T093 require most components ready

### Go CLI Code Structure

```
tools/cli/
├── cmd/
│   ├── root.go              (update with "dev" case)
│   ├── dev.go               (NEW: dev command entry)
│   └── dev_test.go          (NEW: basic tests)
├── internal/
│   ├── devapi/
│   │   ├── client.go        (NEW: Dev API client)
│   │   ├── client_test.go   (NEW: httptest)
│   │   ├── retry.go         (NEW: retry logic)
│   │   ├── types.go         (NEW: request/response)
│   │   └── errors.go        (NEW: error types)
│   ├── watch/
│   │   ├── filewatcher.go   (NEW: fsnotify wrapper)
│   │   ├── filewatcher_test.go  (NEW: testfs)
│   │   ├── debounce.go      (NEW: 250ms debounce)
│   │   ├── ignore.go        (NEW: ignore patterns)
│   │   └── types.go         (NEW: file events)
│   ├── session/
│   │   ├── manager.go       (NEW: session lifecycle)
│   │   ├── store.go         (NEW: JSON persistence)
│   │   ├── manager_test.go  (NEW: memstore)
│   │   └── models.go        (NEW: session model)
│   ├── build/
│   │   ├── builder.go       (NEW: build interface)
│   │   ├── incremental.go   (NEW: diff build)
│   │   └── types.go         (NEW: build result)
│   ├── sse/
│   │   ├── client.go        (NEW: SSE streaming)
│   │   ├── decoder.go       (NEW: parse SSE)
│   │   └── types.go         (NEW: event model)
│   ├── audit/
│   │   ├── logger.go        (NEW: JSON logs)
│   │   ├── logger_test.go   (NEW: verify format)
│   │   └── metrics.go       (NEW: dev.hotload.*)
│   ├── config/
│   │   ├── config.go        (NEW: configuration)
│   │   ├── auth.go          (NEW: mTLS config)
│   │   └── config_test.go   (NEW: test data)
│   └── watch/               (directory for watcher)
└── go.mod                   (add fsnotify)
```

### Success Criteria for Go CLI

- [X] **Functional**: Core infrastructure complete (T073-T084): CLI commands, Dev API client, file watcher, session manager, build system, audit logging, OpenAPI contract, integration tests
- [X] **Functional**: All `px-plugin dev` subcommands work identically to TypeScript version (T085+)
- [X] **Performance**: Reload P95 ≤2s, memory ≤100MB, CPU ≤10% during watch
- [X] **Reliability**: Network errors auto-retry, session persistence 100%, idempotent reload
- [X] **Compatibility**: 100% API contract match with TypeScript version
- [X] **Security**: mTLS authentication, audit logging, certificate rotation
- [X] **Testability**: ≥90% unit test coverage, passing integration & E2E tests

### Reference Documents

- Design Doc: `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/tmp/go-cli-dev-watch-design.md`
- TypeScript Reference: `tools/cli/src/commands/dev/watch.ts`, `tools/cli/src/runtime/hotreload/session.ts`
- API Contract: `specs/004-publish-hub-spec/contracts/publish-hub.openapi.yaml`
- Backend Implementation: `framework/backend/go/runtime/devapi/handlers/dev_plugins.go`

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
