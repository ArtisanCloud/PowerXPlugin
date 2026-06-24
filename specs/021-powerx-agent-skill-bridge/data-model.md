# Data Model — PowerX Agent Skill Bridge Framework 对齐

## PluginSkillPackage

- **标识**: `plugin_id` + `skill_id` + `version`
- **作用**: 插件侧 Skill 标准源格式。目录包是开发、发布、迁移和审计的源定义；`PluginSkillManifest` 是解析后的运行时对象。
- **推荐目录**:
  - `skills/<skill_id>/SKILL.md`
  - `skills/<skill_id>/schema.input.json`
  - `skills/<skill_id>/schema.output.json`
  - `skills/<skill_id>/executor.yaml`
  - `skills/<skill_id>/scripts/`
  - `skills/<skill_id>/references/`
  - `skills/<skill_id>/assets/`
- **核心字段**:
  - `skill_id`
  - `plugin_id`
  - `version`
  - `package_path`
  - `skill_md_path`
  - `raw_markdown`
  - `frontmatter_json`
  - `body_markdown`
  - `input_schema_json`
  - `output_schema_json`
  - `executor_json`
  - `references_manifest_json`
  - `package_checksum`
  - `loaded_at`
- **规则**:
  - `SKILL.md` 必须包含 YAML frontmatter 与 Markdown body。
  - `id/name/version/description/provider/executor` 必填。
  - schema/executor 可内联或引用包内相对路径；禁止引用包外路径。
  - package checksum 必须覆盖 `SKILL.md`、schema、executor、scripts、references、assets。
  - Framework 启动/发现时从 package 解析 manifest，禁止以硬编码 Go manifest 作为唯一源格式。

## PluginSkillManifest

- **标识**: `provider` + `skill_id` + `version`
- **核心字段**:
  - `skill_id`
  - `provider`
  - `version`
  - `title`
  - `description`
  - `intent_examples[]`
  - `input_schema`
  - `output_schema`（可空）
  - `prompt_refs[]`（可空）
  - `executor`
  - `source_format`（`skill_package`）
  - `package_path`
  - `package_checksum`
  - `raw_markdown`
  - `frontmatter_json`
  - `body_markdown`
  - `visibility`
  - `status`
- **规则**:
  - `skill_id/version/description/input_schema/executor` 必填。
  - 同一插件内 `skill_id + version` 唯一。
  - `executor.type` 必须为 `capability`。
  - 必须提供 `action_capabilities` 或 `executor.action_map`，用于 PowerX Core 将 Skill action 解析为 capability_id。

## PluginSkillCapabilityMapping

- **标识**: `skill_id`
- **核心字段**:
  - `skill_id`
  - `capability`
  - `action_map`
  - `timeout_ms`
  - `async_supported`
  - `risk_level`
- **规则**:
  - 映射注册前必须存在对应 `PluginSkillManifest`。
  - Skill 不能脱离 Agent 独立执行；Agent Runtime 必须先命中 Agent 已绑定 Skill。
  - 请求 action 无法映射到 capability 时拒绝执行。
  - 领域业务由 capability handler 处理，不由 Skill executor 处理。

## PluginSkillInvocationPlan

- **标识**: `trace_id`
- **核心字段**:
  - `skill_id`
  - `version`
  - `action`
  - `capability_id`
  - `input`
  - `context`
  - `idempotency_key`
- **规则**:
  - `context` 必须通过强校验后才能进入 capability handler。
  - `input` 必须按 `input_schema` 校验。
  - 同一 `idempotency_key` 可由业务 capability handler 决定幂等行为。

## PluginSkillInvocationContext

- **标识**: `trace_id`
- **核心字段**:
  - `tenant_uuid`
  - `user_uuid`
  - `agent_id`
  - `session_id`
  - `message_id`
  - `skill_id`
  - `trace_id`
  - `channel`
  - `locale`
  - `capability`
  - `request_id`
- **规则**:
  - `tenant_uuid/user_uuid/agent_id/session_id/skill_id/trace_id` 必填。
  - 缺少必填字段返回 `skill.plugin_context_missing`。
  - 插件不得从 `input` 推断租户或用户。

## PluginSkillResult

