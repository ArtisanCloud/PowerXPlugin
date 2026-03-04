---
name: nuxt-stores
description: 基于内嵌规则 的 Pinia Store 规范。用于 CRUD Store 创建与资源绑定。
---

# CRUD Nuxt Stores

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `web-admin/app/stores/**` 生成 store。

## 规则（内嵌）

### nuxt_stores.yaml

```yaml
kind: ruleset
name: plugin/crud/frontend/nuxt_stores
version: 1

stores:
  create_pinia: true
  files:
    - target: web-admin/app/stores/useTemplateStore.ts
      template: builtin/nuxt_pinia_crud
      params:
        resource: "templates"

gates.require: [PG-FE-UI-001]
```
