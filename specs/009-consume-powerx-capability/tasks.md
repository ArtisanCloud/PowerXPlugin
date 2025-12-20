# Tasks: PowerX 通用能力插件消费

**Input**: Design documents from `/specs/009-consume-powerx-capability/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 对齐文档与环境基线，确保所有团队理解凭证、CLI 与入口。

- [ ] T001 更新 `docs/standards/powerx-plugin/deploy/env_vars.md`，补充 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`、`PX_TENANT_UUID`、`PX_TOOL_TOKEN` 的含义与注入位置。
- [ ] T002 扩写 `docs/guides/develop/cli-plugin-tutorial.md`，加入 `px-plugin login --manifest ./skeleton/plugin.yaml` 与 `.env.local` 写入流程。
- [ ] T003 在根 `README.md` 与 `docs/plan/009-consume-powerx-capability.md` 互相添加 quickstart 链接，方便新人找到调用指南。

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 建立 manifest 与 CLI 校验基线，为两种模式解锁后续工作。

- [ ] T004 将 `requiredCapabilities` 示例与注释写入 `skeleton/plugin.yaml`，并在 `capabilities/README.md` 说明如何维护。
- [ ] T005 更新 `docs/guides/quickstart.md`，描述 `requiredCapabilities` 字段与 `px-plugin capabilities plan|apply --manifest ./skeleton/plugin.yaml` 的校验步骤。
- [ ] T006 扩展 `scripts/capabilities/run-from-package.mjs`，新增 `--mode skeleton|host` 以及自动回退 `./skeleton/plugin.yaml` 的 manifest 解析逻辑。

**Checkpoint**: Manifest/CLI 配置完成，Gateway 客户端开发可并行展开。

---

## Phase 3: User Story 1 - 宿主插件统一调用核心能力 (Priority: P1) 🎯 MVP

**Goal**: 宿主模式插件通过框架封装直接调用 Integration Gateway，统一鉴权、错误与 trace。
**Independent Test**: 在宿主环境通过新 Gateway Client 调用 `com.corex.media.assets.manage` 并拿到 traceId，`tests/capabilities/media_invocation_test.go` 通过。

### Implementation

- [ ] T007 [US1] 在 `framework/backend/go/internal/integration/gateway/client.go` 新建 Gateway Client，封装 `/tenant/invocations` REST 与 gRPC `InvokeCapability` 调用。
- [ ] T008 [US1] 将 Gateway Client 注入 `framework/backend/go/bootstrap/app.go` 与相关 DI（如 `framework/backend/go/runtime/bootstrap`），读取 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`、`PX_TENANT_UUID`。
- [ ] T009 [US1] 在 `framework/backend/go/internal/services` 下新增 `capabilityinvoker` 服务，统一 action/payload 校验、错误映射与 trace logging。
- [ ] T010 [US1] 在 `framework/frontend/nuxt/framework-admin/layer/app/plugins/powerx-gateway.client.ts` 与 `app/composables/usePowerXCapability.ts` 提供前端调用封装，默认访问 `runtimeConfig.public.gatewayBaseUrl`。
- [ ] T011 [US1] 扩展 `framework/frontend/nuxt/framework-admin/nuxt.config.ts`，暴露 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`、`PX_TENANT_UUID` 到 runtimeConfig（含类型声明）。
- [ ] T012 [US1] 在 `tests/capabilities/media_invocation_test.go` 添加 stub Gateway 集成测试，校验 traceId 回填与限流错误封装。
- [ ] T013 [US1] 更新 `docs/guides/develop/standalone-mode.md`（或新增章节）记录宿主模式 Gateway Client 的使用示例与错误排查指引。

**Checkpoint**: 宿主插件可直接调用 PowerX 能力，日志含 traceId，测试通过。

---

## Phase 4: User Story 2 - Skeleton 模式复用同一封装 (Priority: P2)

**Goal**: Skeleton 本地环境可登录获取 Tool Token，透过相同封装调用 Gateway，并在离线时显式切换 Mock。
**Independent Test**: `skeleton/backend` 与 `skeleton/web-admin` 通过 `.env.local` 配置在 Dev Gateway 下成功列出媒资；`scripts/capabilities/run-from-package.mjs --mode skeleton` 自动读取 Token 并可切换 `--use-mock`。

### Implementation

- [ ] T014 [US2] 在 `skeleton/backend/internal/config/config.go` 添加 `PX_GATEWAY_BASE_URL`、`PX_TOOL_TOKEN`、`PX_TENANT_UUID` 读取，并写入新的 `skeleton/backend/etc/config.example.yaml` 注释。
- [ ] T015 [US2] 新建 `skeleton/backend/internal/integrations/gateway/client.go`，包装框架 Gateway Client，支持 `PX_USE_MOCK` 与离线提示。
- [ ] T016 [US2] 更新 `skeleton/backend/cmd/server/main.go`，将 Gateway Client 注入到业务 service 并在启动时检测 Tool Token 过期。
- [ ] T017 [US2] 在 `skeleton/web-admin/app/composables/usePowerXCapability.ts` 与 `app/plugins/powerx-gateway.client.ts` 实现 Nuxt 端封装，自动携带 `.env.local` 的 Token/Tenant。
- [ ] T018 [US2] 修改 `skeleton/web-admin/nuxt.config.ts` 和 `skeleton/web-admin/.env.example`，暴露 `gatewayBaseUrl`、`powerx.toolToken`、`powerx.tenantUuid`。
- [ ] T019 [US2] 在 `skeleton/web-admin/tests/e2e` 新增 `capability-invocation.spec.ts`，验证 UI 通过 Gateway 成功/失败提示并在 Mock 模式展示 Banner。
- [ ] T020 [US2] 扩展 `scripts/capabilities/run-from-package.mjs` skeleton 分支，自动读取 `skeleton/.env.local`、支持 `--use-mock=<module>` 并输出请求/响应日志。
- [ ] T021 [US2] 在 `docs/plan/009-consume-powerx-capability.md` Skeleton 小节补充 `.env.local` 样例与 `px-plugin login` 步骤截图。

