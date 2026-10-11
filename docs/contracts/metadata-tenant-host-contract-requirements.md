# Metadata Tenant Host Contract：PowerX 交接要求

## 结论

当前 Core 仅发布 `/api/v1/admin/metadata/*` 管理端路由。它们是 admin-user 边界，不能由 Framework delegated 插件服务调用；现有 `runtime/metadata.Client` 仍经 Gateway invoker 包装这些 admin endpoint，因此不得作为正式 delegated Metadata Host Contract。

请 PowerX Core 发布独立的 tenant-scoped、STS service-actor Host Contract；不要给现有 admin 路由追加插件兼容分支。

## 统一安全规则

- 所有路由置于 `/api/v1/tenant/metadata/*`；租户只能从 STS/API Key 推导，拒绝 body/query/header 的 `tenant_uuid`。
- 调用方必须是 service actor，且同时校验 capability 已发布、tenant registration 与 credential grant；`sts_direct` 不等于授权。
- 外部对象及关联只使用 UUID：`namespace_uuid`、`dictionary_item_uuid`、`taxonomy_uuid`、`taxonomy_node_uuid`、`tag_uuid`、`binding_uuid`、`resource_type_uuid`。
- 不暴露 numeric ID、内部审计主体或数据库主键；跨租户对象一律 404，避免存在性泄露。
- 所有失败响应使用 `error_code` 与 `reason_code`；至少固定 `METADATA_INVALID_ARGUMENT`、`METADATA_UNAUTHORIZED`、`METADATA_FORBIDDEN`、`METADATA_NOT_FOUND`、`METADATA_CONFLICT`、`METADATA_UPSTREAM_DEPENDENCY`。

## 所需路由与能力

| 域 | 路由 | Capability |
|---|---|---|
| Dictionary | `GET/POST /tenant/metadata/dictionaries`；`GET/POST /tenant/metadata/dictionaries/{namespace_uuid}/items` | `com.corex.metadata.dictionary.read/manage` |
| Taxonomy | `GET/POST /tenant/metadata/taxonomies`；`GET/POST /tenant/metadata/taxonomies/{taxonomy_uuid}/nodes` | `com.corex.metadata.taxonomy.read/manage` |
| Tag | `GET/POST /tenant/metadata/tags`；`GET/PUT /tenant/metadata/tags/bindings` | `com.corex.metadata.tag.read/manage` |
| Resource type | `GET/POST /tenant/metadata/resource-types` | `com.corex.metadata.resource_type.read/manage` |

`PUT /tenant/metadata/tags/bindings` 必须是显式全量替换，返回本次绑定的 UUID 列表；不得以自由文本或 numeric ID 推断对象。

## Core 验收与部署

1. OpenAPI、capability REST binding、service authorization、稳定错误 envelope 一并提交。
2. 合同测试覆盖成功、非法 UUID、tenant 注入拒绝、401、未授权 403、跨租户 404、并发/唯一冲突 409 与依赖 503。
3. 执行 `make capability-check`；发布时执行 `make seed`（或 capability seed）并重启 Core。
4. 已安装插件须在 manifest 声明所需 metadata capability，升级/重新启用后再进行真实 STS grant 联调。

Framework 将在该合同发布后把 `runtime/metadata.Client` 从 Gateway/admin transport 替换为 typed STS tenant client，保留既有 `metadata.Service` 与 `Runtime` Factory。
