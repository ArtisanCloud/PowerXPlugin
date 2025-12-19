# Tasks: 插件能力注册与暴露治理闭环

**Input**: design artifacts from `/specs/006-plugin-capability/` (spec, plan, research, data-model, contracts)

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Initialize capability workspace directories (`capabilities/`, `contracts/exposure/`, `dist/agent-sdk/`) and ensure Make/NPM scripts exist for lint/export (update `scripts/capabilities/package.json`).
- [X] T002 [P] Add Go/Nuxt toolchain prerequisites to `README.md` + `quickstart.md` and wire `make capabilities-lint/export` targets.

---

## Phase 2: Foundational (Blocking Prerequisites)

- [X] T010 Define Capabilities Manager interfaces in `skeleton/backend/internal/capabilities/manager.go` (ListCapabilities/ExportProtocols/RegisterWithHost) and wire dependency injection.
- [X] T011 Implement capability catalog parser for `plugin.yaml` + `capabilities/*.yaml` in `scripts/capabilities/catalog-parser.ts`, producing normalized JSON for Go runtime。
- [X] T012 Create OpenAPI/Proto/Workflow/MCP/SSE asset generation commands (`scripts/capabilities/export.ts`) referencing `specs/006-plugin-capability/contracts/`.
- [X] T013 Add manifest schema support for `capabilities.imports` + protocol descriptors in `scaffold/plugin.yaml.tmpl` 与 `docs/guides`。
- [X] T014 Wire failure handling + rollback hooks in plugin installer (`skeleton/backend/cmd/server/runtime/install.go`) when `capability_registry` sync返回失败。
- [X] T015 [US5] Enforce default sync/async contract：在 `scripts/capabilities/export.ts` 与 `skeleton/backend/internal/capabilities/manager.go` 校验 `async` 标记，默认同步执行，async 能力需声明回调/SSE + 状态查询字段。
- [X] T016 [P][US5] Add unit tests `tests/capabilities/async_mode_test.go`（或等价）验证默认同步/async 行为与回滚策略。

**Checkpoint**: Capabilities Manager +目录解析+资产生成 ready。

---

## Phase 3: User Story 1 — 开发者 5 分钟完成能力建模 (P1)

**Goal**: 表单/CLI 引导完成能力建模并通过校验；生成能力 ID 与审计。

- [X] T101 [US1] 更新 web-admin 表单页面 `skeleton/web-admin/app/pages/capabilities/RegisterForm.vue`，支持字段模板、多语言提示、草稿恢复。
- [X] T102 [P][US1] 扩展 CLI `scripts/capabilities/registry-cli.mjs`，新增 `capabilities new` 与 Schema stub 生成。
- [X] T103 [US1] 实现后台 Handler `skeleton/backend/internal/transport/http/admin/capability/register_handler.go`，接入服务层校验/审计。
- [X] T104 [US1] 在 `skeleton/backend/internal/services/capability/register_service.go` 实现命名冲突检查、Schema 校验、ID 生成、审计写入。
- [X] T105 [US1] 添加 Go 单元测试 `skeleton/backend/internal/services/capability/register_service_test.go` 覆盖冲突/草稿场景。

**Independent test**: 表单 + CLI 提交→能力 ID 与审计。

---

## Phase 4: User Story 2 — 多角色审核与整改闭环 (P2)

**Goal**: 审核工作流自动派发、评论、整改、双人复核。

- [X] T201 [US2] 更新服务 `skeleton/backend/internal/services/capability_review/workflow_service.go`，根据敏感度生成任务、SLA、升级策略。
- [X] T202 [P][US2] 扩展 `skeleton/backend/internal/transport/http/admin/capability/review_handler.go` 支持评论、附件、整改再提交。
- [X] T203 [US2] 实现复核/升级事件推送（Notification Center）于 `skeleton/backend/internal/observability/capability/review_events.go`。
- [X] T204 [US2] 编写审核流程集成自测 `skeleton/backend/tests/integration/capability_review_flow_test.go`（需位于 skeleton 模块下以访问 internal 包）。

---

## Phase 5: User Story 3 — 宿主管理员配置暴露通道与租户授权 (P3)

