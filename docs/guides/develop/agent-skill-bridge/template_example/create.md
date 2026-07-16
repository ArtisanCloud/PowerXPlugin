# Template Create Local Agent 调试案例

本文只描述当前正在联调的案例：PowerXPlugin 本地 `.local` 模式下，在插件调试页通过 PowerX Core Agent Runtime 创建一个模板对象。

它不是总规范。总规范见：

- `../README.md`

## 1. 目标链路

用户在 PowerXPlugin 调试页输入：

```text
帮我创建一个标题为测试模板的模板，描述是用于验证插件 CRUD，内容是这是一条测试内容
```

最终必须落到插件本地 backend 的 template create handler，并在插件数据库创建记录。

完整链路：

```text
PowerXPlugin Web 调试页
  -> PowerXPlugin Backend Agent SSE Proxy
  -> PowerX Core /api/v1/agents/stream/sse
  -> PowerX Core Agent Runtime
  -> Skill prepare_capability
  -> Core Capability Invocation
  -> /_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke
  -> PowerXPlugin Local Backend InvokeCapability
  -> templateCreateHandler
  -> 插件数据库 template 表
  -> Core agent_run.task_completed
  -> PowerXPlugin Web 展示执行结果
```

关键判断：

- `plugin_id` 必须是 `com.powerx.plugins.base.local`。
- `skill_id` 必须是 `powerxplugin.template.basic.local`。
- `prepare_capability` 必须是 `com.powerx.plugins.base.local.template.prepare`。
- `create capability` 必须是 `com.powerx.plugins.base.local.template.create`。
- Core 反向调用插件的 path 必须是 `/_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke`。
- 不能调用 `/_p/com.powerx.plugins.base.local/api/v1/templates`。

## 2. 参与组件

| 组件 | 作用 | 关键代码 |
| --- | --- | --- |
| PowerXPlugin Web | 调试页、创建 session、发送 SSE 请求、展示 task trace | `skeleton/web-admin/nuxt/app/pages/_p/com.powerx.plugins.base/admin/agent-skill-bridge/index.vue` |
| PowerXPlugin Backend Agent Proxy | 解析调试页请求，带 Gateway 凭证转发到 PowerX Core Agent SSE | `skeleton/backend/go-gin/internal/transport/http/plugin/agent/routes.go` |
| PowerX Core Agent Runtime | 选择 Agent/Skill，执行 prepare，执行 action capability | `PowerX/backend/internal/server/agent/manager_execute.go` |
| PowerX Core Capability Registry | 根据 capability_id 找 adapter endpoint | `PowerX/backend/internal/service/capability_registry` |
| PowerX Core Plugin Proxy | 把 `/_p/<plugin_id>/...` 代理到 installed 或 local debug host | `PowerX/backend/internal/infra/plugin/manager/router/router.go` |
| PowerXPlugin Integration Invoke | 插件统一能力入口，接收 Core 反向调用 | `skeleton/backend/go-gin/internal/transport/http/integration/handler.go` |
| PowerXPlugin Capability Invoker | 根据 `capabilityId/action` 分发 handler | `skeleton/backend/go-gin/internal/services/integration/capability_invoker.go` |
| Template Service | 真正创建模板并落库 | `skeleton/backend/go-gin/internal/services/admin/templates` |

## 3. 启动前检查

### 3.1 插件必须以 local 身份启动

插件 backend 环境变量必须包含：

```env
POWERX_PLUGIN_REGISTRATION_MODE=local
POWERX_PROXY=1
IAMMode=local
PX_GATEWAY_BASE_URL=http://127.0.0.1:8077/api/v1
PX_GATEWAY_AUTH_SCHEME=apikey
PX_GATEWAY_API_KEY=<tenant integration api key>
```

含义：

- `POWERX_PLUGIN_REGISTRATION_MODE=local` 决定同步出的插件 ID 和 Skill ID 带 `.local`。
- `POWERX_PROXY=1` 表示插件通过 PowerX Core 网关访问底座接口。
- `IAMMode=local` 表示插件本地进程按 local 调试方式处理身份，不是 Core 托管 installed 进程。
- `PX_GATEWAY_API_KEY` 只用于 Plugin -> Core 注册、同步、调试请求，不用于 Core -> Plugin 的 runtime 回调。

### 3.2 Core 必须能看到 local debug host

```bash
curl -sS http://127.0.0.1:8077/__debug/plugins \
  | jq '.apis["com.powerx.plugins.base.local"]'
```

期望：

```json
{
  "basePath": "/api/v1",
  "healthPath": "/healthz",
  "target": "http://127.0.0.1:8078"
}
```

