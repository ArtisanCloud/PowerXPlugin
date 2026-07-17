# 021 PowerX Agent Skill Bridge 对齐计划

## 1. 背景

PowerX 底座在 `024-ai-engineering-skills` 中定义了 PowerX Agent Skill Bridge：渠道、移动端、Web Chat、SCRM 与插件自有 Chat 统一进入 PowerX Agent Session，由 PowerX Agent Runtime 选择 Skill，并通过标准桥接调用插件 Capability Handler。

PowerXPlugin 侧需要提供配套 Framework 能力：

1. 插件声明 Skill 源定义。
2. 插件暴露统一 Skill 发现与执行接口。
3. 插件 Framework Client 封装 PowerX Agent HTTP/SSE/WS 通讯。
4. 插件自有 Chat 使用 PowerX 智能引擎，不直接调用插件业务 API。
5. 插件自有维护 Agent/Skill Plugin Registry，并通过插件 backend proxy 同步到 PowerX 底座生成运行态记录。
6. 插件 Skill 源定义必须采用 `SKILL.md` 目录包规范；本地 Plugin Registry 和 PowerX Registry 只保存解析后的治理态快照。
7. 插件业务回复规范必须分层维护：Agent `persona/prompt_seed` 保存 Agent 级身份与策略，Skill `response_guidance/input_schema` 保存能力级说明、缺参追问和执行结果表达；PowerX Core Runtime 只负责通用编排和安全边界。
8. 插件调试页与 PowerX Web Admin 必须对齐 PowerX Agent Run State Protocol：多任务、多智能体、缺参等待、执行状态、结果链接和 trace 入口由 `agent_run.*` 事件与 `AgentRunState` reducer 表达，插件不得自定义一套私有任务状态。

## 2. 机制定位

本 feature 命名为：

```text
021-powerx-agent-skill-bridge
```

它不是单纯的 capability 暴露，也不是单纯的 Gateway 消费，而是插件框架层的 Agent Skill 对齐机制。

标准链路：

```text
Plugin Debug Chat / Channel Plugin / Web Admin Debug
        ↓
PowerXPlugin Framework Client
        ↓
PowerX Agent Session / Stream API
        ↓
PowerX Agent Runtime
        ↓
PowerX Agent Skill Bridge
        ↓
Skill prepare capability
        ↓
ready_to_execute=true 时返回 capability_request
        ↓
PowerXPlugin Capability Handler
        ↓
插件领域业务任务
```

Agent/Skill 管理同步链路：

```text
PowerXPlugin Web 管理页
        ↓
Plugin Backend Plugin Registry Store
        ↓
Plugin Backend PowerX Proxy
        ↓
PowerX Skill Registry / Agent Admin API
        ↓
PowerX 治理态 Skill + 运行态 Agent + Binding
```

禁止长期链路：

```text
Plugin Debug Chat / Channel Plugin
        ↓
插件领域业务 API
```

## 3. 目标

1. 建立 Framework Skill Runtime：
   - `PluginSkillPackage`
   - `PluginSkillPackageLoader`
   - `PluginSkillPackageValidator`
   - `PluginSkillManifest`
   - `PluginSkillRegistry`
   - `PluginSkillExecutor`
   - `PluginSkillPrepare`
   - `PluginSkillInvocation`
   - `PluginSkillInvocationContext`
   - `PluginSkillResult`
   - `PluginSkillError`

2. 建立 Framework Client：
   - `PowerXSTSClient`
   - `PowerXAgentSessionClient`
   - `PowerXAgentSSEClient`
   - `PowerXAgentWSClient`
   - `PowerXCapabilityClient`

3. 统一插件 Skill 发现接口。Skill 业务执行不提供独立 invoke 入口，必须由 PowerX Core 调用 `executor.prepare_capability`，再按 Skill 返回的 `capability_request` 进入 PowerX Capability Invocation：

```text
GET  /api/v1/plugin/skills
GET  /api/v1/plugin/skills/:skill_id/schema
executor.prepare_capability -> PowerX Capability Invocation
PowerX Capability Invocation
GET  /api/v1/plugin/skills/invocations/:invocation_id
```

4. 统一本地 Chat：

```text
Plugin Chat UI -> Framework Client -> PowerX Agent Stream
```

5. 强制上下文校验：

```text
tenant_uuid
user_uuid
agent_id
session_id
message_id
skill_id
trace_id
```

