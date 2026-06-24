# Agent Skill Bridge

Agent Skill Bridge lets a plugin expose source Skill definitions to PowerX Agent Runtime. A Skill is an Agent-scoped semantic package; it is not a standalone business execution API. Business execution must resolve from Skill action to PowerX capability invocation.

## Contract

- `GET /api/v1/plugin/skills`: list plugin source Skill manifests.
- `GET /api/v1/plugin/skills/:skill_id/schema`: return input/output schema.

Plugins must not expose PowerX Capability Invocation as the standard business path. The standard execution path is:

```text
PowerX Agent Runtime
  -> bound Skill
  -> action_capabilities[action]
  -> capability_id
  -> PowerX Capability Invocation
  -> plugin capability handler
```

Capability handlers must trust PowerX-provided invocation context instead of deriving tenant or user identity from input payloads.

## Skill Package Source Format

Plugin Skills must be authored as `SKILL.md` directory packages. A Go manifest is only the parsed runtime object, not the long-term source format.

Minimum package:

```text
skills/<skill_id>/
  SKILL.md
```

Recommended package:

```text
skills/<skill_id>/
  SKILL.md
  schema.input.json
  schema.output.json
  executor.yaml
  scripts/
  references/
  assets/
```

`SKILL.md` must contain YAML frontmatter and Markdown instructions:

```md
---
id: powerxplugin.template.basic
name: template
title: 模板管理
provider: com.powerx.plugins.base
version: 1.0.0
description: 创建、查询、更新和删除插件模板
capability: powerxplugin.template
visibility: tenant
status: active
executor:
  type: capability
  capability: powerxplugin.template
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
识别用户意图并转换为 action。PowerX Core 会将 action 映射为 capability_id，并通过 Capability Invocation 执行。
```

Framework loader rules:

- Parse frontmatter into `PluginSkillManifest`.
- Keep Markdown body as prompt/instruction source.
- Resolve schema/executor references only inside the package directory.
- Compute package checksum over `SKILL.md`, schemas, executor config, scripts, references, and assets.
- Save parsed snapshots into Plugin Skill Plugin Definition and sync them to PowerX.
- Fail fast when required frontmatter, Markdown body, schema, executor, or checksum validation fails.
- Require `executor.type=capability` and `executor.action_map` / `action_capabilities`.

## Go Registration

```go
pkg := skills.LoadPackage("skills/template")
manifest := pkg.Manifest
reg := skills.NewRegistry()
reg.MustRegisterManifest(manifest)
skills.NewHTTPAdapter(reg).RegisterRoutes(router.Group("/api/v1/plugin/skills"))
```

## Local Chat

The skeleton page at `/_p/com.powerx.plugins.base/admin/agent-skill-bridge` is a PowerX Agent stream client. It must call the plugin backend proxy first, and the plugin backend calls PowerX Agent Session/Stream APIs.

Required request chain:

```text
PowerXPlugin Web
  -> Plugin Backend Proxy
  -> PowerX Agent Session / Stream API
  -> PowerX Agent Runtime
  -> PowerX Capability Invocation
  -> Plugin capability handler
```

The page must not call plugin business APIs to simulate intelligent tasks, and must not call PowerX Admin/Agent APIs directly from the browser.

## Plugin Agent/Skill Plugin Registry

Plugins that provide Agent/Skill management pages must store local plugin records and sync them to PowerX.

Plugin Skill Plugin Definition stores:

- `plugin_skill_id`
- `powerx_skill_id`
- `manifest`
- `prompt_spec`
- `input_schema`
- `output_schema`
- `executor`
- `capability`
- `sync_status`
- `sync_error`
- `last_sync_at`

Plugin Agent Plugin Definition stores:

- `plugin_agent_id`
- `powerx_agent_uuid`
- `tenant_uuid`
- `agent_key`
- `name`
- `description`
- `persona`
- `prompt_seed`
- `model_profile_ref`
- `plugin_skill_ids`
- `powerx_skill_ids`
- `sync_status`
- `sync_error`
- `last_sync_at`

PowerX is still the runtime authority. Local plugin records are development/source records only.

### Agent Prompt Contract

Agent prompt data is split by responsibility. Plugin developers must not put all instructions into one opaque prompt field.

