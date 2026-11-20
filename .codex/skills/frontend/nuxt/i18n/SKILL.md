---
name: nuxt-i18n
description: 基于内嵌规则 的 Nuxt i18n 规范。用于新增/维护 CRUD 文案字典。
---

# CRUD Nuxt i18n

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `web-admin/i18n/locales/{en,zh}.json` 更新字典。

## 规则（内嵌）

### nuxt_i18n.yaml

```yaml
kind: ruleset
name: plugin/crud/frontend/nuxt_i18n
version: 1

i18n:
  locales:
    - { code: "en", file: "en.json" }
    - { code: "zh", file: "zh.json" }
  files:
    - target: web-admin/i18n/locales/en.json
      template: builtin/i18n_dict_minimal
      params:
        entries:
          templates: { title: "Templates", create: "Create", edit: "Edit", delete: "Delete", detail: "Detail" }
    - target: web-admin/i18n/locales/zh.json
      template: builtin/i18n_dict_minimal
      params:
        entries:
          templates: { title: "模板", create: "新建", edit: "编辑", delete: "删除", detail: "详情" }

gates.require: [PG-FE-UI-001]
```
