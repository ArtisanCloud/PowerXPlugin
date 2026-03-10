---
name: nuxt-api-client
description: 基于内嵌规则 的 Nuxt API Client 规范。用于生成/校对 api.ts 与 CRUD composable。
---

# CRUD Nuxt API Client

## 步骤

1) 打开 `本文件内嵌规则`。
2) 生成或校对 `web-admin/plugins/api.ts` 与 `web-admin/composables/useTemplates.ts`。
3) 在 `web-admin/composables/api/index.ts` 统一导出入口。

## 核对点

- 统一使用 `apiGet/apiPost/apiPatch/apiPut/apiDel` 与 `useApiClient`。
- 不直接调用 `$fetch`，保持认证/租户头与错误处理一致。
- body 默认 JSON 序列化，FormData 原样。

## 规则（内嵌）

### nuxt_api_client.yaml

```yaml
kind: ruleset
name: plugin/crud/frontend/nuxt_api_client
version: 1

client:
  runtime_config:
    public_keys:
      apiBase: "NUXT_PUBLIC_API_BASE"
      pluginId: "NUXT_PUBLIC_PLUGIN_ID"
      powerxProxy: "NUXT_PUBLIC_POWERX_PROXY"
    defaults:
      apiBase: "/_p/<plugin-id>/api/v1"
      powerxProxy: 1
  files:
    - target: web-admin/plugins/api.ts
      template: builtin/nuxt_api_plugin_fetch
    - target: web-admin/composables/useTemplates.ts
      template: builtin/nuxt_composable_crud
      params:
        resource: "templates"
        dto:
          item: "TemplateItem"
          create: "TemplateCreateReq"
          update: "TemplateUpdateReq"
  usage:
    import: |
      import { apiGet, apiPost, apiPatch, apiPut, apiDel, useApiClient } from "~/composables/api";
    notes:
      - 不要直接调用 $fetch；统一用 apiGet/apiPost/apiPatch/apiPut/apiDel 以附带认证/租户头并获得一致的错误处理。
      - apiGet(path, query?, init?) / apiPost(path, body?, init?) 语义同 $fetch，body 默认自动 JSON.stringify，FormData 保持原样。
      - useApiClient() 暴露 client/baseURL，如需底层访问请使用 client.raw。
      - 页面/组件不要直接调用 apiGet/apiPost；按域封装 useXXX 组合式函数（参考 web-admin/app/composables/api/useTemplate.ts、useRss.ts），在 ~/composables/api/index.ts 暴露入口，统一类型/字段规范。

gates.require: [PG-FE-API-001]
```
