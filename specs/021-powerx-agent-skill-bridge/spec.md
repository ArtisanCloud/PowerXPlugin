# Feature Specification: PowerX Agent Skill Bridge Framework 对齐

**Feature Branch**: `021-powerx-agent-skill-bridge`  
**Created**: 2026-06-07  
**Status**: Draft  
**Input**: PowerX 底座 `024-ai-engineering-skills` 已定义 Agent Skill Bridge，需要 PowerXPlugin Framework 提供插件侧 Skill Runtime、Framework Client 与本地 Chat 对齐。

## Clarifications

### Session 2026-06-07

- Q: PowerXPlugin 是否需要新 feature 对齐 PowerX Agent Skill Bridge？ → A: 需要，命名为 `021-powerx-agent-skill-bridge`，独立于 `006-plugin-capability`、`009-consume-powerx-capability` 与 `015-framework-websocket`。
- Q: 插件是否可以定义自己的 Skill？ → A: 可以，但插件侧 Skill 是源定义态能力包；必须经 PowerX 导入、校验、审批发布后，才成为 PowerX 治理态 Skill 并进入 Agent 候选池。
- Q: 插件自有 Chat 是否可以直连插件业务接口？ → A: 不可以作为长期路径。本地 Chat 必须通过 Framework Client 调用 PowerX Agent Session/Stream API。
- Q: Agent 通讯中的 SSE/WS 是否放入 Framework Client？ → A: 是。插件不得在业务层重复实现 PowerX Agent SSE/WS 协议解析。

### Session 2026-06-08

- Q: PowerXPlugin 是否需要插件自有维护 Agent/Skill 记录？ → A: 需要。插件侧维护插件 Agent/Skill Plugin Registry 作为开发态与声明源，同时通过插件 backend proxy 同步到底座形成 PowerX 运行态记录。
- Q: PowerXPlugin 前端是否可以直接访问 PowerX 底座 Admin/Agent/Skill API？ → A: 不可以。必须走 `PowerXPlugin Web -> Plugin Backend Proxy -> PowerX API`。
- Q: 插件 Agent Chat 的 Agent 下拉来源是什么？ → A: 只能展示已同步成功且 PowerX 侧 active 的 Agent；未同步或同步失败的插件 Agent 只能在管理页展示，不能创建会话。

### Session 2026-06-09

- Q: 插件 Skill 是否应该只用 Go manifest 定义？ → A: 不应该。插件 Skill 源格式必须对齐 Agent Skills 的 `SKILL.md` 目录包规范，Go manifest 只能作为解析后的运行时对象或测试样例。
- Q: PowerXPlugin 本地 Plugin Registry 入库是否与 `SKILL.md` 冲突？ → A: 不冲突。`SKILL.md` 是源格式；Plugin Registry 保存解析后的 manifest、prompt、schema、executor、checksum 与同步状态；PowerX 底座保存治理态 Registry。

### Session 2026-06-24

- Q: 插件调试页是否需要自己定义多任务、多智能体和缺参状态协议？ → A: 不需要，也不允许。插件调试页必须消费 PowerX Core Agent Run State Protocol 的 `agent_run.*` 事件，并用 Framework Client reducer 聚合为 `AgentRunState`。
- Q: 插件 Skill 的必填参数、slot 映射和结果链接在哪里定义？ → A: 定义在 `SKILL.md` manifest 中，包括 `action_required_args/action_optional_args/slot_mapping/pending_task_policy/result_presentation`；Core 只执行通用校验和状态流转。
- Q: 插件页面能否根据本地判断显示“已创建/已完成”？ → A: 不可以。只有 PowerX 返回 `agent_run.task_completed` 且包含真实 Skill/Capability result 时，页面才能展示成功状态。

## User Scenarios & Testing

### User Story 1 - 插件声明并暴露 Skill 源定义 (Priority: P1)

作为插件开发者，我希望在插件内按 `skills/<skill_id>/SKILL.md` 目录包声明 Skill，并由 Framework 自动解析、校验和暴露 Skill 发现接口，以便 PowerX 安装/启用插件时能发现并导入插件 Skill。

