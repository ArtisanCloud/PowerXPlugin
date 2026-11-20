---
name: nuxt-pages
description: 基于内嵌规则 的 Nuxt 页面规范。用于新增 CRUD 列表页/详情页与主题对齐。
---

# CRUD Nuxt Pages

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `web-admin/app/pages/**` 生成或更新页面。

## 核对点

- 列表页/详情页按模板生成。
- 主题色阶与明暗模式对齐。

## 规则（内嵌）

### nuxt_pages.yaml

```yaml
kind: ruleset
name: plugin/crud/frontend/nuxt_pages
version: 1

pages:
  resource: template
  files:
    - target: web-admin/app/pages/templates/index.vue
      template: builtin/nuxt_page_table
      params:
        title: "Templates"
        columns: ["name","description","updated_at"]
        actions: ["create","edit","delete","view"]
    - target: web-admin/app/pages/templates/[id].vue
      template: builtin/nuxt_page_detail
      params:
        title: "Template Detail"

theme:
  description: |
    浅色模式：使用浅色面板（如 `bg-white/95`）+ 深色前景（不超过 `text-gray-900`），禁止在浅色面板上放白色文字或图标。
    深色模式：完全对齐模板 CRUD 组件，面板使用深海军蓝 (`#0f192a`~`#111a2b`) 并辅以低饱和边框；标题 `#fdfcff`，正文/描述 `#d6e2ff` 附近，不得出现深灰文字。
    图标/按钮：所有状态图标（参考租户配置左侧纵向卡片）在浅色模式用 `text-primary-600`；暗色模式需改为高亮浅色（`#fdfcff` 或 `#93c5fd`），不可保留灰黑图标。
    页面开发必须参照模板 CRUD 组件的色阶搭配（面板、按钮、表格 hover 状态等），保证在明暗主题下对比度一致。

gates.require: [PG-FE-UI-001, PG-FE-RBAC-001]
```
