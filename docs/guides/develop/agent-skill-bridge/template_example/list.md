# Template List Local Agent 调试案例

本文描述 `powerxplugin.template.basic.local` 的列表查询能力。它用于按关键词和分页查询插件本地 `template` 表，并把列表入口返回到对话和任务卡片。

## 1. 目标链路

用户在 PowerXPlugin 调试页输入：

```text
列出模板，关键词是合同，第一页，每页 20 条
```

最终必须调用：

```text
com.powerx.plugins.base.local.template.list
```

完整链路：

```text
PowerXPlugin Web 调试页
  -> PowerXPlugin Backend Agent SSE Proxy
  -> PowerX Core Agent Runtime
  -> com.powerx.plugins.base.local.template.prepare
  -> com.powerx.plugins.base.local.template.list
  -> PowerXPlugin templateListHandler
  -> TemplateService.List
  -> 插件数据库 template 表
  -> contract metadata.agent_response
  -> 对话 content + task links
```

## 2. 参数契约

list action 没有必填字段，支持三个可选结构化参数：

```json
{
  "action": "list",
  "q": "合同",
  "page": 1,
  "page_size": 20
}
```

字段规则：

| 字段 | 含义 | 规则 |
| --- | --- | --- |
| `q` | 模糊搜索关键字 | 可为空 |
| `page` | 页码 | 最小 1；未传时 handler 使用 1 |
| `page_size` | 每页数量 | 最小 1，最大 100；未传时 handler 使用 10 |

也支持嵌套写法：

```json
{
  "action": "list",
  "list": {
    "q": "合同",
    "page": 1,
    "page_size": 20
  }
}
```

不允许从自由文本里兜底解析结构化分页字段。分页参数必须来自 skill 输出的结构化 payload 或 pending state。

## 3. prepare 输出

`templatePrepareHandler` 应返回：

```json
{
  "status": "completed",
  "ready_to_execute": true,
  "missing_fields": [],
  "capability_request": {
    "capability_id": "com.powerx.plugins.base.local.template.list",
    "payload": {
      "action": "list",
      "q": "合同",
      "page": 1,
      "page_size": 20
    }
  }
}
```

如果 prepare 只返回：

```json
{ "action": "list" }
```

说明 `q/page/page_size` 没有从 prepare payload/state 透传到 list capability。检查：

- `templatePreparePayload`
- `mergeTemplatePrepareState`
- `templateCapabilityPayload("list")`
- `skeleton/skills/template/SKILL.md` 的 `action_optional_args.list`

## 4. action capability 输出

handler 返回结构化结果：

```json
{
  "items": [
    {
      "id": "123",
      "name": "合同模板",
      "description": "合同场景",
      "content": "..."
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 1
  }
}
```

插件侧 renderer 根据 `contracts/capabilities/com.powerx.plugins.base.template.list.yaml` 生成：

```json
{
  "content": "已查询到 1 个模板。\n- [合同模板](/templates/crud?template_id=123)（draft）\n[查看模板列表](/templates/crud)",
  "links": [
    {
      "kind": "template_list",
      "label": "查看模板列表",
      "href": "/templates/crud"
    }
  ]
}
```

## 5. 验收

数据库验证：

```sql
select id, name, description, tenant_uuid, created_at
from px_com_powerx_plugins_base.template
where name ilike '%合同%' or description ilike '%合同%' or content ilike '%合同%'
order by id desc
limit 20;
```

Agent trace 必须满足：

```text
skill_id = powerxplugin.template.basic.local
capability_id = com.powerx.plugins.base.local.template.list
status = completed
```

最终对话必须包含业务内容，不应出现：

```text
任务已执行完成（skill=..., protocol=..., status=...）
```

任务卡片应包含：

```text
查看模板列表 -> /templates/crud
```

主回复中应列出查询到的业务对象，至少包含前几条模板名称和对应 `/templates/crud?template_id=...` 链接。只回复“已查询到 N 个模板”不算合格。

## 6. 常见错误

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 查询只返回第一页 10 条 | `page/page_size` 没透传 | 检查 prepare payload 和 `templateCapabilityPayload("list")` |
| `q` 不生效 | prepare 未传 `q` 或 service List 查询字段不匹配 | 看 Core `[agent.skill.invoke] payload` 和插件 handler payload |
| 仍然调用 installed capability | session/agent 选错 | 确认 `skill_id` 和 `plugin_id` 都带 `.local` |
| 返回内部完成文案 | contract 没有 `metadata.agent_response.content` 或 result 未带到 Core | 检查 capability YAML 和 Core task end payload |