缺失关键上下文必须 fail-fast，不允许匿名 fallback。

6. 建立 Agent/Skill Plugin Registry + Sync：

```text
PluginSkillDefinition
PluginAgentDefinition
PluginRegistrySyncRequest
PluginRegistrySyncResult
PluginRegistrySyncState
```

本地 Plugin Registry 只保存开发态声明和同步状态；PowerX 底座 Agent/Skill/Binding 才是运行态权威源。同步失败、漂移、未发布 Skill 绑定时必须 fail-fast，不允许本地模拟运行。

7. 对齐 Agent Run State Protocol：

```text
PowerX Agent SSE/WS agent_run.*
        ↓
PowerXPlugin Framework Client decoder
        ↓
AgentRunState reducer
        ↓
Plugin Agent Chat 调试页面
```

PowerXPlugin 只负责协议消费、状态聚合和 UI 展示，不负责判断任务是否成功。任务成功必须来自 PowerX Core 的 `agent_run.task_completed` 和真实 Skill/Capability result。

## 4. 与既有 feature 的关系

依赖：

1. `005-plugin-auth`：插件鉴权与运行时身份。
2. `006-plugin-capability`：插件能力建模基础。
3. `009-consume-powerx-capability`：调用 PowerX Gateway/STS 的客户端基础。
4. `015-framework-websocket`：可复用实时通讯基础，但不替代 Agent SSE/WS Client。
5. `018-framework-iam-unification`：租户、用户、成员身份统一。

不归入：

1. `006-plugin-capability`：该 feature 只覆盖能力目录与暴露治理，不承担 Agent Session Client。
2. `009-consume-powerx-capability`：该 feature 只覆盖消费 PowerX corex capability，不承担插件 Capability Handler 标准。
3. `015-framework-websocket`：该 feature 只覆盖 WS Bus 发布订阅，不承担 Agent Runtime 事件语义。

## 5. 目录落点

建议实现落点：

```text
skills/<skill_id>/SKILL.md
skills/<skill_id>/schema.input.json
skills/<skill_id>/schema.output.json
skills/<skill_id>/executor.yaml
framework/backend/go/runtime/skills/
framework/backend/go/runtime/powerx/agent/
framework/backend/go/runtime/powerx/sts/
framework/backend/go/runtime/powerx/capability/
framework/frontend/
skeleton/backend/go-gin/
skeleton/backend/go-gin/internal/agent_registry/
skeleton/backend/go-gin/internal/agent_registry/
skeleton/backend/go-gin/internal/sync/
skeleton/web-admin/
docs/guides/develop/agent-skill-bridge/
```

## 6. 模板 Skill 示例

首版 skeleton 用模板 作为最小 Skill，而不是绑定具体业务插件。该 Skill 必须落地为文件包：

```text
skeleton/skills/template/
  SKILL.md
  schema.input.json
  schema.output.json
```

`SKILL.md` 示例：

```md
---
id: powerxplugin.template.basic
name: template
title: 模板管理
provider: com.powerx.plugins.base
version: 1.0.0
description: 创建、查询、更新和删除插件模板
intent_examples:
  - 帮我创建一个名称为视频模板的模板，描述是短视频生成配置，内容是 JSON 配置
  - 查询 ID 为 123 的模板
response_guidance:
  capability_intro:
    - 说明这是模板对象管理能力，不要把它描述成 PowerX Core 的全局能力。
    - 能力介绍可以概括为创建、查询、更新、删除、列表。
  capability_howto:
    - create 和 update 需要 template.name、template.description、template.content。
    - get、update、delete 需要 template_id。
  clarify_params:
    - 只追问缺失字段，不要把缺参当成执行失败。
  skill_execution:
    - 成功时说明模板 ID、名称，以及用户下一步可以查询或更新。
action_required_args:
  create:
    - template.title
    - template.description
    - template.content
  update:
    - template_id
  delete:
    - template_id
  get:
    - template_id
action_optional_args:
  list:
    - q
    - page
    - page_size
slot_mapping:
  template.title:
    labels: ["标题", "名称", "模板标题"]
  template.description:
    labels: ["描述", "用途", "说明"]
  template.content:
    labels: ["内容", "正文", "模板内容"]
pending_task_policy:
  enabled: true
  merge_window_messages: 6
  merge_window_seconds: 900
  confirm_before_execute: false
result_presentation:
  create:
    title: "模板已创建"
    primary_link: "template.detail_path"
    visible_fields:
      - template.id
      - template.title
      - template.detail_path
capability: powerxplugin.template
action_capabilities:
  create: com.powerx.plugins.base.template.create
  get: com.powerx.plugins.base.template.read
  update: com.powerx.plugins.base.template.update
  delete: com.powerx.plugins.base.template.delete
  list: com.powerx.plugins.base.template.list
visibility: tenant
status: active
executor:
  type: capability
  capability: powerxplugin.template
  prepare_capability: com.powerx.plugins.base.template.prepare
  action_map:
    create: com.powerx.plugins.base.template.create
    get: com.powerx.plugins.base.template.read
    update: com.powerx.plugins.base.template.update
    delete: com.powerx.plugins.base.template.delete
    list: com.powerx.plugins.base.template.list
input_schema: ./schema.input.json
output_schema: ./schema.output.json
---

# 模板管理

## When To Use
当用户希望创建、查询、更新、删除模板时使用。

## Instructions
识别用户意图并转换为 action。create/update 时提取 template；get/delete 时提取 template_id。
```

