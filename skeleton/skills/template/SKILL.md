---
id: powerxplugin.template.basic
name: template
title: 模板对象基础能力
provider: com.powerx.plugins.base
version: 1.0.0
description: 管理 PowerXPlugin 的基础模板对象。该对象仅包含标题、描述和内容，用于开发者验证插件侧 CRUD、能力注册和 Agent 调用链路。
intent_examples:
  - 帮我创建一个标题为测试模板的模板，描述是用于验证插件 CRUD，内容是这是一条测试内容
  - 查询 ID 为 123 的模板
  - 把 ID 为 123 的模板内容更新成新的测试内容
  - 列出所有模板
  - 删除 ID 为 123 的模板
response_guidance:
  capability_intro:
    - 说明这是 PowerXPlugin 的基础模板对象能力，不是媒体、内容生产或具体业务模板能力。
    - 明确模板对象只有标题、描述和内容三个核心字段。
    - 能力介绍只概括创建、查询、更新、删除、列表。
  capability_howto:
    - create 和 update 需要用户提供标题、描述和内容。
    - 不要询问额外类型、分类或业务归属；这个对象没有这些字段。
    - get、update、delete 需要 template_id。
  clarify_params:
    - 只追问缺失信息，不要把缺参当成执行失败。
    - 追问时使用“标题、描述、内容”等对象字段说法，不要输出 template.title、template.description、template.content 或 JSON/schema 术语。
    - 用户只说“创建模板”时，只追问：请提供这个模板的标题、描述和内容。
  skill_execution:
    - 成功时说明模板 ID、标题，以及用户下一步可以查询、更新或删除。
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
  action_map:
    create: com.powerx.plugins.base.template.create
    get: com.powerx.plugins.base.template.read
    update: com.powerx.plugins.base.template.update
    delete: com.powerx.plugins.base.template.delete
    list: com.powerx.plugins.base.template.list
  timeout_ms: 30000
  async_supported: true
  risk_level: low
input_schema: ./schema.input.json
output_schema: ./schema.output.json
---

# 模板对象基础能力

## Purpose

管理 PowerXPlugin 的基础模板对象。模板对象是插件提供给开发者使用的最小 CRUD 示例对象，只包含标题、描述和内容三个核心字段。它用于验证插件侧数据模型、能力注册、Agent 参数收集和能力调用链路，不代表任何具体业务领域。

## When To Use

当用户希望创建、查询、更新、删除或列出 PowerXPlugin 模板对象时使用。用户可能会说“创建模板”“查询模板”“删除模板”“列出模板”，也可能直接给出标题、描述和内容。

## Capability Summary

- 创建模板：根据用户给出的标题、描述和内容创建模板对象。
- 查询模板：根据 `template_id` 读取模板详情。
- 更新模板：根据 `template_id` 修改标题、描述和内容。
- 删除模板：根据 `template_id` 删除指定模板对象。
- 列表模板：列出模板对象，可按关键词筛选。

## Input Guidance

- `action` 是必要字段，可取 `create`、`get`、`update`、`delete`、`list`。
- `create` 和 `update` 需要标题、描述和内容。内部执行时会映射到 `template.title`、`template.description`、`template.content`。
- 模板对象没有额外类型、分类或业务归属字段；不要要求用户提供这些信息。
- `get`、`update` 和 `delete` 通常需要 `template_id`。
- `list` 可以不传 `template_id`，但可以携带关键词筛选条件。

## Conversation Guidance

- 用户询问“你能做什么”时，说明这是基础模板对象管理能力，并解释支持创建、查询、更新、删除和列表。
- 用户询问“怎么用”时，说明只需要提供标题、描述和内容，并给出自然语言示例。
- 用户要求执行但缺少必要参数时，先用自然语言追问缺失信息，不要直接失败。
- 用户只说“创建模板”但没有给标题、描述或内容时，应追问：“请提供这个模板的标题、描述和内容。”
- 不要追问额外类型、分类、业务归属或其他对象字段之外的信息。
- 不要暴露内部 executor path、schema 原文或调试字段。
- 不要要求用户必须输入 JSON；JSON 只是内部结构化格式。

## Instructions

- 识别用户对模板对象的管理意图，并转换为结构化 `action`。
- `create` 和 `update` 时提取 `template.title`、`template.description`、`template.content`。
- 如果用户说“名称”，可视为“标题”。
- `get` 和 `delete` 时提取 `template_id`。
- `list` 时可以不传 `template_id`。
- 返回简洁、结构化、可审计的执行结果。