- **标识**: `trace_id`
- **核心字段**:
  - `success`
  - `skill_id`
  - `status`（`queued|running|completed|failed|denied`）
  - `message`
  - `task_id`
  - `data`
  - `trace_id`
  - `error`
- **规则**:
  - 同步成功返回 `completed`。
  - 异步任务返回 `queued` 或 `running`，并提供 `task_id`。
  - 失败必须填充稳定 `error.code`。

## PluginSkillError

- **核心字段**:
  - `code`
  - `message`
  - `details`
  - `trace_id`
  - `request_id`
- **标准错误码**:
  - `skill.not_found`
  - `skill.plugin_context_missing`
  - `skill.plugin_capability_mismatch`
  - `skill.plugin_executor_unavailable`
  - `skill.execution_failed`
  - `skill.schema_invalid`
  - `skill.auth_denied`

## PowerXAgentClientConfig

- **标识**: `plugin_id` + `mode`
- **核心字段**:
  - `base_url`
  - `invoke_path`
  - `sse_path`
  - `ws_path`
  - `auth_scheme`
  - `sts_client_id`
  - `timeout_ms`
  - `reconnect_policy`
- **规则**:
  - delegated 模式 `auth_scheme` 固定为 `bearer`。
  - 缺少 base URL 或凭证时启动期 fail-fast。
  - 不允许 fallback 到插件业务 API。

## AgentStreamEvent

- **标识**: `trace_id` + `event_index`
- **核心字段**:
  - `type`（`intent|plan|node_start|node_end|token|final|end|error`）
  - `trace_id`
  - `session_id`
  - `plan_id`
  - `node_id`
  - `payload`
  - `created_at`
- **规则**:
  - SSE 与 WS 输出统一为同一 typed event。
  - 未知事件类型必须返回可诊断错误，不得静默丢弃。
  - `error` 事件必须保留 `trace_id/request_id`。

## PluginLocalChatSession

- **标识**: `local_session_id`
- **核心字段**:
  - `powerx_session_id`
  - `agent_id`
  - `tenant_uuid`
  - `last_trace_id`
  - `created_at`
  - `updated_at`
- **规则**:
  - 本地 Chat 只是 PowerX Agent Session 的客户端镜像。
  - 不保存独立长期对话权威状态。
  - 生产会话权威在 PowerX。

## PluginSkillDefinition

- **标识**: `plugin_id` + `plugin_skill_id` + `version`
- **作用**: 插件 Skill 开发态记录，是插件声明源和同步状态载体，不是 PowerX Agent Runtime 权威源。
- **核心字段**:
  - `plugin_skill_id`
  - `plugin_id`
  - `powerx_skill_id`
  - `version`
  - `title`
  - `description`
  - `intent_examples`
  - `input_schema`
  - `output_schema`
  - `prompt_spec`
  - `executor`
  - `capability`
  - `source_format`
  - `package_path`
  - `package_checksum`
  - `raw_markdown`
  - `frontmatter_json`
  - `body_markdown`
  - `checksum`
  - `sync_status`（`draft|pending|synced|failed|drifted|disabled`）
  - `sync_error_code`
  - `sync_error_message`
  - `last_sync_trace_id`
  - `last_sync_at`
  - `created_at`
  - `updated_at`
- **规则**:
  - `plugin_skill_id/version/executor/capability` 必填。
  - `source_format=skill_package` 时 `package_path/raw_markdown/frontmatter_json/package_checksum` 必填。
  - `sync_status=synced` 前不得作为可运行 Agent 的有效绑定。
  - 本地变更 manifest、schema、prompt 或 executor 后必须将状态置为 `drifted` 或 `draft`，等待重新同步。
  - 同步请求必须由插件 backend proxy 发往 PowerX，前端不得直接调用 PowerX Admin API。

## PluginAgentDefinition