| Layer | Plugin source | PowerX runtime storage | Purpose |
| --- | --- | --- | --- |
| Agent identity | `plugin_agents.persona` | `agents.persona` | Defines who the Agent is, who it serves, and its speaking identity. |
| Agent behavior seed | `plugin_agents.prompt_seed` | `agents.prompt_seed` | Defines default behavior policy for the Agent, such as how to explain capabilities, clarify parameters, and choose bound Skills. |
| Skill instructions | `plugin_skills.prompt_spec` and `SKILL.md` body | `skills_registry_records.manifest_json.prompt_spec` | Defines how one Skill should be understood and invoked. |
| Runtime final prompt | Not stored as a static plugin field | Built dynamically by PowerX Agent Runtime | Combines platform rules, Agent persona, prompt seed, bound Skill metadata, session context, response plan, and current user message. |

### Response Guidance Contract

PowerXPlugin does not own the final wording logic. It provides Agent/Skill guidance as data, and PowerX Core composes it during Agent Runtime.

Merge order in PowerX Core:

```text
Core generic runtime rules
  -> Agent persona
  -> Agent prompt_seed
  -> Skill response_guidance
  -> ResponsePlan.answer_requirements
```

Responsibilities:

| Layer | Owner | What belongs here | What must not be here |
| --- | --- | --- | --- |
| Core runtime rules | PowerX Core | Security boundaries, no hidden IDs, no unbound capabilities, response mode constraints | Plugin-specific fields or business wording |
| Agent persona | Plugin Agent Definition | Agent identity, audience, speaking boundaries | Skill parameter validation |
| Agent prompt_seed | Plugin Agent Definition | How this Agent introduces capabilities and chooses/uses bound Skills | Full Skill schema or executor implementation details |
| Skill response_guidance | `skills/<name>/SKILL.md` | Skill-specific usage, missing-field questions, execution-result wording | Tenant/user/session identity or permission logic |

`SKILL.md` frontmatter supports:

```yaml
response_guidance:
  general:
    - 不要输出内部 executor path 或 schema 原文。
  capability_intro:
    - 说明这个 Skill 面向的业务对象和可用动作。
  capability_howto:
    - 说明必要参数、可选参数和自然语言示例。
  clarify_params:
    - 只追问缺失字段，不要把缺参当成执行失败。
  skill_execution:
    - 成功时说明业务结果和下一步。
  error_explain:
    - 把错误转成用户可操作的修正建议。
```

Storage and sync path:

```text
skills/<name>/SKILL.md response_guidance
  -> PluginSkillManifest.response_guidance
  -> plugin_skills.prompt_spec.response_guidance
  -> PowerX skills_registry_records.manifest_json.prompt_spec.response_guidance
  -> PowerX ToolCallCandidate.ResponseGuidance
  -> PowerX CapabilityContextItem.ResponseGuidance
  -> Final Response context
```

Strict rules:

- Do not put template-specific or business-specific wording in PowerX Core runtime code.
- Do not put Skill field rules into Agent `persona`.
- Use Agent `prompt_seed` for Agent-level behavior policy only.
- Use Skill `response_guidance` and `input_schema` for capability-level parameter, how-to, clarification, and result wording.
- Missing `response_guidance` is allowed only when the Skill has no special response behavior; it must not be replaced by hidden Core business rules.

Required API mapping:

```text
plugin_agents.persona
  -> PowerX Agent Admin body.persona
  -> agents.persona

plugin_agents.prompt_seed
  -> PowerX Agent Admin body.promptSeed
  -> agents.prompt_seed
```

Strict rules:

- Plugin Agent APIs must use `persona` and `prompt_seed`.
- PowerX Agent Admin sync must use `persona` and `promptSeed`.
- Do not use `system_prompt` for Agent Registry.
- Do not hide Agent prompt fields under `meta`, `parameters`, or legacy compatibility payloads.
- Do not put Skill execution instructions into `persona`; put them in `SKILL.md` / `prompt_spec`.
- Do not put tenant, user, or session identity into prompt fields; those come from PowerX invocation context.

Recommended content split:

```text
persona:
  Define the Agent's domain identity, audience, and speaking boundaries.

prompt_seed:
  Define how this Agent handles capability introductions, how-to questions,
  execution requests, missing parameters, and failure explanations.

SKILL.md body / prompt_spec:
  Define when to use the Skill, how to map user intent to action/input,
  required fields, validation, and executor behavior.
```