异常判断：

| 结果 | 含义 | 排查 |
| --- | --- | --- |
| `curl: Failed to connect` | 8077 上没有 PowerX Core 进程 | 检查 Core backend 是否启动、端口是否是 8077 |
| `null` | Core 在跑，但 local debug host 未登记 | 重启 PowerXPlugin backend，检查 `POWERX_PLUGIN_REGISTRATION_MODE=local` 和 debug host 注册日志 |
| target 不是插件 backend 端口 | 登记到了错误进程 | 检查插件 backend 实际端口，不要填 Nuxt 端口 |

## 4. 初始化和同步

在 PowerXPlugin Web Admin 点击：

```text
PowerX底座能力 -> Agent 管理 -> 初始化/同步模板智能体
```

这个动作必须完成四件事：

1. 读取 `skeleton/skills/template/SKILL.md`。
2. 以 `.local` 身份写入插件侧 Agent/Skill registry。
3. 同步 Skill 到 PowerX Core。
4. 同步 capability catalog 和 Agent 绑定到 PowerX Core。

同步后检查 PowerX Core DB。

### 4.1 Agent 必须是 local

```sql
select id, uuid, key, owner_plugin_id, status
from public.agents
where key like 'powerxplugin.template.agent%'
order by id desc;
```

期望存在：

```text
key = powerxplugin.template.agent.local
owner_plugin_id = com.powerx.plugins.base.local
```

### 4.2 Skill 必须是 local

```sql
select skill_id,
       manifest_json #>> '{executor,prepare_capability}' as prepare,
       manifest_json #> '{executor,action_map}' as action_map
from public.skills_registry_records
where skill_id like 'powerxplugin.template.basic%';
```

期望：

```text
skill_id = powerxplugin.template.basic.local
prepare = com.powerx.plugins.base.local.template.prepare
action_map.create = com.powerx.plugins.base.local.template.create
```

### 4.3 Capability endpoint 必须是统一 invoke

```sql
with latest as (
  select id, capability_id, tenant_uuid, version, status, updated_at
  from public.capability_registry_registrations
  where capability_id in (
    'com.powerx.plugins.base.local.template.prepare',
    'com.powerx.plugins.base.local.template.create'
  )
  order by capability_id, version desc
),
latest_per_capability as (
  select distinct on (capability_id)
    id, capability_id, tenant_uuid, version, status, updated_at
  from latest
  order by capability_id, version desc
)
select
  l.capability_id,
  l.version,
  l.status,
  e.adapter_id,
  e.transport_type,
  e.endpoint,
  e.service_ref,
  e.weight,
  e.is_active,
  e.labels,
  e.updated_at
from latest_per_capability l
join public.capability_registry_adapter_endpoints e
  on e.registration_id = l.id
order by l.capability_id, e.adapter_id;
```

期望：

```text
version = 当前最新版本
status = published
endpoint = /_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke
labels.method = POST
```

不允许出现：

```text
/_p/com.powerx.plugins.base.local/api/v1/templates
```

判断规则：

- 只看最新 `capability_registry_registrations.version` 对应的 adapter。
- `capability_registry_adapter_endpoints` 会保留历史 registration version 的 adapter，直接按 `capability_id` 全表查询会看到旧 endpoint，这是历史快照，不等于当前 runtime 会使用它。
- PowerX Core runtime 读取 registration 时使用 latest version：`GetLatest -> order by version desc -> preload adapters`。
- 如果最新 version 的 REST adapter 还是 `/api/v1/templates`，说明 Core registry 当前生效版本仍是旧能力描述。需要重启 PowerXPlugin backend 后重新初始化/同步模板智能体；必要时重启 PowerX Core 清缓存。
- 历史 adapter 可以通过 Core 的 stale snapshot prune/硬删除策略清理，但清理不是判断当前 runtime 是否正确的前提。

## 5. 正常请求逐步发生什么

### Step 1: Web 调试页发送 Agent SSE

浏览器请求插件 backend 的 Agent proxy。插件 backend 再请求 PowerX Core：

```text
POST /api/v1/agents/stream/sse
```

插件日志应看到类似阶段：

```text
gateway stream agent sse resolve session done
gateway stream agent sse request core
```

如果这里失败：

| 错误 | 问题点 |
| --- | --- |
| session 不存在 | 调试页 session/agent 选择不对，或复用了已污染旧 session |
| gateway credential failed | 插件访问 Core 的 `PX_GATEWAY_API_KEY` 或 STS 配置不对 |
| Core 返回 401/403 | Plugin -> Core 方向权限不对，不是 Core -> Plugin 反向能力问题 |

