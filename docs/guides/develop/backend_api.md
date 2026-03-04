# 后端 API 调试指南（FastAPI/Gin 对齐）

> 目标：以 Gin 为权威基线，FastAPI 与 Gin 路由、权限与返回字段保持一致。

## 统一规则

调试变量（示例，可按需修改）：

```bash
export BASE=http://127.0.0.1:8277/api/v1
export TOKEN=YOUR_TOKEN
export TENANT_UUID=00000000-0000-0000-0000-000000000001
export PLUGIN_ID=com.powerx.plugins.base
export REFRESH_TOKEN=YOUR_REFRESH_TOKEN
export RESET_TOKEN=YOUR_RESET_TOKEN
export CLIENT_ID=YOUR_CLIENT_ID
export CLIENT_SECRET=YOUR_CLIENT_SECRET
export TOOLGRANT_TOKEN=YOUR_TOOLGRANT_TOKEN
```

- API 前缀：`/api/v1`（可通过 `backend/etc/config.yaml` 的 `server.api_prefix` 调整）
- 宿主反代前缀：`/_p/{plugin_id}{api_prefix}`
- 根路径白名单（与 Gin 对齐）：仅 `/healthz`、`/assets/builds/meta`、`/assets/builds/meta/{build_id}` 支持不带 `api_prefix`
- 请求头：
  - `Authorization: Bearer <token>`
  - `tenant_uuid: <tenant_uuid>`（部分路由强制）
  - `X-Request-ID`（可选；若不传会自动生成）
- 响应包裹：统一 `success/message/data/error/timestamp/request_id`

## 建议调试顺序（先联通再拓展）

1) 认证链路（`/admin/user/auth/*`）
2) IAM 核心（tenants / roles / permissions / departments / members）
3) 模板 CRUD（`/admin/templates/*` + `/templates/*`）
4) 能力管理（`/admin/capabilities/*`）
5) 运行时会话（`/admin/runtime/sessions/*`）
6) 业务模块（integration / marketplace / operations / security / privacy / tool-grant）

---

## Health / Manifest