**Goal**: 多协议通道配置、租户授权、文档/SDK 自动生成。

- [X] T301 [US3] 提供多协议通道配置 UI `skeleton/web-admin/app/pages/capabilities/ExposureForm.vue`（REST/GraphQL/gRPC/Webhook/Workflow/Agent/SDK），支持限流/额度。
- [X] T302 [US3] 在 `skeleton/backend/internal/services/capability/exposure_service.go` 同步渠道配置、生成 docs bundle、SDK bundle。
- [X] T303 [P][US3] 实现租户授权 API `skeleton/backend/internal/transport/http/admin/capability/quota_handler.go`，支持额度调整与审计。
- [X] T304 [US3] 更新 `contracts/exposure/openapi.yaml` & `dist/agent-sdk/` 生成脚本，确保 3 分钟内同步至 API Gateway/Portal。
- [X] T305 [US3] 将暴露配置整合进能力注册页：复用 catalog 协议（`metadata.protocols.*`）生成只读通道模板，统一在 `RegisterForm.vue` 中管理租户授权/限流，并移除 `/capabilities/exposure` 独立页面；同时在 UI 与 `spec.md` 里明确要求开发者在修改 `skeleton/plugin.yaml` 或 `contracts/capabilities/*.yaml` 后执行 `scripts/capabilities run catalog -- --manifest "$(pwd)/skeleton/plugin.yaml"` 以保持 `capabilities/catalog.json` 最新。

---

## Phase 6: User Story 4 — 订阅方感知能力版本变更与下线 (P4)

**Goal**: 生成差异报告、灰度计划、通知/回滚。

- [X] T401 [US4] 实现变更申请入口 `skeleton/web-admin/app/pages/capabilities/Lifecycle.vue`，允许上传 diff、配置灰度窗口。
- [X] T402 [US4] 服务层 `skeleton/backend/internal/services/capability_lifecycle/plan_service.go` 生成差异报告、触发通知、控制双版本并行。
- [X] T403 [US4] 实现通知编排 `skeleton/backend/internal/observability/capability/lifecycle_notifier.go`，保证 100% 覆盖与失败重试。
- [X] T404 [US4] 集成测试 `tests/integration/capability_lifecycle_plan_test.go` 验证灰度暂停/回滚。

---

## Phase 7: User Story 5 — 插件能力目录与 PowerX Workflow/Agent 统一同步 (P2)

**Goal**: Capabilities Manager 导出能力目录 + Workflow/Agent 模板，3 分钟内同步至宿主。

- [X] T501 [US5] 在 `scripts/capabilities/export.mjs` 生成 Workflow Step JSON (`contracts/exposure/workflow/*.json`) 与 Agent manifest (`contracts/exposure/mcp-tools.json`, `agent-streams/*.yaml`)。
- [X] T502 [US5] 实现 Capabilities Manager 与宿主的 REST 客户端 `skeleton/backend/internal/integrations/powerx/capability_client.go`（调用 `/internal/plugins/capabilities/**`）。
- [X] T503 [US5] 在插件安装流程 `skeleton/backend/cmd/plugin/runtime/install.go` 调用 Cap Manager → PowerX，同步失败立即回滚。
- [X] T504 [US5] 编写 Workflow/Agent E2E 验证脚本 `skeleton/backend/tests/integration/capability_catalog_sync_test.go`，覆盖协议资产导出与宿主注册。
- [X] T505 [US5] 增加 async 能力端到端校验脚本 `skeleton/backend/tests/integration/capability_async_mode_test.go`，验证默认同步与 async 回调/SSE 行为。
- [X] T506 [US5] 实现 MCP REGISTER/ACK/INVOKE API：在 `skeleton/backend/internal/transport/http/admin/runtime_ops/sessions_handler.go` 与 `services/admin/runtime_ops/mcp_session.go` 完成会话注册、心跳、关闭处理，并把请求委托给 `integration.SessionAdapter`。
- [X] T507 [US5] 提供 MCP 长连接通道：在插件后端新增 `/mcp/sse`（SSE）与 `/mcp/ws`（WebSocket）入口，复用 `integration.DispatchService`，确保与普通 REST/gRPC 能力共享同一套调度逻辑。
- [X] T508 [US5] 补充集成测试 `skeleton/backend/tests/integration/mcp_agent_mode_test.go`，验证同一能力可被 REST Agent 与 MCP Client 同时调度，覆盖 REGISTER→INVOKE→SSE 推送。

