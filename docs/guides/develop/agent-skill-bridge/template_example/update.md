# Template Update Local Agent 调试案例

本文描述 `powerxplugin.template.basic.local` 的更新能力。它根据 `template_id` 更新模板标题、描述和内容，并返回模板详情链接。

## 1. 目标链路

用户输入：

```text
把 ID 为 123 的模板更新为标题合同模板新版，描述是新版合同场景，内容是新版内容
```

最终必须调用：

```text
com.powerx.plugins.base.local.template.update
```

链路：

```text
Web 调试页
  -> Core Agent Runtime
  -> template.prepare
  -> template.update
  -> templateUpdateHandler
  -> TemplateService.Update
  -> 插件数据库 template 表
  -> update.yaml metadata.agent_response
  -> 对话 content + task links
```

## 2. 参数契约

当前 skill 定义要求 update 一次性具备：

```json
{
  "action": "update",
  "template_id": 123,
  "template": {
    "title": "合同模板新版",
    "description": "新版合同场景",
    "content": "新版内容"
  }
}
```

等价扁平 payload：

```json
{
  "action": "update",
  "template_id": 123,
  "name": "合同模板新版",
  "description": "新版合同场景",
  "content": "新版内容"
}
```

必填字段来自 `skeleton/skills/template/SKILL.md`：

```text
template_id
template.title
template.description
template.content
```

不要把缺失字段静默保留旧值作为 Agent 语义。缺字段时 prepare 应进入 `awaiting_params`。

## 3. prepare 输出

成功时：

```json
{
  "status": "completed",
  "ready_to_execute": true,
  "missing_fields": [],
  "capability_request": {
    "capability_id": "com.powerx.plugins.base.local.template.update",
    "payload": {
      "action": "update",
      "template_id": 123,
      "name": "合同模板新版",
      "description": "新版合同场景",
      "content": "新版内容"
    }
  }
}
```

缺参数时：

```json
{
  "status": "awaiting_params",
  "ready_to_execute": false,
  "missing_fields": ["template.content"],
  "message": "请补充要更新的模板 ID，以及内容。"
}
```

## 4. action capability 输出

handler 返回：

```json
{
  "template": {
    "id": "123",
    "name": "合同模板新版",
    "description": "新版合同场景",
    "content": "新版内容",
    "status": "draft"
  }
}
```

`contracts/capabilities/com.powerx.plugins.base.template.update.yaml` 渲染：

```text
已更新模板「合同模板新版」。[查看模板](/templates/crud?template_id=123)
```

## 5. 验收

SQL 验证：

```sql
select id, name, description, content, tenant_uuid, updated_at
from px_com_powerx_plugins_base.template
where id = 123;
```

期望：

```text
name = 合同模板新版
description = 新版合同场景
content = 新版内容
tenant_uuid = 发起用户/租户的 origin_tenant_uuid
```

Agent trace：

```text
skill_id = powerxplugin.template.basic.local
capability_id = com.powerx.plugins.base.local.template.update
status = completed
```

## 6. 常见错误

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 缺参追问 | `template_id/title/description/content` 不完整 | 按追问补齐，不要绕过 prepare |
| 更新到 API key 租户 | origin tenant 没透传 | 检查 `origin_tenant_uuid` 和 `resourceTenantUUIDFromEnvelope` |
| 更新后页面看不到 | 前端当前租户不是记录租户 | 检查 local tenant 和页面 tenant context |
| 返回 `jwt Unauthorized` | registry endpoint 错到 `/api/v1/templates/:id` | 重新同步 capability，endpoint 必须是统一 invoke |