**Why this priority**: 没有标准发现接口，PowerX 无法把插件能力导入为治理态 Skill，Agent Runtime 也无法选择插件能力。

**Independent Test**: 启动示例插件，访问 `GET /api/v1/plugin/skills`，返回完整 `skill_id/version/provider/input_schema/executor`；缺失必填字段时启动或发现阶段失败。

**Acceptance Scenarios**:

1. **Given** 插件包含合法 `skills/<skill_id>/SKILL.md` 包，**When** Framework 启动或执行发现，**Then** 解析为 `PluginSkillManifest` 并可通过 `GET /api/v1/plugin/skills` 返回。
2. **Given** `SKILL.md` 缺少 `id/name/version/description/executor`、Markdown body 为空、schema 引用越界或 checksum 不一致，**When** Framework 加载 Skill，**Then** 立即失败并输出明确错误。
3. **Given** 同一插件内出现重复 `skill_id + version`，**When** Framework 初始化，**Then** 拒绝启动或拒绝注册重复项。

---

### User Story 2 - 插件 Skill action 通过 Capability 执行业务 (Priority: P1)

作为插件业务开发者，我希望 Skill 通过 `executor.prepare_capability` 自己完成业务状态合并、缺参判断和执行阀门，并在 `ready_to_execute=true` 时返回标准 `capability_request`，由既有 Capability Invocation 统一处理路由、上下文校验、错误模型和结果包装。

**Why this priority**: 如果 Skill 再单独实现 executor 协议，就会形成 Agent 调用、Capability 调用、Skill 调用三套执行链路，租户上下文、错误码、trace 与鉴权会漂移。

**Independent Test**: Agent Runtime 命中插件 Skill 后，先调用 `executor.prepare_capability`。当 prepare 返回 `ready_to_execute=false` 时，页面显示缺参等待；当 prepare 返回 `ready_to_execute=true` 与 `capability_request` 时，Core 才通过 `/api/v1/integration/capabilities/invoke` 或 PowerX Capability Gateway 执行业务；缺少上下文、Skill 不存在、prepare capability 缺失、capability_request 缺失或 capability 不匹配时 fail-fast。

**Acceptance Scenarios**:

1. **Given** PowerX Agent Runtime 命中插件 Skill 并携带完整上下文，**When** planner 生成 Skill task，**Then** Core 先调用 `executor.prepare_capability`，不得直接按 action_map 执行。
2. **Given** prepare 返回缺参，**When** Core 收到 `missing_fields/state_patch/message`，**Then** Core 持久化 SkillState 并发出 `agent_run.awaiting_params`。
3. **Given** prepare 返回 `ready_to_execute=true`，**When** Core 收到 `capability_request`，**Then** Core 只按该请求调用对应 capability handler。
4. **Given** 请求缺少 `tenant_uuid` 或 `trace_id`，**When** capability handler 收到调用，**Then** 返回稳定上下文缺失错误。
5. **Given** prepare 未返回 `capability_request` 却声明 ready，**When** Agent Runtime 尝试执行，**Then** Core fail-fast 并记录 Agent Trace。

---

### User Story 3 - 插件自有 Chat 使用 PowerX Agent Runtime (Priority: P1)

作为插件开发者，我希望插件自有 Chat 页面可以使用 PowerX 智能引擎调试插件 Skill，以便本地调试与生产渠道行为一致。

**Why this priority**: 本地 Chat 如果直连插件业务接口，会形成与生产 Agent Runtime 并行的对话系统，后续难以维护。

**Independent Test**: 在插件自有 Chat 输入自然语言任务，网络请求必须发往 PowerX Agent Session/Stream API；最终由 PowerX Agent Runtime 通过 Skill action 映射调用插件 capability handler。

**Acceptance Scenarios**:

1. **Given** 插件自有 Chat 页面已打开，**When** 用户发送消息，**Then** 页面通过 Framework Client 调用 PowerX Agent Stream。
2. **Given** PowerX Agent Runtime 命中当前插件 Skill，**When** 执行计划到达 `node.kind=skill`，**Then** 当前插件 capability handler 收到 PowerX 注入的完整 Invocation Context。
3. **Given** 页面代码试图直接调用插件领域业务 API 模拟智能任务，**When** 运行静态检查或 E2E，**Then** 验收失败。