---

## Phase 8: Polish & Cross-Cutting

- [X] T801 [P] 文档更新：`docs/plan/006-plugin-capability.md`、`docs/guides/publish/capabilities.md`、`docs/guides/quickstart.md`。
- [X] T802 [P] Logging/metrics：新增 `capability.catalog.sync_status`, `capability.workflow.async_duration` 指标并接入观察面板。
- [X] T803 [P] Run `make test && npm run test && make capabilities-lint/export`，验证 quickstart 流程（已作为默认脚本）。
- [X] T804 收敛 manifest/RBAC 更新、版本号与 release note（`plugin.yaml` Schema 路径、CHANGELOG 对齐）。

## Phase 9: Template Demo Enhancements (Add-on)

- [X] T901 [US5] 新增「批量克隆/导入」能力契约：在 `contracts/schema/input|output/com.powerx.plugins.base.template.batch_clone.json`、`contracts/capabilities/com.powerx.plugins.base.template.batch_clone.yaml`、`plugin.yaml`、`capabilities/catalog.json` 声明 REST/gRPC/Workflow/Agent 协议，并扩展 `scripts/capabilities/export` 以导出新的 artefact。
- [X] T902 [US5] Service 支撑：为 `TemplateService` 添加 `BatchClone` 与 `Validate` 方法（含 repository/transaction 与规则执行器），新增测试 `skeleton/backend/internal/services/admin/templates/template_service_batch_clone_test.go`、`template_service_validate_test.go` 覆盖成功/失败路径。
- [X] T903 [US5] HTTP/gRPC 入口：
  - 能力接口：在新的 capability handler（同目录或拆分文件）实现 `POST /api/v1/templates/batch-clone`、`POST /api/v1/templates/{id}/validate`，纳入 OpenAPI/能力鉴权。
  - Admin 业务接口：在现有 `template_handler.go` + 路由组新增 `/admin/api/templates/batch-clone`、`/admin/api/templates/{id}/validate`，供 Web Admin 操作；可以包含附加字段但最终委托同一 Service。
  - gRPC：同步在模板 gRPC server/Proto 中注册 `BatchCloneTemplates`、`ValidateTemplate`。
  - 其余：补充 DTO、错误处理、审计日志，确保两套接口共用 Service 并保持行为一致。
- [X] T904 [US5] Workflow/MCP 暴露：创建 `contracts/exposure/workflow/template-quality-distribute.json`、`contracts/exposure/agent-streams/template-quality-distribute.yaml` 与 Tool `base.template.quality_distribute`，并在 `contracts/exposure/exposure-packages.json`、`skeleton/plugin.yaml` 注册新 Workflow 节点。
- [X] T905 [P] 文档与示例：更新 `docs/guides/publish/capabilities/workflow-agent-guide.md`、`mcp-guide.md`、`agent-rest-grpc-guide.md`，描述批量克隆 + 校验的 REST/gRPC 示例与 Workflow/Agent 调用指南，同时在 `docs/plan/006-plugin-capability.md` 追加新场景介绍。

## Dependencies & Execution Order

1. Phase 1 → Phase 2（基础设施 Ready）。
2. 完成 Phase 2 后，US1–US5 可按优先级并行实施：
   - US1 (P1) 最先交付，支撑后续目录功能。
   - US2 依赖 US1 的 CapabilityRecord；US3 依赖 US1/US2 数据但可并行开发 UI。
   - US4 依赖 US1/US3 的能力与暴露元数据。
   - US5 依赖基础目录解析（Phase 2）和 US3 的多协议资产。
3. Phase 8 在所有故事完成并验收后执行；Phase 9 作为 add-on，依赖 Phase 7/8 的基础设施与文档框架，可在主线验收后插入。
