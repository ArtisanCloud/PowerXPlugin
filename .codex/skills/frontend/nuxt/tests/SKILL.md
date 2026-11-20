---
name: nuxt-tests
description: 基于内嵌规则 的 Nuxt 测试规范。用于补齐 CRUD 页面 Vitest 用例。
---

# CRUD Nuxt Tests

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `web-admin/tests/**` 增加/更新 Vitest 测试。

## 规则（内嵌）

### nuxt_tests.yaml

```yaml
kind: ruleset
name: plugin/crud/frontend/nuxt_tests
version: 1

tests:
  runner: vitest
  files:
    - target: web-admin/tests/templates.spec.ts
      template: builtin/nuxt_test_table_page
      params:
        route: "/templates"
        expect:
          - "renders table"
          - "open create modal"
          - "submit form calls API"

gates.require: [PG-FE-UI-001]
```