Agent 与 Skill 的回复规范分工：

```text
plugin_agents.persona:
  当前 Agent 是谁、服务谁、边界是什么。

plugin_agents.prompt_seed:
  当前 Agent 如何介绍已绑定能力、如何推荐用户先测试什么、如何按 Skill metadata 处理执行请求。

skills/template/SKILL.md response_guidance:
  模板能力自己的参数、缺参追问、how-to、执行结果表达。
```

同步时 `response_guidance` 会进入：

```text
SKILL.md frontmatter
  -> PluginSkillManifest.response_guidance
  -> plugin_skills.prompt_spec.response_guidance
  -> PowerX skills_registry_records.manifest_json.prompt_spec.response_guidance
  -> PowerX Agent Runtime CapabilityContextItem.response_guidance
```

解析后生成的 manifest 等价于：

```json
{
  "skill_id": "powerxplugin.template.basic",
  "provider": "com.powerx.plugins.base",
  "version": "1.0.0",
  "title": "模板管理",
  "description": "创建、查询、更新和删除插件模板",
  "intent_examples": [
    "帮我创建一个视频模板",
    "查询当前模板",
    "把这个模板状态改成启用"
  ],
  "input_schema": {
    "type": "object",
    "required": ["action"],
    "properties": {
      "action": {"type": "string", "enum": ["create", "get", "update", "delete", "list"]},
      "template_id": {"type": "string"},
      "template": {"type": "object"}
    }
  },
  "executor": {
    "type": "capability",        "capability": "powerxplugin.template"
  },
  "action_required_args": {
    "create": ["template.title", "template.description", "template.content"]
  },
  "pending_task_policy": {
    "enabled": true,
    "merge_window_messages": 6,
    "merge_window_seconds": 900
  },
  "result_presentation": {
    "create": {
      "title": "模板已创建",
      "primary_link": "template.detail_path"
    }
  }
}
```

同步闭环：

```text
SKILL.md Skill Package
        ↓ parse + validate
插件 Skill Plugin Definition
        ↓ sync
PowerX Skill Registry(source=plugin)
        ↓ bind
插件 Agent Plugin Definition
        ↓ sync
PowerX Agent + AgentSkillBinding
        ↓ debug
Agent Chat 调试
```

## 7. MediaX 示例

插件声明：

```json
{
  "skill_id": "mediax.video_rebuilder.cn",
  "provider": "com.powerx.plugin.mediax-studio",
  "version": "1.0.0",
  "title": "视频智能重构",
  "description": "根据视频链接和模板要求创建视频自动化重构任务",
  "intent_examples": [
    "帮我重构这个 shorts",
    "用篮球模板处理这个视频"
  ],
  "input_schema": {
    "type": "object",
    "required": ["urls"],
    "properties": {
      "urls": {"type": "array", "items": {"type": "string"}},
      "template_hint": {"type": "string"}
    }
  },
  "executor": {
    "type": "capability",        "capability": "creation.video_automation.ingest"
  }
}
```

插件 executor 内部可以调用自身业务服务，但该业务接口不作为渠道长期入口。

## 8. 管理页面设计

在 skeleton 管理端新建菜单分类：

```text
PowerX底座能力
├── Agent 管理
├── Skill 管理
└── Agent Chat 调试
```

页面职责：