- **标识**: `plugin_id` + `plugin_agent_id`
- **作用**: 插件 Agent 开发态记录，用于配置 prompt、模型引用和插件 Skill 绑定；运行态权威 Agent 在 PowerX。
- **核心字段**:
  - `plugin_agent_id`
  - `plugin_id`
  - `powerx_agent_uuid`
  - `tenant_uuid`
  - `agent_key`
  - `name`
  - `description`
  - `persona`
  - `prompt_seed`
  - `model_profile_ref`
  - `plugin_skill_ids[]`
  - `powerx_skill_ids[]`
  - `sync_status`（`draft|pending|synced|failed|drifted|disabled`）
  - `sync_error_code`
  - `sync_error_message`
  - `last_sync_trace_id`
  - `last_sync_at`
  - `created_at`
  - `updated_at`
- **规则**:
  - `persona` 表示 Agent 身份、人设、服务对象和表达边界，同步到 PowerX `agents.persona`。
  - `prompt_seed` 表示 Agent 默认行为种子，同步到 PowerX Admin API 的 `promptSeed` 并落到 PowerX `agents.prompt_seed`。
  - `system_prompt` 不是 Agent Registry 字段；不得用 `system_prompt`、`meta.persona`、`meta.prompt_seed` 或 `parameters.persona` 作为兼容输入。
  - Skill 执行说明必须写入 `SKILL.md` body / `prompt_spec`，不得塞进 Agent `persona`。
  - Agent `prompt_seed` 只能描述 Agent 级默认行为策略，例如如何介绍已绑定能力、如何选择 Skill、如何处理缺参；具体字段规则必须来自 Skill `input_schema/response_guidance`。
  - 创建 PowerX Session 必须使用 `powerx_agent_uuid`，不得使用 `plugin_agent_id`。
  - `sync_status=synced` 且 PowerX 侧 Agent active 时，才允许出现在 Agent Chat 下拉框。
  - 绑定 Skill 必须已同步到底座并已发布；未同步 Skill 绑定必须 fail-fast。

## PluginSkillResponseGuidance

- **标识**: `plugin_skill_id` + `version` + `response_mode`
- **来源**: `skills/<name>/SKILL.md` frontmatter 的 `response_guidance`
- **标准分组**:
  - `general`
  - `capability_intro`
  - `capability_howto`
  - `clarify_params`
  - `skill_execution`
  - `error_explain`
- **保存路径**:
  - `PluginSkillManifest.response_guidance`
  - `plugin_skills.prompt_spec.response_guidance`
  - PowerX `skills_registry_records.manifest_json.prompt_spec.response_guidance`
- **规则**:
  - `response_guidance` 是 Skill 级表达规范，不是 executor 业务逻辑。
  - 缺参规则必须与 `input_schema.required/properties` 对齐。
  - 禁止在 `response_guidance` 中保存租户、用户、session、token 或权限判断。
  - 插件初始化、seed、同步按钮都必须从 `SKILL.md` 包解析该字段，不得维护另一份硬编码长期定义。
  - PowerXPlugin 只提供该数据；最终 `response_mode` 判断、上下文拼装和自然语言回复由 PowerX Core Agent Runtime 完成。

## PluginRegistrySyncRequest

- **标识**: `trace_id`
- **核心字段**:
  - `sync_kind`（`skill|agent|binding|refresh`）
  - `action`（`create|update|disable|refresh`）
  - `plugin_id`
  - `tenant_uuid`
  - `plugin_agent_id`
  - `plugin_skill_id`
  - `powerx_agent_uuid`
  - `powerx_skill_id`
  - `payload`
  - `operator`
  - `trace_id`
- **规则**:
  - 必须由插件 backend 发送，且使用 delegated bearer/STS 鉴权。
  - 缺少 `plugin_id/tenant_uuid/trace_id` 时 fail-fast。
  - `payload` 必须只包含同步所需字段，不得携带前端会话 token。

## PluginRegistrySyncResult

- **标识**: `trace_id`
- **核心字段**:
  - `success`
  - `sync_kind`
  - `plugin_agent_id`
  - `plugin_skill_id`
  - `powerx_agent_uuid`
  - `powerx_skill_id`
  - `sync_status`
  - `message`
  - `error`
  - `trace_id`
  - `synced_at`
- **规则**:
  - 成功时必须返回 PowerX 侧映射 ID。
  - 失败时必须返回稳定错误码并写回本地 Plugin Registry。
  - 不允许失败后自动改用插件 Agent 或本地 executor 模拟运行。