### Step 2: Core 选择 local Agent 和 Skill

Core 必须选中：

```text
agent.owner_plugin_id = com.powerx.plugins.base.local
skill_id = powerxplugin.template.basic.local
```

如果 trace 里是：

```text
com.powerx.plugins.base
powerxplugin.template.basic
```

说明当前 session/agent 仍然是 installed，不是 local。重新初始化 `.local` Agent 后新建 session 测试。

### Step 3: Core 调用 prepare capability

Core 先调用：

```text
capability_id = com.powerx.plugins.base.local.template.prepare
path = /_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke
```

插件 handler：

```text
templatePrepareHandler
```

prepare 的成功输出必须包含：

```json
{
  "ready_to_execute": true,
  "status": "completed",
  "missing_fields": [],
  "capability_request": {
    "capability_id": "com.powerx.plugins.base.local.template.create",
    "payload": {
      "action": "create",
      "name": "测试模板",
      "description": "用于验证插件 CRUD",
      "content": "这是一条测试内容"
    }
  }
}
```

异常判断：

| 错误 | 所在步骤 | 原因 |
| --- | --- | --- |
| `skill prepare failed: status=accepted` | prepare 返回协议 | prepare handler 返回了非 prepare 状态，Core 不认为可执行 |
| `skill prepare completed without ready_to_execute=true` | prepare 返回协议 | prepare payload 结构不符合 Core 期待 |
| 缺参追问但参数已提供 | prepare 参数抽取/合并 | 检查 `templatePreparePayload`、`mergeTemplatePrepareState`、slot mapping |
| `capability_request.capability_id` 不是 `.local` | local 重写 | 检查 `localizeCapabilityIDForRequest` 和 Skill manifest rewrite |

### Step 4: Core 调用 create capability

Core 日志应出现：

```text
[agent.skill.invoke] skill_id=powerxplugin.template.basic.local action=create capability_id=com.powerx.plugins.base.local.template.create plugin_id=com.powerx.plugins.base.local
```

随后 Core plugin proxy 应调用：

```text
POST /_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke
```

请求 envelope 的 body 由 Core 包装，核心结构：

```json
{
  "body": {
    "capabilityId": "com.powerx.plugins.base.local.template.create",
    "action": "create",
    "preferredProtocol": "agent",
    "payload": {
      "action": "create",
      "name": "测试模板",
      "description": "用于验证插件 CRUD",
      "content": "这是一条测试内容"
    },
    "metadata": {
      "capability_id": "com.powerx.plugins.base.local.template.create"
    }
  }
}
```

插件侧分发：

```text
InvokeCapability
  -> CapabilityInvoker.Invoke
  -> HandlerByCapability("com.powerx.plugins.base.local.template.create")
  -> normalizeLocalCapabilityID
  -> templateCreateHandler.Handle
```

异常判断：

| 错误 | 所在步骤 | 原因 | 修复方向 |
| --- | --- | --- | --- |
| `jwt Unauthorized`，path 是 `/api/v1/templates` | Core registry endpoint | action capability descriptor/DB 仍是页面 REST | 重新同步 capability descriptor，确认 adapter endpoint 是 `/integration/capabilities/invoke` |
| `root privileges required` | 调到了 installed 或错误路由 | 不是 runtime invoke，或 session 仍是 installed | 检查 plugin_id/skill_id/path 是否带 `.local` |
| `capabilityId is required` | Core envelope 包装 | endpoint 没被识别为 `/integration/capabilities/invoke` 或 payload 结构未包 body | 检查 capability adapter endpoint |
| `context deadline exceeded` 且打到 `/api/v1/tenant/invocations` | 插件侧 fallback/gateway 循环 | local handler 没接管，落到 fallback 又打回 Core | 检查 framework/local invoker 是否注册 create capability |
| `404 page not found`，path 是 `/api/v1/integration/capabilities/invoke` 但没有 `/_p` | endpoint 归一化错误 | Core 没走 plugin proxy 或 registry endpoint 缺 plugin prefix | 检查 capability registry adapter endpoint |

### Step 5: 插件创建模板并返回

`templateCreateHandler` 读取 payload：

```json
{
  "name": "测试模板",
  "description": "用于验证插件 CRUD",
  "content": "这是一条测试内容"
}
```

然后调用：

```text
TemplateService.Create(ctx, name, description, content)
```

handler 返回结构化业务结果：

```json
{
  "id": "123",
  "status": "draft",
  "template": {
    "id": "123",
    "name": "测试模板",
    "description": "用于验证插件 CRUD",
    "content": "这是一条测试内容"
  }
}
```