---

### User Story 4 - Framework Client 封装 PowerX Agent HTTP/SSE/WS (Priority: P2)

作为框架使用者，我希望通过统一 Client 创建会话、发送消息、消费 SSE/WS 事件，而不是在每个插件里手写 Agent 通讯协议。

**Why this priority**: Agent Stream 事件语义需要跨插件一致；重复手写会造成事件处理、重连、错误映射不一致。

**Independent Test**: 使用 Framework Client 调用 Agent Invoke、Agent SSE、Agent WS 三种入口，均能输出统一事件对象。

**Acceptance Scenarios**:

1. **Given** PowerX Agent SSE 返回 `intent/plan/node_start/node_end/token/final/end`，**When** Framework Client 消费流，**Then** 输出统一 typed event。
2. **Given** WS 连接断开，**When** 配置允许重连，**Then** Client 按策略重连并保留 trace。
3. **Given** PowerX 返回鉴权错误，**When** Client 处理响应，**Then** 返回标准框架错误并保留 `trace_id/request_id`。

---

### User Story 5 - 与 PowerX delegated 鉴权和观测对齐 (Priority: P2)

作为平台运维，我希望插件 Agent Skill 调用遵循 STS/delegated 鉴权、租户隔离和 trace 规范，以便能统一审计和排障。

**Why this priority**: Agent Skill Bridge 是跨 PowerX 与插件的调用链，缺少鉴权与观测会直接破坏安全边界。

**Independent Test**: delegated 模式下未配置 STS/Bearer 时启动失败；成功调用时日志包含 `tenant_uuid/plugin_id/skill_id/session_id/trace_id`。

**Acceptance Scenarios**:

1. **Given** delegated 模式缺少 `PX_GATEWAY_BASE_URL` 或 STS 凭据，**When** 插件启动，**Then** fail-fast。
2. **Given** 插件 executor 被 PowerX 调用，**When** 执行完成，**Then** 日志和 trace 可按 `trace_id` 串联 PowerX 与插件。

---

### User Story 6 - 插件自有维护 Agent/Skill Plugin Registry 并同步 PowerX (Priority: P1)

作为插件开发者，我希望在 PowerXPlugin 本地创建和维护 Agent/Skill 记录，同时一键同步到底座生成 PowerX Agent、Skill 与绑定关系，以便插件侧管理页可以对齐 PowerX 底座 UI，并且调试链路仍然使用 PowerX Agent Runtime。

**Why this priority**: 只有 manifest 暴露无法满足插件自有开发、同步状态追踪和调试入口管理；但插件不能自建一套运行态 Agent 系统，否则会绕开 PowerX 会话、权限、Trace 和租户边界。

**Independent Test**: 在 skeleton 中创建模板 Skill Plugin Definition 与模板 Agent Plugin Definition；点击同步后，插件 backend 调 PowerX API 创建/更新对应 PowerX Skill/Agent；Agent Chat 只能选择该已同步 Agent 创建会话并消费 SSE。

**Acceptance Scenarios**:

1. **Given** 用户在插件侧创建 Skill Plugin Definition，**When** 点击同步到底座，**Then** 插件自有保存 Plugin Registry，并通过 backend proxy 调 PowerX Skill API，回写 `powerx_skill_id/sync_status/last_sync_at`。
2. **Given** 用户在插件侧创建 Agent Plugin Definition 并绑定已同步 Skill，**When** 点击同步到底座，**Then** 插件 backend 调 PowerX Agent API 创建或更新 Agent，并回写 `powerx_agent_uuid`。
3. **Given** Agent Plugin Definition 绑定了未同步或未发布 Skill，**When** 执行同步，**Then** 插件或 PowerX 必须 fail-fast，记录 `sync_error`。
4. **Given** Agent Plugin Definition 未同步成功，**When** 打开 Agent Chat，**Then** 该 Agent 不出现在可运行 Agent 下拉框。
5. **Given** 插件前端发起同步、会话创建或 SSE 请求，**When** 检查网络链路，**Then** 请求必须先进入插件 backend proxy，不允许直接访问 PowerX 底座。

## Edge Cases