**Checkpoint**: Skeleton 默认连接 Dev Gateway，可一键切 Mock 并复用 CLI 校验。

---

## Phase 5: User Story 3 - 能力调用治理与观测 (Priority: P3)

**Goal**: 统一记录 capabilityId/tenant/traceId/限流事件，提供 doctor/CLI 诊断与运维文档。
**Independent Test**: 触发一次成功调用与一次限流，`framework/backend/go/observability` 生成指标，`tools/cli/src/executors/doctor.ts` 能查出缺失 Token，`docs/operations/observability.md` 提供排查步骤。

### Implementation

- [ ] T022 [US3] 在 `framework/backend/go/observability/tracing.go` 与 `capability_metrics.go` 添加 capabilityId、tenantUUID 维度日志与指标，并暴露 `rateLimitExceeded` 事件。
- [ ] T023 [US3] 更新 `framework/backend/go/internal/services/capabilityinvoker/service.go`，在限流/鉴权失败时记录 `audit.capability.invocation.denied` 并携带 traceId。
- [ ] T024 [US3] 扩展 `tools/cli/src/executors/doctor.ts`，新增 Gateway/Token 检查项（读取 `PX_GATEWAY_BASE_URL`、Token 过期时间、`skeleton/.env.local` 状态）。
- [ ] T025 [US3] 新建 `docs/operations/observability.md`，涵盖指标名称、日志字段、常见错误与定位流程。
- [ ] T026 [US3] 在 `tests/capabilities/rate_limit_test.go` 构造限流 stub，验证日志/事件是否按预期产生。
- [ ] T027 [US3] 新增 `scripts/capabilities/contract-digest.mjs`（或集成至 `run-from-package`），生成 `dist/capability-contracts.json` 并记录能力契约版本/哈希。
- [ ] T028 [US3] 在 `framework/backend/go/internal/integration/gateway/client.go` 加入契约版本检测逻辑（可配置 `PX_GATEWAY_CONTRACT_VERSION`），并向日志/Admin UI 输出升级提示。
- [ ] T029 [US3] 扩展 `tools/cli/src/commands/capabilities/quota.ts`（或新增命令）以及 `docs/plan/009-consume-powerx-capability.md`，提供限流/配额配置指引与示例。

**Checkpoint**: 观测与治理链路齐备，可快速定位与追踪 Gateway 调用问题。

---

## Phase 6: Polish & Cross-Cutting

**Purpose**: 文档、变更记录与最终验收。

- [ ] T030 汇总本次变更并更新 `CHANGELOG.md`「Unreleased」区块，链接到 `docs/plan/009-consume-powerx-capability.md`。
- [ ] T031 按 `specs/009-consume-powerx-capability/quickstart.md` 执行一次端到端校验，并将结果记录在 `logs/quickstart-capability.txt`。
- [ ] T032 检查 `plugin.yaml`、`docs/plan/009-consume-powerx-capability.md`、`specs/009-consume-powerx-capability/spec.md` 是否一致，必要时同步字段描述。

---

## Dependencies & Execution Order

- **Setup → Foundational → User Stories**：Phase 1 与 Phase 2 必须完成后，宿主与 Skeleton 开发才可开始。
- **User Stories**：US1 与 US2 均依赖 Gateway Client（T007～T010）。US3 依赖 US1/US2 产出的调用埋点。
- **Tests**：`tests/capabilities/*.go` 与 `skeleton/web-admin/tests/e2e/*.ts` 需在对应实现完成后运行。

## Parallel Opportunities

- 标记为 [P] 的任务当前为空；当实现过程中确认任务互不依赖，可在 PR 中并行拆分。
- US1 与 US2 完成 Foundational 后可分配给不同开发者；US3 可在 US1 日志接口稳定后并行推进文档与 CLI 部分。

## Implementation Strategy

1. **MVP**：完成 Phase 1~3，确保宿主环境可调用 Gateway 并通过 tests/capabilities 验证。
2. **增量**：Skeleton（Phase 4）与观测（Phase 5）可按团队带宽并行交付，均独立可验收。
3. **收尾**：Polish 阶段同步文档、CHANGELOG，并以 quickstart 流程作为最终验收。

## Parallel Execution Examples

- **US1**：并行推进 T007（Go Client）与 T010/T011（Nuxt 插件）——双方只共享契约文件，可同时开发，最后由 T012 集成测试验证。
- **US2**：T014/T015（后端配置与客户端）与 T017/T018（前端 runtimeConfig）互不依赖，可在不同分支并行；完成后再合流到 T019/T020 的 e2e 与 CLI 调试。
- **US3**：T022（观测埋点）可与 T024（CLI Doctor）同时推进，最后由 T025 文档和 T026 测试进行统一收尾。