- [x] `GET /healthz`
  - 参数: 无（也支持根路径 `/healthz`）
  - 示例: `curl -s -X GET "${BASE}/healthz"`
  - 示例: `curl -s -X GET "http://127.0.0.1:8277/healthz"`
  - 响应示例: `{"success":true,"data":{"status":"ok"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `GET /assets/builds/meta`
  - 参数: 无（也支持根路径 `/assets/builds/meta`）
  - 示例: `curl -s -X GET "${BASE}/assets/builds/meta"`
  - 示例: `curl -s -X GET "http://127.0.0.1:8277/assets/builds/meta"`
  - 响应示例: `{"success":true,"data":{"id":"dev","timestamp":0,"matcher":{},"prerendered":[]},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `GET /assets/builds/meta/{build_id}`
  - 参数: Path: build_id（也支持根路径 `/assets/builds/meta/{build_id}`）
  - 示例: `curl -s -X GET "${BASE}/assets/builds/meta/${BUILD_ID}"`
  - 示例: `curl -s -X GET "http://127.0.0.1:8277/assets/builds/meta/${BUILD_ID}"`
  - 响应示例: `{"success":true,"data":{"id":"dev","timestamp":0,"matcher":{},"prerendered":[]},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `GET /admin/manifest`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/manifest"`
  - 响应示例: `{"success":true,"data":{"id":"com.powerx.plugins.base","name":"Base Template Plugin"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `GET /admin/rbac`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/rbac"`
  - 响应示例: `{"success":true,"data":{"resources":[],"roles":[],"permissions":[]},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Auth（Admin）

- [x] `POST /admin/user/auth/login`
  - 参数: Body: {identifier, password}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/login" -H 'Content-Type: application/json' -d '{"identifier":"demo","password":"demo"}'`
  - 响应示例: `{"success":true,"message":"","data":{"token_type":"Bearer","access_token":"b5ceccd8130a48b2930f46d1f2736ca8","refresh_token":"ed8578ab13b94aafb6b171893e4efc7f","expires_in":3600,"expires_at":1769994728323,"scope":"default","policy_version":null,"plugin_id":null},"error":null,"timestamp":"2026-02-02T08:12:08.323592Z","request_id":"1770019928317562000"}`
- [x] `POST /admin/user/auth/register`
  - 参数: Body: {username|email, password}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/register" -H 'Content-Type: application/json' -d '{"username":"demo","password":"demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `POST /admin/user/auth/logout`
  - 参数: Body: {refresh_token|refreshToken}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/logout" -H 'Content-Type: application/json' -d '{"refresh_token":"${REFRESH_TOKEN}"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `POST /admin/user/auth/refresh`
  - 参数: Body: {refresh_token|refreshToken}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/refresh" -H 'Content-Type: application/json' -d '{"refresh_token":"${REFRESH_TOKEN}"}'`
  - 响应示例: `{"success":true,"data":{"access_token":"token","refresh_token":"token","expires_in":3600},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `GET /admin/user/auth/me`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/admin/user/auth/me" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"id":1,"email":"demo@example.com"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `PUT /admin/user/auth/profile`
  - 参数: Header: Authorization Bearer; Body: {profile fields}
  - 示例: `curl -s -X PUT "${BASE}/admin/user/auth/profile" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"display_name":"Demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `POST /admin/user/auth/change-password`
  - 参数: Header: Authorization Bearer; Body: {oldPassword|old_password, newPassword|new_password, confirmPassword|confirm_password}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/change-password" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"oldPassword":"old","newPassword":"new","confirmPassword":"new"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [x] `POST /admin/user/auth/reset-password`
  - 参数: Body: {email}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/reset-password" -H 'Content-Type: application/json' -d '{"email":"demo@example.com"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/user/auth/reset-password/confirm`
  - 参数: Body: {token, newPassword|new_password, confirmPassword|confirm_password}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/reset-password/confirm" -H 'Content-Type: application/json' -d '{"token":"${RESET_TOKEN}","newPassword":"new","confirmPassword":"new"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/user/auth/validate`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/admin/user/auth/validate" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/user/auth/permissions`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/admin/user/auth/permissions" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/user/auth/me/context`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/admin/user/auth/me/context" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/user/auth/me/tenants`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/admin/user/auth/me/tenants" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/user/auth/me/switch-tenant`
  - 参数: Header: Authorization Bearer; Body: {tenant_uuid?|tenant_id?}
  - 示例: `curl -s -X POST "${BASE}/admin/user/auth/me/switch-tenant" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/user/auth/me/roles`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/admin/user/auth/me/roles" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## IAM（Admin）

- [ ] `GET /admin/iam/tenants`
  - 参数: Query: 任意过滤字段（透传）
  - 示例: `curl -s -X GET "${BASE}/admin/iam/tenants" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/iam/tenants/{tenant_id}`
  - 参数: Path: tenant_id
  - 示例: `curl -s -X GET "${BASE}/admin/iam/tenants/${TENANT_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/tenants`
  - 参数: Body: {key, name, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/tenants" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"key":"tenant-a","name":"Tenant A"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/iam/tenants/{tenant_id}`
  - 参数: Path: tenant_id; Body: {fields}
  - 示例: `curl -s -X PATCH "${BASE}/admin/iam/tenants/${TENANT_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Tenant A"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

- [ ] `GET /admin/iam/roles`
  - 参数: Query: tenant_uuid (必填) + 其它过滤字段
  - 示例: `curl -s -X GET "${BASE}/admin/iam/roles?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/iam/roles/{role_id}`
  - 参数: Path: role_id
  - 示例: `curl -s -X GET "${BASE}/admin/iam/roles/${ROLE_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/roles`
  - 参数: Body: {tenant_uuid|tenantUuid, code, name, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/roles" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","code":"admin","name":"Admin"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/iam/roles/{role_id}`
  - 参数: Path: role_id; Body: {fields}
  - 示例: `curl -s -X PATCH "${BASE}/admin/iam/roles/${ROLE_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Admin"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `DELETE /admin/iam/roles/{role_id}`
  - 参数: Path: role_id
  - 示例: `curl -s -X DELETE "${BASE}/admin/iam/roles/${ROLE_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PUT /admin/iam/roles/{role_id}/permissions`
  - 参数: Path: role_id; Body: {tenant_uuid|tenantUuid, permission_ids[]}
  - 示例: `curl -s -X PUT "${BASE}/admin/iam/roles/${ROLE_ID}/permissions" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","permission_ids":["perm-1"]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/roles/{role_id}/members`
  - 参数: Path: role_id; Body: {tenant_uuid|tenantUuid, member_ids[]}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/roles/${ROLE_ID}/members" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","member_ids":["member-1"]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `DELETE /admin/iam/roles/{role_id}/members`
  - 参数: Path: role_id; Body: {tenant_uuid|tenantUuid, member_ids[]}
  - 示例: `curl -s -X DELETE "${BASE}/admin/iam/roles/${ROLE_ID}/members" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

- [ ] `GET /admin/iam/permissions`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/iam/permissions" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/iam/audit/logs`
  - 参数: Query: 任意过滤字段（透传）
  - 示例: `curl -s -X GET "${BASE}/admin/iam/audit/logs" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/auth/local/sts`
  - 参数: 无
  - 示例: `curl -s -X POST "${BASE}/admin/iam/auth/local/sts" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

- [ ] `GET /admin/iam/departments`
  - 参数: Query: tenant_uuid|tenantUuid (必填) + 其它过滤字段
  - 示例: `curl -s -X GET "${BASE}/admin/iam/departments?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/iam/departments/tree`
  - 参数: Query: tenant_uuid|tenantUuid (必填)
  - 示例: `curl -s -X GET "${BASE}/admin/iam/departments/tree?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/departments`
  - 参数: Body: {tenant_uuid, name, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/departments" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","name":"Dept"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/iam/departments/{department_id}`
  - 参数: Path: department_id; Body: {fields}
  - 示例: `curl -s -X PATCH "${BASE}/admin/iam/departments/${DEPARTMENT_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Dept"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `DELETE /admin/iam/departments/{department_id}`
  - 参数: Path: department_id
  - 示例: `curl -s -X DELETE "${BASE}/admin/iam/departments/${DEPARTMENT_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

- [ ] `GET /admin/iam/members`
  - 参数: Query: tenant_uuid|tenantUuid (必填), page?, page_size|pageSize?
  - 示例: `curl -s -X GET "${BASE}/admin/iam/members?tenant_uuid=$${TENANT_UUID}&page=1&page_size=20" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/members`
  - 参数: Body: {tenant_uuid, email, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/members" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","email":"demo@example.com"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/iam/members/{member_id}`
  - 参数: Path: member_id; Body: {fields}
  - 示例: `curl -s -X PATCH "${BASE}/admin/iam/members/${MEMBER_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/members/import`
  - 参数: Body: {tenant_uuid|tenantUuid, users[]|members[]}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/members/import" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","users":[{"email":"demo@example.com"}]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

- [ ] `GET /admin/iam/users`
  - 参数: Query: tenant_uuid|tenantUuid (必填), page?, page_size|pageSize?
  - 示例: `curl -s -X GET "${BASE}/admin/iam/users?tenant_uuid=$${TENANT_UUID}&page=1&page_size=20" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/users`
  - 参数: Body: {tenant_uuid, email, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/users" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","email":"demo@example.com"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/iam/users/{member_id}`
  - 参数: Path: member_id; Body: {fields}
  - 示例: `curl -s -X PATCH "${BASE}/admin/iam/users/${MEMBER_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/iam/users/import`
  - 参数: Body: {tenant_uuid|tenantUuid, users[]|members[]}
  - 示例: `curl -s -X POST "${BASE}/admin/iam/users/import" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","users":[{"email":"demo@example.com"}]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Templates（Admin + Public）

Admin:
- [ ] `POST /admin/templates/batch-clone`
  - 参数: Body: {source_ids|sourceIds (必填), copies?, name_prefix|namePrefix?, description_prefix|descriptionPrefix?}
  - 示例: `curl -s -X POST "${BASE}/admin/templates/batch-clone" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"source_ids":["tpl-1"],"copies":1}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/templates/{template_id}/validate`
  - 参数: Path: template_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/templates/${TEMPLATE_ID}/validate" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

Public:
- [ ] `GET /templates`
  - 参数: Query: q?, page?, limit?
  - 示例: `curl -s -X GET "${BASE}/templates" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /templates/{template_id}`
  - 参数: Path: template_id
  - 示例: `curl -s -X GET "${BASE}/templates/${TEMPLATE_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /templates`
  - 参数: Body: {name, description, content}
  - 示例: `curl -s -X POST "${BASE}/templates" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Demo","description":"Demo","content":"hello"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PUT /templates/{template_id}`
  - 参数: Path: template_id; Body: {name, description, content}
  - 示例: `curl -s -X PUT "${BASE}/templates/${TEMPLATE_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Demo","description":"Demo","content":"hello"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `DELETE /templates/{template_id}`
  - 参数: Path: template_id
  - 示例: `curl -s -X DELETE "${BASE}/templates/${TEMPLATE_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Capabilities（Admin）

- [ ] `GET /admin/capabilities`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/register/template`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/register/template" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/register`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/register" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"cap","version":"1.0.0"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/register/validate`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/register/validate" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"cap","version":"1.0.0"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/lifecycle/template`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/lifecycle/template" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/lifecycle`
  - 参数: Query: capability_id?（可选筛选）
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/lifecycle" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/lifecycle`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/lifecycle" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"capability_id":"cap","status":"active"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/lifecycle/{plan_id}/status`
  - 参数: Path: plan_id; Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/lifecycle/${PLAN_ID}/status" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"status":"active"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/exposure/template`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/exposure/template" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/exposure/{capability_id}`
  - 参数: Path: capability_id
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/exposure/${CAPABILITY_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PUT /admin/capabilities/exposure/{capability_id}`
  - 参数: Path: capability_id; Body: 任意 JSON（非空）
  - 示例: `curl -s -X PUT "${BASE}/admin/capabilities/exposure/${CAPABILITY_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"exposure":"public"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/quotas/{capability_id}`
  - 参数: Path: capability_id
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/quotas/${CAPABILITY_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/quotas/{capability_id}`
  - 参数: Path: capability_id; Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/quotas/${CAPABILITY_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"quotas":[]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/capabilities/reviews/{capability_id}`
  - 参数: Path: capability_id
  - 示例: `curl -s -X GET "${BASE}/admin/capabilities/reviews/${CAPABILITY_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/reviews/{capability_id}/resubmit`
  - 参数: Path: capability_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/reviews/${CAPABILITY_ID}/resubmit" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"comment":"resubmit"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/reviews/tasks/{task_id}/comments`
  - 参数: Path: task_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/reviews/tasks/${TASK_ID}/comments" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"comment":"ok"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/capabilities/reviews/tasks/{task_id}/decision`
  - 参数: Path: task_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/capabilities/reviews/tasks/${TASK_ID}/decision" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"decision":"approve"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Runtime Sessions（Admin）

- [ ] `POST /admin/runtime/sessions/register`
  - 参数: Body: {runtime_assignment_id (必填), ...}
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/sessions/register" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"runtime_assignment_id":"assign-1"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/sessions/{session_id}/ack`
  - 参数: Path: session_id; Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/sessions/${SESSION_ID}/ack" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/sessions/{session_id}/heartbeat`
  - 参数: Path: session_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/sessions/${SESSION_ID}/heartbeat" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/sessions/{session_id}/close`
  - 参数: Path: session_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/sessions/${SESSION_ID}/close" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/sessions/{session_id}/invoke`
  - 参数: Path: session_id; Body: {message_id, trace_id, correlation_id, tenant_uuid, tool_scope, issued_at, payload_ref, signature}
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/sessions/${SESSION_ID}/invoke" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"message_id":"m1","trace_id":"t1","correlation_id":"c1","tenant_uuid":"${TENANT_UUID}","tool_scope":"tool","issued_at":"2025-01-01T00:00:00Z","payload_ref":"ref","signature":"sig"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/bootstrap`
  - 参数: Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/bootstrap" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/runtime/metrics`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/runtime/metrics" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/runtime/quota/status`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/runtime/quota/status" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/quota/overrides`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/quota/overrides" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"overrides":[]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/runtime/event-bridge/emit`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/runtime/event-bridge/emit" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"event":"test"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Dev Console（Admin）

- [ ] `GET /admin/dev-console/config/sections`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/config/sections" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PUT /admin/dev-console/config/sections/{section_key}`
  - 参数: Path: section_key; Body: 任意 JSON（非空）
  - 示例: `curl -s -X PUT "${BASE}/admin/dev-console/config/sections/${SECTION_KEY}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"enabled":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/dev-console/audit/events`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/audit/events" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/dev-console/audit/export`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/audit/export" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/dev-console/jobs/runs`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/jobs/runs" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/dev-console/jobs/runs/{run_id}/retry`
  - 参数: Path: run_id
  - 示例: `curl -s -X POST "${BASE}/admin/dev-console/jobs/runs/${RUN_ID}/retry" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/dev-console/safe-ops/actions`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/dev-console/safe-ops/actions" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"action":"noop"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/dev-console/troubleshooting/summary`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/troubleshooting/summary" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/dev-console/webhooks/attempts`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/webhooks/attempts" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/dev-console/webhooks/attempts/{attempt_id}`
  - 参数: Path: attempt_id
  - 示例: `curl -s -X GET "${BASE}/admin/dev-console/webhooks/attempts/${ATTEMPT_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Integration

Admin:
- [ ] `GET /admin/integration/approvals`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/integration/approvals" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/approvals/{approval_id}/approve`
  - 参数: Path: approval_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/integration/approvals/${APPROVAL_ID}/approve" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"note":"ok"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/approvals/{approval_id}/reject`
  - 参数: Path: approval_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/integration/approvals/${APPROVAL_ID}/reject" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"note":"reject"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/integration/grant-matrix`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/integration/grant-matrix" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/integration/webhooks`
  - 参数: Query: tenant_uuid|tenantUuid?
  - 示例: `curl -s -X GET "${BASE}/admin/integration/webhooks" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/webhooks`
  - 参数: Body: {event_type, target_url, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/integration/webhooks" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"event_type":"plugin.updated","target_url":"https://example.com/webhook"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PUT /admin/integration/webhooks/{webhook_id}`
  - 参数: Path: webhook_id; Body: 任意 JSON（非空）
  - 示例: `curl -s -X PUT "${BASE}/admin/integration/webhooks/${WEBHOOK_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"target_url":"https://example.com/webhook"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `DELETE /admin/integration/webhooks/{webhook_id}`
  - 参数: Path: webhook_id
  - 示例: `curl -s -X DELETE "${BASE}/admin/integration/webhooks/${WEBHOOK_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/integration/webhooks/{webhook_id}/attempts`
  - 参数: Path: webhook_id
  - 示例: `curl -s -X GET "${BASE}/admin/integration/webhooks/${WEBHOOK_ID}/attempts" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/webhooks/attempts/{attempt_id}/replay`
  - 参数: Path: attempt_id
  - 示例: `curl -s -X POST "${BASE}/admin/integration/webhooks/attempts/${ATTEMPT_ID}/replay" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/integration/secrets`
  - 参数: Query: tenant_uuid|tenantUuid?
  - 示例: `curl -s -X GET "${BASE}/admin/integration/secrets" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/secrets`
  - 参数: Body: {integration_type, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/integration/secrets" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"integration_type":"slack"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/secrets/{secret_id}/rotate`
  - 参数: Path: secret_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/integration/secrets/${SECRET_ID}/rotate" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"reason":"rotate"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/secrets/{secret_id}/rotate/complete`
  - 参数: Path: secret_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/integration/secrets/${SECRET_ID}/rotate/complete" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/integration/secrets/{secret_id}/revoke`
  - 参数: Path: secret_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/integration/secrets/${SECRET_ID}/revoke" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"reason":"revoke"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/integration/secrets/{secret_id}/audit`
  - 参数: Path: secret_id
  - 示例: `curl -s -X GET "${BASE}/admin/integration/secrets/${SECRET_ID}/audit" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

Public:
- [ ] `POST /integration/dispatch`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/integration/dispatch" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"event_type":"demo","payload":{}}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /integration/capabilities/invoke`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/integration/capabilities/invoke" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"capability_id":"cap","payload":{}}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /integration/grant-matrix`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/integration/grant-matrix" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /integration/grant-matrix`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/integration/grant-matrix" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"overrides":[]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /integration/webhooks/subscriptions`
  - 参数: Body: {event_type, target_url, ...}
  - 示例: `curl -s -X POST "${BASE}/integration/webhooks/subscriptions" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"event_type":"plugin.updated","target_url":"https://example.com/webhook"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /integration/webhooks/subscriptions`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/integration/webhooks/subscriptions" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /integration/webhooks/dlq/{attempt_id}/replay`
  - 参数: Path: attempt_id
  - 示例: `curl -s -X POST "${BASE}/integration/webhooks/dlq/${ATTEMPT_ID}/replay" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /integration/secrets`
  - 参数: Body: {integration_type, ...}
  - 示例: `curl -s -X POST "${BASE}/integration/secrets" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"integration_type":"slack"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /integration/secrets/{secret_id}/rotate`
  - 参数: Path: secret_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/integration/secrets/${SECRET_ID}/rotate" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"reason":"rotate"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Marketplace

Admin:
- [ ] `GET /admin/marketplace/listings`
  - 参数: Query: tenant_uuid?, status?
  - 示例: `curl -s -X GET "${BASE}/admin/marketplace/listings?tenant_uuid=$${TENANT_UUID}&status=draft" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/listings`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/listings" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"Demo","status":"draft"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/marketplace/listings/{listing_id}`
  - 参数: Path: listing_id
  - 示例: `curl -s -X GET "${BASE}/admin/marketplace/listings/${LISTING_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/marketplace/listings/{listing_id}`
  - 参数: Path: listing_id; Body: 任意 JSON（非空）
  - 示例: `curl -s -X PATCH "${BASE}/admin/marketplace/listings/${LISTING_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"status":"draft"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/listings/{listing_id}/review`
  - 参数: Path: listing_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/listings/${LISTING_ID}/review" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"note":"review"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/listings/{listing_id}/publish`
  - 参数: Path: listing_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/listings/${LISTING_ID}/publish" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"note":"publish"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/listings/{listing_id}/suspend`
  - 参数: Path: listing_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/listings/${LISTING_ID}/suspend" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"note":"suspend"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/checklist/graphql`
  - 参数: Body: GraphQL payload 可选
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/checklist/graphql" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"query":"query { __typename }"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/marketplace/recommendation/config`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/admin/marketplace/recommendation/config" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/recommendation/sync`
  - 参数: Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/recommendation/sync" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"ok":true}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `PATCH /admin/marketplace/recommendation/experiment`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X PATCH "${BASE}/admin/marketplace/recommendation/experiment" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"name":"exp"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/marketplace/usage`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/admin/marketplace/usage" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"items":[]}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/marketplace/usage/tenants/{tenant_id}/licenses/{license_id}/metrics`
  - 参数: Path: tenant_id, license_id
  - 示例: `curl -s -X GET "${BASE}/admin/marketplace/usage/tenants/${TENANT_ID}/licenses/${LICENSE_ID}/metrics" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/marketplace/revenue-share/reports`
  - 参数: Query: tenant_uuid?
  - 示例: `curl -s -X GET "${BASE}/admin/marketplace/revenue-share/reports?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

Public:
- [ ] `GET /marketplace/listings`
  - 参数: 无
  - 示例: `curl -s -X GET "${BASE}/marketplace/listings"`
  - 响应示例: `{"success":true,"data":{"items":[]},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /marketplace/sla/{plugin_id}`
  - 参数: Path: plugin_id
  - 示例: `curl -s -X GET "${BASE}/marketplace/sla/${PLUGIN_ID}"`
  - 响应示例: `{"success":true,"data":{"items":[]},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /marketplace/licenses`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/marketplace/licenses" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"plugin_id":"${PLUGIN_ID}","tenant_uuid":"${TENANT_UUID}"}'`
  - 响应示例: `{"success":true,"data":{"id":"lic"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /marketplace/licenses/{license_id}`
  - 参数: Path: license_id
  - 示例: `curl -s -X GET "${BASE}/marketplace/licenses/${LICENSE_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{"id":"lic"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /marketplace/licenses/{license_id}`
  - 参数: Path: license_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/marketplace/licenses/${LICENSE_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"action":"renew"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /marketplace/licenses/{license_id}/offline-extend`
  - 参数: Path: license_id; Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/marketplace/licenses/${LICENSE_ID}/offline-extend" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"hours":24}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Operations（Admin）

- 通用说明
  - Header: `Authorization: Bearer <token>`
  - `plugin_id` 解析优先级：`x-plugin-id` Header → payload `plugin_id` → query `plugin_id`
  - 成功响应 envelope：`{ success, data, message?, timestamp, request_id }`

- Support
  - [ ] `GET /admin/operations/support/playbook`
    - 参数: Query: plugin_id (必填), tenant_uuid?
    - 示例: `curl -s -X GET "${BASE}/admin/operations/support/playbook?plugin_id=$${PLUGIN_ID}" -H 'Authorization: Bearer ${TOKEN}'`
    - 响应示例: `{"success":true,"data":{"channels":[],"knowledge_base":[],"readiness":[]},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Query: `tenant_uuid`(可选), `plugin_id`(必填)
    - Response `data`:
      - `channels[]`: `{ id, channel, address?, escalates?, service_window?, metadata?, enabled }`
      - `knowledge_base[]`: `{ label, url }`
      - `readiness[]`: `{ key, status, blocking, completed, notes? }`
  - [ ] `PUT /admin/operations/support/playbook`
    - 参数: Body: {plugin_id?, tenant_uuid?, channels[], knowledge_base[]}
    - 示例: `curl -s -X PUT "${BASE}/admin/operations/support/playbook" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"plugin_id":"${PLUGIN_ID}","channels":[],"knowledge_base":[]}'`
    - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Body: `{ tenant_uuid?, plugin_id?, channels: [...], knowledge_base: [...] }`
    - `channels[]`: `{ channel, address?, escalates?, service_window?, metadata?, enabled? }`
    - `knowledge_base[]`: `{ label, url }`
  - [ ] `POST /admin/operations/support/channels/test`
    - 参数: Body: JSON 可选
    - 示例: `curl -s -X POST "${BASE}/admin/operations/support/channels/test" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{}'`
    - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Response `data`: `{ status: "ok" }`（message: `channel validation dispatched`）
  - [ ] `GET /admin/operations/support/metrics`
    - 参数: Query: plugin_id (必填)
    - 示例: `curl -s -X GET "${BASE}/admin/operations/support/metrics?plugin_id=$${PLUGIN_ID}" -H 'Authorization: Bearer ${TOKEN}'`
    - 响应示例: `{"success":true,"data":{"first_response_hours":0,"resolution_hours":0,"csat_average":0,"resolution_rate":0},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Query: `plugin_id`(必填)
    - Response `data`: `{ first_response_hours, resolution_hours, csat_average, resolution_rate }`

- SLA
  - [ ] `GET /admin/operations/sla/profiles`
    - 参数: Query: plugin_id (必填)
    - 示例: `curl -s -X GET "${BASE}/admin/operations/sla/profiles?plugin_id=$${PLUGIN_ID}" -H 'Authorization: Bearer ${TOKEN}'`
    - 响应示例: `{"success":true,"data":[],"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Query: `plugin_id`(必填)
    - Response `data[]`（snake_case）:
      - `id, plugin_id, plan_type, uptime_target, uptime_actual, response_target_ms, response_actual_ms, success_target_pct, success_actual_pct, support_frt_target_hours, support_frt_actual_hours, sla_score, incentive_applied_at?, penalty_applied_at?, notes?, computed_at, created_at, updated_at`
  - [ ] `POST /admin/operations/sla/profiles`
    - 参数: Body: {planType, targets{uptimeTarget,responseTargetMs,successTargetPct,supportFrtTargetHours}, plugin_id?}
    - 示例: `curl -s -X POST "${BASE}/admin/operations/sla/profiles" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"planType":"real_time","targets":{"uptimeTarget":99.9,"responseTargetMs":600,"successTargetPct":99.5,"supportFrtTargetHours":4},"plugin_id":"${PLUGIN_ID}"}'`
    - 响应示例: `{"success":true,"data":{"plan_type":"real_time"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Body: `{ planType, targets: { uptimeTarget, responseTargetMs, successTargetPct, supportFrtTargetHours }, plugin_id? }`
    - Response `data`: 同 GET 列表单项（snake_case）
  - [ ] `POST /admin/operations/sla/profiles/recompute`
    - 参数: Query/Body: plugin_id (必填)
    - 示例: `curl -s -X POST "${BASE}/admin/operations/sla/profiles/recompute?plugin_id=$${PLUGIN_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"plugin_id":"${PLUGIN_ID}"}'`
    - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Query/Body: `plugin_id`(必填)
    - Response: 202（无 envelope）
  - [ ] `PATCH /admin/operations/sla/profiles/actuals`
    - 参数: Body: {planType, actuals{uptimeActual,responseActualMs,successActualPct,supportFrtActualHours}, plugin_id?}
    - 示例: `curl -s -X PATCH "${BASE}/admin/operations/sla/profiles/actuals" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"planType":"real_time","actuals":{"uptimeActual":99.0,"responseActualMs":800,"successActualPct":98.5,"supportFrtActualHours":6},"plugin_id":"${PLUGIN_ID}"}'`
    - 响应示例: `{"success":true,"data":{"plan_type":"real_time"},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Body: `{ planType, actuals: { uptimeActual, responseActualMs, successActualPct, supportFrtActualHours }, plugin_id? }`
    - Response `data`: 同 GET 列表单项（snake_case）

- Incidents
  - [ ] `POST /admin/operations/incidents`
    - 参数: Body: {severity, detection_source, summary, tenant_uuid?, labels?, mitigation?, confidentiality?, impact?, next_update_at?, plugin_id?}
    - 示例: `curl -s -X POST "${BASE}/admin/operations/incidents" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"severity":"sev2","detection_source":"monitor","summary":"demo","plugin_id":"${PLUGIN_ID}"}'`
    - 响应示例: `{"success":true,"data":{"incident":{"id":"inc"},"timeline":[],"checklist":[],"checklist_status":{"support_ready":false,"incident_ready":false,"sla_ready":false,"blocking_items":[]}},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Body: `{ plugin_id?, tenant_uuid?, severity, detection_source, summary, impact?, mitigation?, labels?, confidentiality?, next_update_at? }`
    - Response `data`: `IncidentResponse`（见 GET 单条）
  - [ ] `GET /admin/operations/incidents`
    - 参数: Query: plugin_id (必填), severity*, status*, label*, from?, to?
    - 示例: `curl -s -X GET "${BASE}/admin/operations/incidents?plugin_id=$${PLUGIN_ID}&severity=sev2&status=detected" -H 'Authorization: Bearer ${TOKEN}'`
    - 响应示例: `{"success":true,"data":[],"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Query: `plugin_id`(必填), `severity`(可多值), `status`(可多值), `label`(可多值), `from`(ISO8601), `to`(ISO8601)
    - Response `data[]`: `IncidentRecord`
  - [ ] `GET /admin/operations/incidents/{incident_id}`
    - 参数: Path: incident_id; Query: plugin_id (必填)
    - 示例: `curl -s -X GET "${BASE}/admin/operations/incidents/${INCIDENT_ID}?plugin_id=$${PLUGIN_ID}" -H 'Authorization: Bearer ${TOKEN}'`
    - 响应示例: `{"success":true,"data":{"incident":{"id":"inc"},"timeline":[],"checklist":[],"checklist_status":{"support_ready":false,"incident_ready":false,"sla_ready":false,"blocking_items":[]}},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Query: `plugin_id`(必填)
    - Response `data`: `IncidentResponse`
  - [ ] `PATCH /admin/operations/incidents/{incident_id}`
    - 参数: Path: incident_id; Body: {status?, mitigation?, root_cause?, next_update_at?, confidentiality?, labels?, plugin_id?}
    - 示例: `curl -s -X PATCH "${BASE}/admin/operations/incidents/${INCIDENT_ID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"status":"acknowledged","plugin_id":"${PLUGIN_ID}"}'`
    - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Body: `{ plugin_id?, status?, mitigation?, root_cause?, next_update_at?, confidentiality?, labels? }`
    - Response `data`: `IncidentResponse`
  - [ ] `POST /admin/operations/incidents/{incident_id}/timeline`
    - 参数: Path: incident_id; Body: {entry_type, message, stakeholder_channel?, author_role?, metadata?, plugin_id?}
    - 示例: `curl -s -X POST "${BASE}/admin/operations/incidents/${INCIDENT_ID}/timeline" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"entry_type":"update","message":"status","plugin_id":"${PLUGIN_ID}"}'`
    - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
    - Body: `{ plugin_id?, entry_type, message, stakeholder_channel?, author_role?, metadata? }`
    - Response `data`: `IncidentTimelineEntry`（message: `timeline entry recorded`）

- 类型参考
  - `IncidentRecord`: `{ id, severity, status, detection_source, summary, impact?, mitigation?, root_cause?, labels?, confidentiality?, detected_at, acknowledged_at?, mitigated_at?, resolved_at?, closed_at?, next_update_at? }`
  - `IncidentTimelineEntry`: `{ id, incident_id, entry_type, message, stakeholder_channel?, author_role?, posted_at, metadata? }`
  - `IncidentChecklistItem`: `{ id, incident_id, item_key, description, status, completed_at? }`
  - `IncidentResponse`: `{ incident, timeline[], checklist[], checklist_status: { support_ready, incident_ready, sla_ready, blocking_items[] } }`

---

## Security（Admin）

- [ ] `GET /admin/security/consent-tokens`
  - 参数: Query: tenant_uuid (必填), status*?
  - 示例: `curl -s -X GET "${BASE}/admin/security/consent-tokens?tenant_uuid=$${TENANT_UUID}&status=active" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/security/consent-tokens/{token_id}/revoke`
  - 参数: Path: token_id; Query: tenant_uuid (必填); Body: JSON 可选
  - 示例: `curl -s -X POST "${BASE}/admin/security/consent-tokens/${TOKEN_ID}/revoke?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{}'`
  - 响应示例: `{"status":204}`
- [ ] `GET /admin/security/lifecycle-events`
  - 参数: Query: tenant_uuid (必填), event_type*?, limit?
  - 示例: `curl -s -X GET "${BASE}/admin/security/lifecycle-events?tenant_uuid=$${TENANT_UUID}&event_type=consent&limit=10" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/security/audit-reports`
  - 参数: Query: limit?
  - 示例: `curl -s -X GET "${BASE}/admin/security/audit-reports" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/security/advisories`
  - 参数: Body: {reference, severity, summary, sla_deadline?}
  - 示例: `curl -s -X POST "${BASE}/admin/security/advisories" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"reference":"CVE-0000","severity":"high","summary":"demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/security/advisories`
  - 参数: Query: severity*?, status*?, limit?
  - 示例: `curl -s -X GET "${BASE}/admin/security/advisories" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/security/advisories/{advisory_id}/publish`
  - 参数: Path: advisory_id; Body: {patched_in_version, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/security/advisories/${ADVISORY_ID}/publish" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"patched_in_version":"1.0.1"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /admin/security/toolgrants/revoke`
  - 参数: Body: {tenant_uuid, toolgrant_id, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/security/toolgrants/revoke" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","toolgrant_id":"tg-1"}'`
  - 响应示例: `{"status":204}`
- [ ] `GET /admin/security/toolgrants/revocations`
  - 参数: Query: tenant_uuid (必填), limit?
  - 示例: `curl -s -X GET "${BASE}/admin/security/toolgrants/revocations?tenant_uuid=$${TENANT_UUID}&limit=10" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/security/toolgrants/usage`
  - 参数: Query: tenant_uuid (必填), toolgrant_id?, limit?
  - 示例: `curl -s -X GET "${BASE}/admin/security/toolgrants/usage?tenant_uuid=$${TENANT_UUID}&limit=10" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Privacy（Admin）

- [ ] `GET /admin/privacy/consent-tokens`
  - 参数: Query: tenant_uuid (必填)
  - 示例: `curl -s -X GET "${BASE}/admin/privacy/consent-tokens?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/privacy/lifecycle-events`
  - 参数: Query: tenant_uuid (必填)
  - 示例: `curl -s -X GET "${BASE}/admin/privacy/lifecycle-events?tenant_uuid=$${TENANT_UUID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Tool Grant（Admin）

- [ ] `POST /admin/tool-grant/revoke`
  - 参数: Body: {tenant_uuid, toolgrant_id, ...}
  - 示例: `curl -s -X POST "${BASE}/admin/tool-grant/revoke" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"tenant_uuid":"${TENANT_UUID}","toolgrant_id":"tg-1"}'`
  - 响应示例: `{"status":204}`
- [ ] `GET /admin/tool-grant/revocations`
  - 参数: Query: tenant_uuid (必填), limit?
  - 示例: `curl -s -X GET "${BASE}/admin/tool-grant/revocations?tenant_uuid=$${TENANT_UUID}&limit=10" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /admin/tool-grant/usage`
  - 参数: Query: tenant_uuid (必填), toolgrant_id?, limit?
  - 示例: `curl -s -X GET "${BASE}/admin/tool-grant/usage?tenant_uuid=$${TENANT_UUID}&limit=10" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Agent

- [ ] `POST /agent/tenants/{tenant_id}/credentials`
  - 参数: Path: tenant_id(UUID); Body: {plugin_id, client_id, client_secret}
  - 示例: `curl -s -X POST "${BASE}/agent/tenants/${TENANT_ID}/credentials" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"plugin_id":"${PLUGIN_ID}","client_id":"${CLIENT_ID}","client_secret":"${CLIENT_SECRET}"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /agent/sts/exchange`
  - 参数: Body: 无（依赖服务端配置）
  - 示例: `curl -s -X POST "${BASE}/agent/sts/exchange" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /agent/security/privacy/consent`
  - 参数: Header: Authorization Bearer; tenant_uuid 由 tenant_uuid 或 query tenant_uuid 提供
  - 示例: `curl -s -X GET "${BASE}/agent/security/privacy/consent" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /agent/security/privacy/lifecycle`
  - 参数: Header: Authorization Bearer; Body: {event_type, asset_key, metadata?}; tenant_uuid 由 tenant_uuid 或 query tenant_uuid 提供
  - 示例: `curl -s -X POST "${BASE}/agent/security/privacy/lifecycle" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"event_type":"consent","asset_key":"asset-1"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /agent/security/toolgrants/verify`
  - 参数: Header: Authorization Bearer; Body: {token}; tenant_uuid 由 tenant_uuid 或 query tenant_uuid 提供
  - 示例: `curl -s -X POST "${BASE}/agent/security/toolgrants/verify" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"token":"${TOOLGRANT_TOKEN}"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`

---

## Mini App

- [ ] `POST /mini-app/auth/register`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/mini-app/auth/register" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"identifier":"demo","password":"demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `POST /mini-app/auth/login`
  - 参数: Body: 任意 JSON（非空）
  - 示例: `curl -s -X POST "${BASE}/mini-app/auth/login" -H 'Authorization: Bearer ${TOKEN}' -H 'Content-Type: application/json' -d '{"identifier":"demo","password":"demo"}'`
  - 响应示例: `{"success":true,"data":{"ok":true},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /mini-app/ping`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/mini-app/ping" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /mini-app/templates`
  - 参数: Header: Authorization Bearer
  - 示例: `curl -s -X GET "${BASE}/mini-app/templates" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
- [ ] `GET /mini-app/templates/{template_id}`
  - 参数: Header: Authorization Bearer; Path: template_id
  - 示例: `curl -s -X GET "${BASE}/mini-app/templates/${TEMPLATE_ID}" -H 'Authorization: Bearer ${TOKEN}'`
  - 响应示例: `{"success":true,"data":{},"message":"","error":null,"timestamp":"2025-01-01T00:00:00Z","request_id":"req"}`