- 插件声明重复 `skill_id + version`。
- 插件 executor 内部业务任务是异步队列，需要返回 `queued` 与 `task_id`。
- 插件自有 Chat 无 PowerX 连接或凭证缺失。
- PowerX Agent SSE 中途断开。
- PowerX 调用插件 executor 时缺少租户或会话上下文。
- 插件卸载后 PowerX 仍尝试调用旧 executor。
- capability 不匹配或插件未启用。
- 插件前端绕过 Framework Client 直连业务 API。
- 插件自有 Plugin Registry 与 PowerX 侧记录漂移。
- Agent Plugin Definition 绑定未同步、未发布或不同 provider 的 Skill。
- PowerX 侧 Agent/Skill 被禁用后，插件自有仍显示为可运行。

## Requirements

### Functional Requirements

- **FR-001**: Framework 必须提供 `PluginSkillManifest` 标准结构，覆盖 `skill_id`、`provider`、`version`、`title`、`description`、`intent_examples`、`input_schema`、`output_schema`、`executor`。
- **FR-001a**: Framework 必须支持 `skills/<skill_id>/SKILL.md` 目录包作为插件 Skill 标准源格式，并解析 YAML frontmatter、Markdown body、schema 引用、executor 声明和 package checksum。
- **FR-001b**: Framework 不得要求插件长期以 Go struct 作为 Skill 源定义；Go struct 仅允许作为 loader 输出、测试 fixture 或运行时注册对象。
- **FR-002**: Framework 必须提供 Skill 注册表，插件启动时校验必填字段、重复 ID、executor 声明和 schema 合法性。
- **FR-003**: Framework 必须暴露 `GET /api/v1/plugin/skills`，返回当前插件已注册 Skill 列表。
- **FR-004**: Framework 必须暴露 `GET /api/v1/plugin/skills/:skill_id/schema`，返回指定 Skill 的输入输出 schema。
- **FR-005**: Framework 不得提供独立 Skill invoke 业务执行路径；Skill 执行必须先经过 `executor.prepare_capability`，由 Skill 返回 `ready_to_execute/capability_request` 后进入 PowerX Capability Invocation。
- **FR-006**: Framework 必须定义 `PluginSkillInvocationContext`，至少包含 `tenant_uuid/user_uuid/agent_id/session_id/message_id/skill_id/trace_id/channel/locale`。
- **FR-007**: Framework 必须在 capability handler 执行前强制校验关键上下文，缺失时返回稳定上下文错误。
- **FR-008**: Framework 必须校验请求 `skill_id` 与已注册 Skill 匹配；未找到时返回 `skill.not_found`。
- **FR-009**: Framework 必须校验 prepare 返回的 `capability_request.capability_id` 与 Skill manifest 的 `action_capabilities` / `executor.action_map` 一致；不一致时返回稳定 capability mismatch 错误。
- **FR-010**: Framework 必须复用 Capability Invocation 结果模型，覆盖 `success/status/message/task_id/data/trace_id/error`。
- **FR-011**: Framework 必须提供稳定错误模型，至少覆盖 `skill.not_found`、上下文缺失、capability mismatch、capability unavailable、execution failed。
- **FR-012**: Framework Client 必须封装 PowerX Agent 非流式调用、SSE 流式调用和 WS 流式调用。
- **FR-013**: Framework Client 必须将 PowerX Agent Stream 事件统一解析为 typed event，至少覆盖 `intent/plan/node_start/node_end/token/final/end/error`。
- **FR-014**: 插件自有 Chat 必须通过 Framework Client 调用 PowerX Agent Session/Stream API，禁止直连插件业务 API 作为智能任务主路径。
- **FR-015**: delegated 模式下 Framework Client 必须使用 STS/Bearer 调用 PowerX，禁止读取旧的 `PX_TOOL_TOKEN` 或 `PX_GATEWAY_API_KEY` 作为 delegated 凭证。
- **FR-016**: Framework 必须在日志中输出低基数字段 `plugin_id/tenant_uuid/skill_id/session_id/trace_id/component`，用于链路排查。
- **FR-017**: Framework 必须提供本地调试配置校验器，PowerX Agent base URL 或凭证缺失时给出明确错误，不得静默降级为直连业务 API。
- **FR-018**: Framework 必须支持异步 Skill 结果，executor 可返回 `queued/running/completed/failed` 与 `task_id`。
- **FR-019**: Skeleton 示例必须包含一个最小 Skill 与本地 Chat 调试页面，用于演示 PowerX Agent Runtime 调用插件 executor。
- **FR-020**: 文档必须说明该机制依赖 PowerX `024-ai-engineering-skills`，并给出 MediaX 类插件的接入模板。
- **FR-021**: Framework/Skeleton 必须支持插件 Agent/Skill Plugin Registry 记录，至少保存插件 ID、PowerX 映射 ID、manifest/prompt/schema/executor、绑定 Skill、同步状态、同步错误和最后同步时间。
- **FR-022**: 插件侧 Skill Plugin Definition 同步必须通过插件 backend proxy 调 PowerX Skill Registry/Import/Publish 相关 API，禁止插件前端直接调用 PowerX Admin API。
- **FR-023**: 插件侧 Agent Plugin Definition 同步必须通过插件 backend proxy 调 PowerX Agent Admin API，并携带已同步 PowerX Skill IDs 建立绑定关系。
- **FR-024**: 插件 Agent Chat 必须只展示 `sync_status=synced` 且 PowerX 侧可运行的 Agent；未同步、漂移或失败记录不得创建 PowerX Session。
- **FR-025**: Framework Client 必须提供 Agent/Skill 管理代理能力或调用约束，确保 `list/create/update/sync/refresh` 操作都经插件 backend。
- **FR-026**: 同步失败时必须 fail-fast 并保存稳定 `sync_error.code/message/trace_id`，不得静默降级为本地运行。
- **FR-027**: Skeleton 示例必须提供最小模板 Skill（如 `powerxplugin.template.basic`）与绑定该 Skill 的 Agent Plugin Definition，用于验证本地管理、同步和 Chat 调试闭环。
- **FR-028**: Skeleton 模板 Skill 必须落地为 `skills/template/SKILL.md` 包；初始化按钮必须从该包解析并 upsert/sync，不得继续依赖硬编码 manifest 作为唯一来源。
- **FR-029**: Framework 必须支持 `SKILL.md` frontmatter 的 `response_guidance` 分组，并解析到 `PluginSkillManifest.response_guidance`；同步到底座时必须写入 `prompt_spec.response_guidance`。
- **FR-030**: 插件业务回复规范必须分层维护：Agent `persona/prompt_seed` 只描述 Agent 级身份和策略，Skill `response_guidance/input_schema` 描述能力级参数、缺参追问、how-to 和执行结果表达；不得要求 PowerX Core 写入插件业务专用规则。
- **FR-031**: Framework Client 必须支持 PowerX Agent Run State Protocol typed event，至少解析 `agent_run.started/response_plan/intent_detected/plan_created/task_status/task_started/awaiting_params/task_completed/task_failed/final/ended`。
- **FR-032**: Framework Client 必须提供 `AgentRunState` reducer，将实时 SSE/WS 与历史 payload 聚合为 `run/session/message/response_plan/tasks/pending_params/results/errors/trace_links`。
- **FR-033**: 插件 Agent Chat 调试页必须渲染 `AgentRunState`，展示 task 状态、缺参卡、结果卡和 trace 入口，不得只展示最终文本。
- **FR-034**: Skeleton `SKILL.md` loader 必须解析并校验 `action_required_args/action_optional_args/slot_mapping/pending_task_policy/result_presentation`，同步到底座时保留这些字段。
- **FR-035**: 插件页面不得基于本地状态生成业务成功结论；成功状态必须来自 `agent_run.task_completed` 和真实 `result/links`。
- **FR-036**: Trace 跳转必须携带足够定位字段，至少包含 `tenant_uuid/session_id/message_id/run_id/task_id`；不得只跳到底座 session 列表。

