# PowerX Core Metadata Host 合同补齐要求

## 背景与边界

PowerXPlugin 的 `metadata.Runtime` 已通过 local `Service` 和 delegated `HostClient` 选择实现。Framework Lab 用启动时装配的两个调试 Runtime 分别验证本地 Store 和 Core tenant Host Contract；API Key 只作为独立开发联调凭证，已安装插件使用 STS。Core 当前 `RegisterTenantHostRoutes` 没有下表三条路由；Framework 已有 typed 方法，因此 delegated 请求会在 Core 侧失败。此处只要求新增 service actor Host 合同，不复用 `/api/v1/admin/metadata/*` 的 `admin_user` 授权。

| Framework 方法 | Core 应发布的路由 | Capability | 输入 | 成功 `data.payload` |
|---|---|---|---|---|
| `UpdateTag` | `PATCH /api/v1/tenant/metadata/tags/{tag_uuid}` | `com.corex.metadata.tag.manage` | `label_i18n`、`description_i18n`、`color`、`status` 均按已有更新 DTO 可选 | 更新后的 Tag 对象 |
| `ListTagBindings` | `GET /api/v1/tenant/metadata/tag-bindings?resource_type=...&resource_uuid=...` | `com.corex.metadata.tag.read` | 已登记资源类型与资源 UUID；可选 locale | `{ "items": [TagBinding...] }` |
| `ReplaceTagBindings` | `PUT /api/v1/tenant/metadata/tag-bindings:replace` | `com.corex.metadata.tag.manage` | `{ "resource_type": "...", "resource_uuid": "<UUID>", "tag_uuids": ["<UUID>"] }`；空数组表示清空 | `{ "items": [TagBinding...] }` |

上述返回结构与 Core 现有 `listTagBindings`、`replaceTagBindings` handler 一致；Framework `HostClient` 已按该结构解码。不要发布裸数组、将 `:replace` 改成 Admin 的 `PUT /tag-bindings`，或给 service actor 开放 admin 路由。若 Core 决定采用不同正式路径/结构，须先同步 Framework typed client 和合同测试后再发布。

## Core 侧落点

1. 在 `backend/internal/transport/http/admin/metadata/api.go` 的 `RegisterTenantHostRoutes` 添加三条路由，分别接入 `tenantHostAuthorize(access, "tag", "read"/"manage")`，再复用已有 `h.updateTag`、`h.listTagBindings`、`h.replaceTagBindings`。继续使用 `rejectTenantHostOverride()`，tenant 从已验证的 API Key 或 STS 服务凭证推导。
2. 在 `backend/config/platform_capabilities/metadata.yaml` 给现有 `com.corex.metadata.tag.read/manage` 增加对应 service actor REST binding；`auth_type: api_key_or_bearer`、`sts_direct: true`、`resource_scope: tenant`。API Key read 使用 `_scope.metadata.tag.read` / `read` / `api` / `tag`，manage 使用 `_scope.metadata.tag.manage` / `manage` / `api` / `tag`。保持 admin_user 绑定独立。
3. 发布/注册 capability，并为联调 API Key 精确授权所需 read/manage scope；已安装实例通过 manifest required 与 STS grant 获取权限。仅部署路由或重启插件都不会自动授予能力。

## 验收

- 同租户持有对应 grant 的 API Key 和已安装实例 STS 分别完成三项操作；返回结构由 Framework typed client 成功解码。
- read-only 凭证可列举绑定，但更新标签和替换绑定返回 403；撤销 grant 后即时拒绝。无凭证、跨租户对象、非法 UUID/资源类型明确失败。
- 替换绑定以单事务完成，空 `tag_uuids` 清空目标绑定；重复提交不产生重复绑定。写入后的 GET 返回同一组 binding UUID、tag UUID 与资源 UUID。
- 跑 Core 路由/能力清单测试和 `make capability-check`，再进行真实 API Key/STS 调用。PowerXPlugin 侧只保留本地合同测试；在 Core 更新并重新授权前不宣称 delegated 的这三项已联调通过。