For the skeleton template Agent:

```text
persona:
  你是 PowerXPlugin 模板对象管理助手，服务对象是插件开发者和插件管理员。

prompt_seed:
  当用户询问你是谁或能做什么时，请以模板对象管理助手身份回答，并只基于当前已绑定 Skill 的真实 metadata 介绍能力。
  能力介绍应先说明服务对象，再概括模板创建、查询、更新、删除和列表；推荐先测试创建模板。
  用户要求执行时，按 Skill response_guidance 和 input_schema 判断缺参并追问，参数足够后调用绑定 Skill。
```

Plugin Skill Plugin Definition must preserve Skill Package provenance:

- `source_format=skill_package`
- `package_path`
- `package_checksum`
- `raw_markdown`
- `frontmatter_json`
- `body_markdown`

## Sync Flow

Skill sync:

```text
Create Plugin Skill Plugin Definition
  -> Plugin Backend saves manifest/schema/executor/capability
  -> Plugin Backend calls PowerX Skill Registry API
  -> PowerX creates or updates source=plugin Skill
  -> Plugin Backend saves powerx_skill_id and sync status
```

Agent sync:

```text
Create Plugin Agent Plugin Definition
  -> Select synced PowerX Skill IDs
  -> Plugin Backend calls PowerX Agent Admin API
  -> PowerX creates or updates Agent + AgentSkillBinding
  -> Plugin Backend saves powerx_agent_uuid and sync status
```

Fail-fast rules:

- Unsynced Skill cannot be bound to a runnable Agent.
- Unsynced, failed, drifted, or disabled Agent cannot appear in the Agent Chat runnable dropdown.
- Sync failures must save `sync_error.code`, `sync_error.message`, and `trace_id`.
- Do not fall back to a Plugin Agent runtime.

## Basic Agent Q&A Test Flow

Use this flow after changing Agent prompt metadata or Skill packages.

1. Run plugin migration when Agent Registry schema changes:

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
make migrate
```

2. Restart the PowerXPlugin backend with the latest code.

3. Restart PowerX Core backend when Core Agent Runtime or Agent Admin contract changed.

4. In PowerXPlugin Web Admin, open:

```text
PowerX底座能力 -> Agent 管理 -> 初始化模板能力
```

This is an idempotent upsert. It updates local `plugin_skills/plugin_agents`, syncs Skill metadata to PowerX, then syncs Agent metadata and bindings to PowerX.

5. Verify PowerX Core received the latest Agent metadata:

```sql
select key, name, description, persona, prompt_seed
from agents
where key = 'powerxplugin.template.agent';
```

6. Verify PowerX Core received the latest Skill metadata:

```sql
select skill_id, status, is_latest_published, manifest_json->>'description' as description
from skills_registry_records
where skill_id = 'powerxplugin.template.basic'
order by id desc
limit 1;
```

7. Open:

```text
PowerX底座能力 -> Agent Chat 调试
```

Select the synced template Agent and ask:

```text
你是什么智能体？你能做什么？请只列出你已绑定的能力。
```

Expected behavior:

- The answer describes the PowerXPlugin template object management Agent.
- The listed capability is the bound template Skill.
- It does not list unrelated abilities such as data extraction, format conversion, or global PowerX platform tools.

8. Test how-to and execution paths:

```text
第一个能力怎么用？
帮我创建一个模板
查询模板 template-001
```

Expected behavior:

- How-to questions explain required information and natural-language examples.
- Execution requests map to `create/get/update/delete/list`.
- Missing required fields trigger clarification instead of hallucinated execution.

## Skeleton Menu

Skeleton should expose these pages under `PowerX底座能力`:

- `Agent 管理`
- `Skill 管理`
- `Agent Chat 调试`

`Agent Chat 调试` may only list agents that have a valid `powerx_agent_uuid`, `sync_status=synced`, and an active PowerX status.

## Delegated Auth

Delegated deployments require:

- `PX_GATEWAY_BASE_URL`
- `PX_GATEWAY_AUTH_SCHEME=bearer`
- `POWERX_STS_CLIENT_ID`
- `POWERX_STS_CLIENT_SECRET`

The Agent client rejects legacy `PX_TOOL_TOKEN` and `PX_GATEWAY_API_KEY` as delegated credentials.