### Key Entities

- **PluginSkillManifest**: 插件侧 Skill 源定义，包含自然语言命中信息、schema 和 executor 声明。
- **PluginSkillPackage**: 插件侧 `SKILL.md` 目录包源格式，包含 frontmatter、Markdown prompt/instructions、schema、executor、scripts/references/assets 与 checksum。
- **PluginSkillRegistry**: 运行时 Skill 注册表，负责注册、校验、查询和分发。
- **PluginSkillInvocation**: PowerX 发起的 Skill 调用请求，包含输入与上下文。
- **PluginSkillInvocationContext**: PowerX 注入的租户、用户、Agent、会话、消息、渠道和 trace 上下文。
- **PluginSkillResult**: executor 返回的结构化结果。
- **PowerXAgentClientConfig**: Framework Client 调用 PowerX Agent 的配置，包含 base URL、STS/Bearer、SSE/WS 路径和超时策略。
- **AgentStreamEvent**: Framework Client 输出的统一事件对象。
- **AgentRunState**: Framework Client 根据 PowerX `agent_run.*` 事件聚合出的运行状态树，用于插件调试页渲染多任务、多 Agent、缺参、结果和错误。
- **AgentTaskState**: 单个 task 的状态对象，包含 Agent、Skill、Capability、action、缺参字段、结果链接、错误与 trace 定位字段。
- **PluginSkillDefinition**: 插件 Skill 开发态记录，包含 manifest、prompt/schema、executor、capability、PowerX skill 映射和同步状态。
- **PluginAgentDefinition**: 插件 Agent 开发态记录，包含 prompt、模型引用、绑定本地/PowerX Skill、PowerX agent 映射和同步状态。
- **PluginRegistrySyncState**: 插件自有同步状态模型，记录 `pending/synced/failed/drifted/disabled`、`sync_error`、`last_sync_at` 与 `trace_id`。

