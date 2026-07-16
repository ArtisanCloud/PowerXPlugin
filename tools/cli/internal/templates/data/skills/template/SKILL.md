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
    - get、update、delete 优先使用模板名称定位；不要要求用户输入内部 template_id。
    - delete 在唯一定位到模板后也必须二次确认，确认消息必须包含可点击的模板详情链接。
  clarify_params:
    - 只追问缺失信息，不要把缺参当成执行失败。
    - 追问时使用“标题、描述、内容”等对象字段说法，不要输出 template.title、template.description、template.content 或 JSON/schema 术语。
    - 用户只说“创建模板”时，只追问：请提供这个模板的标题、描述和内容。
  skill_execution:
    - 成功时说明模板标题，以及用户下一步可以查询、更新或删除；内部 ID 只作为链接或调试信息使用。
action_required_args:
  create:
    - template.title
    - template.description
    - template.content
  update:
    - template_ref
    - template.title
    - template.description
    - template.content
  get:
    - template_ref
  delete:
    - template_ref
    - confirmation
  list: []
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
  confirm_before_execute: true
state_contract:
  schema_version: "1.0"
  state_keys:
    template.create:
      action: create
      status_enum:
        - collecting
        - ready
        - awaiting_confirmation
        - executing
        - completed
        - failed
      required_args:
        - template.title
        - template.description
        - template.content
      merge_policy:
        mode: skill_defined
        allow_cross_turn: true
        window_messages: 6
        window_seconds: 900
    template.update:
      action: update
      status_enum:
        - collecting
        - ready
        - awaiting_confirmation
        - executing
        - completed
        - failed
      required_args:
        - template_ref
        - template.title
        - template.description
        - template.content
      merge_policy:
        mode: skill_defined
        allow_cross_turn: true
        window_messages: 6
        window_seconds: 900
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
- 查询模板：根据模板名称定位模板，内部解析为执行所需 ID 后读取详情。
- 更新模板：根据模板名称定位模板，内部解析为执行所需 ID 后修改标题、描述和内容。
- 删除模板：根据模板名称定位模板，内部解析为执行所需 ID 后删除指定模板对象。
- 列表模板：列出模板对象，可按关键词筛选。

## Input Guidance

- `action` 是必要字段，可取 `create`、`get`、`update`、`delete`、`list`。
- `create` 和 `update` 需要标题、描述和内容。内部执行时会映射到 `template.title`、`template.description`、`template.content`。
- 模板对象没有额外类型、分类或业务归属字段；不要要求用户提供这些信息。
- `get`、`update` 和 `delete` 优先提取用户给出的模板名称，写入 `template_ref` 或 `template_name`；不要要求用户输入内部 `template_id`。
- 删除模板必须二次确认。唯一命中时先返回模板名称、详情链接和模板 ID；用户明确确认后才执行删除。
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
- `get`、`update` 和 `delete` 时提取模板名称；只有用户明确给出数字 ID 时才写入 `template_id`。
- 用户回复“确认删除”“确定删除”“yes”等明确确认语义时，写入 `confirmed: true` 或 `confirmation: "确认删除"`。
- `list` 时可以不传 `template_id`。
- 返回简洁、结构化、可审计的执行结果。
