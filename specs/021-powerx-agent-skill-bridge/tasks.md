# Tasks: PowerX Agent Skill Bridge Framework 对齐

**Input**: Design documents from `/specs/021-powerx-agent-skill-bridge/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md  
**Tests**: 必须包含 manifest/registry/context/client 级单元测试、插件路由集成测试和本地 Chat E2E。  

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行
- **[Story]**: US1/US2/US3/US4/US5/Shared
- 每个任务包含明确文件路径

## Phase 1: Setup

- [x] T001 [Shared] 建立 feature 文档集：`specs/021-powerx-agent-skill-bridge/`
- [x] T002 [P] [Shared] 创建 Skill Runtime 目录说明：`framework/backend/go/runtime/skills/README.md`
- [x] T003 [P] [Shared] 创建 PowerX Agent Client 目录说明：`framework/backend/go/runtime/powerx/agent/README.md`
- [x] T004 [P] [Shared] 创建开发指南目录：`docs/guides/develop/agent-skill-bridge/README.md`
- [x] T005 [Shared] 创建 skeleton 示例目录说明：`skeleton/backend/go-gin/internal/skills/README.md`

---

## Phase 2: Foundational

**CRITICAL**: 本阶段完成前，不进入用户故事实现。

- [x] T006 [Shared] 定义 `PluginSkillManifest` 与 executor schema：`framework/backend/go/runtime/skills/manifest.go`
- [x] T007 [P] [Shared] 定义 `PluginSkillInvocation/Context/Result/Error`：`framework/backend/go/runtime/skills/invocation.go`, `framework/backend/go/runtime/skills/result.go`, `framework/backend/go/runtime/skills/errors.go`
- [x] T008 [P] [Shared] 定义 `PowerXAgentClientConfig` 与配置校验：`framework/backend/go/runtime/powerx/agent/config.go`
- [x] T009 [Shared] 实现 manifest 校验器（必填字段、重复 ID、executor 声明）：`framework/backend/go/runtime/skills/validator.go`
- [x] T010 [Shared] 实现 Skill Registry（注册、查询、按 skill_id 分发）：`framework/backend/go/runtime/skills/registry.go`
- [x] T011 [P] [Shared] 实现上下文强校验器：`framework/backend/go/runtime/skills/context_validator.go`
- [x] T012 [P] [Shared] 实现统一错误映射：`framework/backend/go/runtime/skills/error_mapper.go`
- [x] T013 [P] [Shared] 单元测试：manifest/registry/context/error：`framework/backend/go/runtime/skills/*_test.go`

---

## Phase 3: User Story 1 - 插件声明并暴露 Skill 源定义 (P1)

**Goal**: 插件可通过 Framework 注册 Skill 并暴露发现/schema 接口。

### Tests

- [x] T014 [P] [US1] 单元测试：合法 manifest 注册成功：`framework/backend/go/runtime/skills/registry_test.go`
- [x] T015 [P] [US1] 单元测试：缺少必填字段或重复 ID 被拒绝：`framework/backend/go/runtime/skills/validator_test.go`
- [x] T016 [P] [US1] 集成测试：`GET /api/v1/plugin/skills` 返回注册列表：`skeleton/backend/go-gin/internal/transport/http/plugin/skills/skills_routes_test.go`

### Implementation

- [x] T017 [US1] 实现 Skill 发现 HTTP adapter：`framework/backend/go/runtime/skills/http_routes.go`
- [x] T018 [US1] 在 skeleton go-gin 接入 `GET /api/v1/plugin/skills`：`skeleton/backend/go-gin/internal/transport/http/plugin/skills/routes.go`
- [x] T019 [US1] 在 skeleton 注册最小示例 Skill：`skeleton/backend/go-gin/internal/skills/sample_skill.go`
- [x] T020 [US1] 增加 schema 查询路由：`GET /api/v1/plugin/skills/:skill_id/schema`：`framework/backend/go/runtime/skills/http_routes.go`

---

## Phase 4: User Story 2 - 插件统一 Executor 执行业务 Skill (P1)

**Goal**: PowerX 调用统一 executor，Framework 校验上下文并分发领域 handler。

### Tests

- [x] T021 [P] [US2] 单元测试：context 缺失返回 `skill.plugin_context_missing`：`framework/backend/go/runtime/skills/context_validator_test.go`
- [x] T022 [P] [US2] 单元测试：capability 不匹配返回 `skill.plugin_capability_mismatch`：`framework/backend/go/runtime/skills/executor_test.go`
- [x] T023 [P] [US2] 集成测试：插件 Skill 发现返回 capability action_map，业务执行走 Capability Invocation：`skeleton/backend/go-gin/internal/transport/http/plugin/skills/skills_routes_test.go`

### Implementation

- [x] T024 [US2] 实现 executor 注册与 handler 接口：`framework/backend/go/runtime/skills/executor.go`
- [x] T025 [US2] 实现 invoke HTTP adapter：`framework/backend/go/runtime/skills/http_invoke.go`
- [x] T026 [US2] 在 skeleton 接入 PowerX Capability Invocation：`skeleton/backend/go-gin/internal/transport/http/plugin/skills/routes.go`
- [x] T027 [US2] 实现示例 executor，返回 `queued/task_id`：`skeleton/backend/go-gin/internal/skills/sample_executor.go`
- [x] T028 [US2] 接入统一 logger 字段 `plugin_id/tenant_uuid/skill_id/session_id/trace_id`：`framework/backend/go/runtime/skills/logging.go`

---

## Phase 5: User Story 4 - Framework Client 封装 PowerX Agent HTTP/SSE/WS (P2)

**Goal**: 插件通过 Framework Client 调用 PowerX Agent Runtime。

### Tests

- [x] T029 [P] [US4] 单元测试：Agent config 缺失 base URL 或凭证 fail-fast：`framework/backend/go/runtime/powerx/agent/config_test.go`
- [x] T030 [P] [US4] 单元测试：SSE 事件解码为 typed event：`framework/backend/go/runtime/powerx/agent/sse_test.go`
- [x] T031 [P] [US4] 单元测试：WS 事件解码与断线错误映射：`framework/backend/go/runtime/powerx/agent/websocket_test.go`

### Implementation

- [x] T032 [US4] 实现 Agent non-stream invoke client：`framework/backend/go/runtime/powerx/agent/client.go`, `framework/backend/go/runtime/powerx/agent/session.go`
- [x] T033 [US4] 实现 Agent SSE client 与 event decoder：`framework/backend/go/runtime/powerx/agent/sse.go`, `framework/backend/go/runtime/powerx/agent/events.go`
- [x] T034 [US4] 实现 Agent WS client 与重连策略：`framework/backend/go/runtime/powerx/agent/websocket.go`
- [x] T035 [US4] 接入 STS/Bearer 凭证提供器：`framework/backend/go/runtime/powerx/sts/client.go`, `framework/backend/go/runtime/powerx/agent/auth.go`
- [x] T036 [US4] 实现 Agent Client 标准错误对象：`framework/backend/go/runtime/powerx/agent/errors.go`

---

## Phase 6: User Story 3 - 插件调试 Chat 使用 PowerX Agent Runtime (P1)

**Goal**: 本地 Chat 是 PowerX Agent Session 客户端，不直连插件业务 API。

### Tests

- [x] T037 [P] [US3] E2E：本地 Chat 请求目标为 PowerX Agent SSE：`skeleton/web-admin/tests/e2e/agent-skill-bridge.spec.ts`
- [x] T038 [P] [US3] E2E：页面不调用插件业务 API 模拟智能任务：`skeleton/web-admin/tests/e2e/agent-skill-bridge.spec.ts`
- [x] T039 [P] [US3] 集成测试：PowerX Agent 命中插件 Skill 后 executor 收到完整 context（mock PowerX）：`skeleton/backend/go-gin/internal/skills/agent_bridge_integration_test.go`

### Implementation

- [x] T040 [US3] 新增本地 Chat API/service 封装，调用 Framework Agent Client：`skeleton/backend/go-gin/internal/skills/agent_chat_service.go`
- [x] T041 [US3] 新增本地 Chat 前端页面或组件：`skeleton/web-admin/`
- [x] T042 [US3] 前端接入 SSE/WS typed event 渲染：`skeleton/web-admin/`
- [x] T043 [US3] 增加静态检查或 E2E 断言，禁止 Chat 页面直连示例业务 API：`skeleton/web-admin/tests/e2e/agent-skill-bridge.spec.ts`

---

## Phase 7: User Story 5 - delegated 鉴权和观测对齐 (P2)

**Goal**: Agent Skill Bridge 遵循 STS/delegated 鉴权和统一观测。

### Tests

- [x] T044 [P] [US5] 单元测试：delegated 模式禁止读取 `PX_TOOL_TOKEN/PX_GATEWAY_API_KEY`：`framework/backend/go/runtime/powerx/agent/auth_test.go`
- [x] T045 [P] [US5] 单元测试：缺失 STS 配置 fail-fast：`framework/backend/go/runtime/powerx/sts/client_test.go`
- [x] T046 [P] [US5] 集成测试：日志字段包含 trace/tenant/session/skill：`skeleton/backend/go-gin/internal/skills/observability_test.go`

### Implementation

- [x] T047 [US5] 实现 delegated 配置解析与禁止旧 token 规则：`framework/backend/go/runtime/powerx/agent/config.go`
- [x] T048 [US5] 实现 STS token provider 与缓存：`framework/backend/go/runtime/powerx/sts/client.go`
- [x] T049 [US5] 将 Agent Client 与 Capability Handler trace 串联：`framework/backend/go/runtime/skills/logging.go`, `framework/backend/go/runtime/powerx/agent/client.go`
- [x] T050 [US5] 增加诊断输出：mode/base_url/auth_scheme/sts_present：`framework/backend/go/runtime/powerx/agent/diagnostics.go`

---

## Phase 8: Documentation & Rollout

- [x] T051 [P] [Shared] 编写开发指南：`docs/guides/develop/agent-skill-bridge/README.md`
- [x] T052 [P] [Shared] 编写 MediaX 类 Skill 模板：`docs/guides/develop/agent-skill-bridge/mediax-example.md`
- [x] T053 [P] [Shared] 更新插件能力文档，标注 Skill Bridge 与 Capability 的边界：`docs/guides/develop/plugin-capability/README.md`
- [x] T054 [P] [Shared] 更新消费 PowerX 能力文档，标注 Agent Client 与 Gateway Client 的边界：`docs/guides/develop/consume-powerx-capability/README.md`
- [x] T055 [Shared] 执行 quickstart 回归并记录结果：`specs/021-powerx-agent-skill-bridge/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

1. Phase 1 无依赖。
2. Phase 2 依赖 Phase 1，阻塞所有用户故事。
3. Phase 3/4 依赖 Phase 2。
4. Phase 5 依赖 Phase 2，可与 Phase 3/4 并行。
5. Phase 6 依赖 Phase 4/5。
6. Phase 7 依赖 Phase 5。
7. Phase 8 依赖全部用户故事。

### User Story Dependencies

1. US1 是 MVP 入口。
2. US2 依赖 US1 的 registry 与 manifest。
3. US4 可与 US1/US2 并行开发，但最终与 US3 集成。
4. US3 依赖 US2 与 US4。
5. US5 横切 US2/US4。

### Parallel Opportunities

1. T006/T007/T008 可并行。
2. T014/T015/T016 可并行。
3. T021/T022/T023 可并行。
4. T029/T030/T031 可并行。
5. T044/T045/T046 可并行。

## Implementation Strategy

### MVP First

1. 完成 Phase 1-2。
2. 完成 US1 + US2。
3. 能让 PowerX 调 `GET /plugin/skills` 发现 Skill，并通过 Capability Invocation 执行业务能力。

### Incremental Delivery

1. Skill Runtime。
2. Agent Client。
3. 本地 Chat。
4. delegated 鉴权与观测。
5. 文档与模板。