## Success Criteria

- **SC-001**: 合法插件 Skill 在启动后 1 秒内可通过 `GET /api/v1/plugin/skills` 查询，非法 manifest 100% 被拒绝并返回明确错误。
- **SC-002**: PowerX Capability Invocation 不作为标准业务入口；业务执行通过 Capability Invocation，对缺失关键上下文、未知 Skill action、capability 不匹配的请求 100% fail-fast。
- **SC-003**: 插件自有 Chat、PowerX Web Chat 与任意渠道触发同一 Skill 时，100% 走 PowerX Agent Runtime，trace 中可关联 `session_id/skill_id/plugin_id`。
- **SC-004**: Framework Client 能解析 PowerX Agent SSE/WS 的所有标准事件，事件解析错误率为 0。
- **SC-005**: delegated 模式下错误凭证或缺失配置 100% 启动失败，不允许隐式降级到匿名或业务直连。
- **SC-006**: 合法 Agent/Skill Plugin Registry 同步后 95% 在 3 秒内回写 `powerx_agent_uuid/powerx_skill_id/sync_status`。
- **SC-007**: Agent Chat 可运行 Agent 列表中，未同步、同步失败或 PowerX 侧 disabled 的 Agent 出现率为 0%。
- **SC-008**: 插件调试页消费 `agent_run.*` 后，100% 可展示缺参、运行中、完成、失败四类 task 状态。
- **SC-009**: 创建类任务缺参时，插件调试页 100% 展示自然语言缺参提示，不要求用户输入 JSON 或 schema 字段路径。
- **SC-010**: 没有 `agent_run.task_completed` 与真实 result 的请求，插件调试页成功状态误报率为 0。
- **SC-011**: 点击插件调试页 trace 入口，100% 可定位到底座对应 message/task trace，而不是仅进入 session 列表。

## Assumptions

- PowerX 底座已实现或规划 `024-ai-engineering-skills` 中的 Agent Skill Bridge。
- PowerXPlugin 已具备基础 Auth、Gateway Client、WS Bus 与 IAM 机制，可作为本 feature 的依赖。
- 首版以 Go Framework 为主，前端提供最小 Chat Client 封装与 Skeleton 示例。
- 本 feature 不实现 PowerX 侧 Skill Registry，只提供插件侧源定义、executor、client 封装、本地 Plugin Registry 管理和调用 PowerX 的同步代理。
- Agent Run State Protocol 的权威定义在 PowerX Core；PowerXPlugin 只实现 typed event、reducer、UI 消费与 Skill manifest 元数据供给。