插件侧 renderer 根据 `contracts/capabilities/com.powerx.plugins.base.template.create.yaml` 的 `metadata.agent_response` 追加对话可展示结果：

```json
{
  "content": "已创建模板「测试模板」，状态为 draft。[查看模板](/templates/crud?template_id=123)",
  "links": [
    {
      "id": "123",
      "kind": "template",
      "label": "查看模板「测试模板」",
      "href": "/templates/crud?template_id=123"
    }
  ]
}
```

注意：

- `content/links` 的话术定义属于 capability contract，不应硬编码在 handler。
- handler 只负责返回 `id/status/template` 等结构化业务字段。
- 如果 Agent 调用的 capability 缺少 `metadata.agent_response.content`，应该明确失败，不做默认话术兜底。

验证插件 DB：

```sql
select id, name, description, content, tenant_uuid, created_at
from px_com_powerx_plugins_base.template
where name = '测试模板'
order by id desc
limit 10;
```

如果 Core 显示成功但 DB 没记录，这是 UI 或 task 状态判断错误。只能以 `agent_run.task_completed` 和真实 DB 记录作为成功依据。

## 6. 日志怎么跟

### 6.1 PowerX Core 日志

Core 侧主要看：

```bash
rg -n "agent.skill.invoke|PROXY-BACKEND|/_p/com.powerx.plugins.base.local|<trace_id>" \
  /private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX/backend/logs -S
```

重点判断：

- `[agent.skill.invoke]` 的 `skill_id/action/capability_id/plugin_id/payload` 是否正确。
- `[API-IN]` 或 proxy 日志 path 是否是 `/_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke`。
- `[PROXY-BACKEND-ERR]` 的 `upstream_status/upstream_body` 是插件返回的真实错误。

### 6.2 PowerXPlugin 日志

插件侧主要看：

```bash
rg -n "gateway stream agent sse|LOCAL-INVOKE|InvokeCapability|template.create|<trace_id>" \
  /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/logs -S
```

重点判断：

- 是否收到 Core 反向调用。
- 收到的 `capability_id` 是否是 `.local.template.create`。
- 是否进入 `templateCreateHandler`。
- 是否有 service/database error。

### 6.3 Agent Debug Trace

Core agent debug 文件通常在：

```text
PowerX/backend/logs/agent_debug/<date>/trace-*.json
```

看这些字段：

```text
trace_id
session_id
message_id
task_id
skill_id
capability_id
node_ref
status
error
```

如果同一个 `message_id` 反复重发，可能会看到旧 task 残留。修复链路后建议新建 session 或发送一条新的用户消息验证。

## 7. 一眼判断错误在哪一层

| 现象 | 层级 | 直接判断 |
| --- | --- | --- |
| `curl 8077 failed` | Core 进程 | PowerX Core 没在该端口运行 |
| `__debug/plugins` 返回 `null` | local debug host | 插件 backend 没登记到 Core |
| trace 里 `plugin_id=com.powerx.plugins.base` | Agent/Skill 选择 | 选中 installed，不是 local |
| prepare 通过但 create 打 `/api/v1/templates` | capability registry | create adapter endpoint 是旧的页面 REST |
| `/api/v1/templates` 返回 `jwt Unauthorized` | endpoint 配置 | Agent 错打页面 API，不该放开 JWT |
| `/integration/capabilities/invoke` 返回 `capabilityId is required` | envelope | Core 没按统一 invoke envelope 包装 |
| 请求打回 `/api/v1/tenant/invocations` 超时 | 插件 invoker | local handler 未接管，走了 fallback |
| `root privileges required` | 路由/权限 | runtime invoke 误走管理调试接口或 installed 路径 |
| `task_completed` 没有但 UI 显示完成 | 前端展示 | UI 把 final/ended 当成业务完成 |

## 8. 本案例的正确验收

一次 create 测试通过必须同时满足：

1. `__debug/plugins` 有 `com.powerx.plugins.base.local`。
2. Core DB 的 Agent/Skill 都是 `.local`。
3. Core DB 的 create adapter endpoint 是 `/_p/com.powerx.plugins.base.local/api/v1/integration/capabilities/invoke`。
4. Core 日志 `[agent.skill.invoke]` 的 capability 是 `com.powerx.plugins.base.local.template.create`。
5. Core proxy path 没有 `/api/v1/templates`。
6. 插件日志收到 `LOCAL-INVOKE` 或 `InvokeCapability`。
7. 插件进入 `templateCreateHandler`。
8. 插件 DB 出现模板记录。
9. Core 发出 `agent_run.task_completed`。
10. 前端复制错误按钮在失败时能复制 trace payload。

只满足前端最终回复不算通过。