1. `Skill 管理`：管理插件 Skill Plugin Definition，展示 `powerx_skill_id/sync_status/sync_error/last_sync_at`，提供同步和刷新按钮。
2. `Agent 管理`：管理插件 Agent Plugin Definition，选择已同步 Skill，展示 `powerx_agent_uuid/sync_status/sync_error/last_sync_at`，提供同步和调试入口。
3. `Agent Chat 调试`：只展示已同步成功且 PowerX 侧 active 的 Agent，创建 PowerX Agent Session 并消费 SSE/WS。

所有页面请求必须先到插件 backend；插件前端不得直接访问 PowerX 底座 Admin API。

`初始化模板能力` 的语义：

1. 从 `skills/template/SKILL.md` 读取 Skill Package。
2. 校验 frontmatter、Markdown body、schema、executor 与 checksum。
3. upsert 插件 Skill Plugin Definition。
4. 通过插件 backend proxy 同步到底座 Skill Registry。
5. upsert 插件 Agent Plugin Definition，并绑定刚同步的 PowerX Skill ID。
6. 同步到底座 Agent/Binding。
7. 重复点击必须幂等，不得重复创建本地记录；Gateway 不可用时 fail-fast。

## 9. 验收口径

1. PowerX 能发现插件 Skill 并导入为治理态草稿。
2. PowerX Agent Runtime 命中插件 Skill 后，插件收到完整 Invocation Context。
3. 插件自有 Chat 与 PowerX Web Chat 走同一 Agent Stream 事件语义。
4. 缺少上下文、capability 不匹配、未注册 Skill、插件未启用时全部 fail-fast。
5. trace 可按 `tenant_uuid/plugin_id/skill_id/session_id/trace_id` 串联。
6. 插件自有创建 Skill/Agent Plugin Definition 后，可同步生成 PowerX 侧 Skill/Agent/Binding 并回写映射 ID。
7. Agent Chat 下拉框不展示未同步、同步失败、漂移或 PowerX 侧 disabled 的 Agent。

## 10. Agent Run State Protocol 对齐

PowerXPlugin 不定义独立智能体运行协议。插件调试页、skeleton 示例和 Framework Client 必须消费 PowerX Core 的 Agent Run State Protocol。

标准事件：

```text
agent_run.started
agent_run.response_plan
agent_run.intent_detected
agent_run.plan_created
agent_run.task_status
agent_run.task_started
agent_run.awaiting_params
agent_run.task_completed
agent_run.task_failed
agent_run.final
agent_run.ended
```

Framework Client 输出统一状态：

```text
AgentRunState
  run
  session
  message
  response_plan
  tasks[]
  pending_params[]
  results[]
  errors[]
  trace_links[]
```

插件侧页面要求：

1. `Agent Chat 调试` 必须展示 task 状态卡，不只展示最终 assistant 文本。
2. 缺参时展示 `AgentPendingParamsCard`，字段名称来自 Skill `slot_mapping`，用户可以用自然语言补充。
3. 成功时展示 `AgentTaskResultCard`，链接来自 Skill `result_presentation` 与 capability result。
4. 失败时展示稳定错误码和 trace 入口，不允许空白回复。
5. 点击 trace 必须带上 `tenant_uuid/session_id/message_id/run_id/task_id`，跳到底座 Trace 详情而不是只跳 session 列表。
6. PowerXPlugin 不根据本地判断生成“已创建/已完成”文案；只有收到 `agent_run.task_completed` 且包含真实 result 时，UI 才展示成功状态。

Skill 包要求：

1. 必须声明 `executor.prepare_capability`，由该能力合并业务状态、判断缺参并返回 `ready_to_execute`。
2. `ready_to_execute=true` 时必须返回 `capability_request.capability_id` 与 `capability_request.payload`。
3. 必须声明 action 到 capability 的映射，用于 Skill 自己构造 `capability_request` 与治理校验；Core 不得绕过 prepare 直接按 action_map 执行。
4. 创建、更新、删除等动作必须声明 `action_required_args`。
5. 需要多轮补参的动作必须声明 `pending_task_policy`。
6. 需要跳转详情页的结果必须声明 `result_presentation`，实际链接值由 capability result 返回。

与 A2A 的关系：

1. A2A 是底座多智能体委派能力。
2. Agent Run State Protocol 是插件调试页与底座页面共同消费的 UI 状态协议。
3. 插件只需要按协议展示子 Agent task，不需要实现 Google A2A。
