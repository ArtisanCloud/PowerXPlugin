# Runbook: Standalone IAM & RBAC

适用于启用本地 IAM（`POWERX_PROXY=0`）的插件部署，帮助运维在租户锁定、成员解锁与审计检索场景下快速处理。

## 0. 前置条件

1. 已执行 `go run ./cmd/database/main.go setup` 并确认本地管理员可登录 `skeleton/web-admin/nuxt`。
2. 后端 `skeleton/backend/go-gin` 正在运行且监听默认 `http://localhost:8077`。
3. 已获取管理员 Token，可通过登录 Web Admin 后复制 `localStorage.access_token`，或直接使用 `useAuth` 的刷新 Token。

下文 `API_BASE` 默认指向 `http://localhost:8077/api/v1`.

## 1. 锁定 / 恢复租户

1. 查询租户列表：

   ```bash
   curl -H "Authorization: Bearer $TOKEN" "$API_BASE/admin/iam/tenants"
   ```

2. 锁定租户（status=`suspended`），会自动禁用该租户所有成员并清理 Refresh Token：

   ```bash
   curl -X PATCH \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     "$API_BASE/admin/iam/tenants/1" \
     -d '{"status":"suspended"}'
   ```

3. 恢复租户，status 设置回 `active` 并提示管理员重新启用必要成员：

   ```bash
   curl -X PATCH \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     "$API_BASE/admin/iam/tenants/1" \
     -d '{"status":"active"}'
   ```

## 2. 成员解锁 / 禁用

1. 查询成员（支持 `q` 与 `status` 过滤）：

   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "$API_BASE/admin/iam/members?tenant_uuid=$TENANT_UUID"
   ```

2. 禁用成员：

   ```bash
   curl -X PATCH \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     "$API_BASE/admin/iam/members/1001" \
     -d '{"status":"disabled"}'
   ```

   禁用后后台会立即 revoke 对应 Refresh Token，确保无法继续访问。

3. 重新启用成员：

   ```bash
   curl -X PATCH \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     "$API_BASE/admin/iam/members/1001" \
     -d '{"status":"active"}'
  ```

## 2.1 同一账号加入多个租户

适用于排查“账号凭证共享、但成员数据按租户隔离”的场景，可验证 `iam_users` 与 `iam_members` 之间的映射。

1. 使用账号邮箱创建第一个租户成员：

   ```bash
   curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     "$API_BASE/admin/iam/members" \
     -d '{
       "tenant_uuid": "'$TENANT_A'",
       "email": "dev1@example.com",
       "display_name": "Dev One",
       "department_id": null,
       "roles": []
     }'
   ```

2. 在另一个租户重复创建同一邮箱，会复用相同的 `iam_users` 记录、但生成新的 `iam_members`：

   ```bash
   curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     "$API_BASE/admin/iam/members" \
     -d '{
       "tenant_uuid": "'$TENANT_B'",
       "email": "dev1@example.com",
       "display_name": "Dev One @ TenantB",
       "roles": []
     }'
   ```

3. 通过 API 或直接查询数据库验证：

   ```bash
   # 账户表只保留一条邮箱记录
   psql "$DATABASE_URL" -c "SELECT id,email FROM iam_users WHERE email='dev1@example.com'"

   # 成员表存在两条记录，对应不同 tenant_uuid
   psql "$DATABASE_URL" -c "SELECT tenant_uuid,user_id FROM iam_members WHERE user_id=<ACCOUNT_ID>"
   ```

4. 使用 Web Admin 登录该账号，切换至不同租户（若开放租户切换），或直接在 API 请求中带上 `tenant_uuid`，即可验证相同账号在不同租户拥有不同角色/部门。

## 2.2 API 速查表

| 操作 | HTTP 方法 | 路径 | 说明 |
| ---- | -------- | ---- | ---- |
| 查询成员 | GET | `/api/v1/admin/iam/members?tenant_uuid=...` | 列表支持 `status`、`q` 过滤；等价 `/users` 仅用于旧版兼容。 |
| 创建成员 | POST | `/api/v1/admin/iam/members` | 提交 `tenant_uuid/email/roles` 等字段创建租户内成员。 |
| 更新成员 | PATCH | `/api/v1/admin/iam/members/{member_id}` | 更新成员状态、部门、角色等。 |
| 批量导入 | POST | `/api/v1/admin/iam/members/import` | `tenant_uuid` + `users[]` 数组，返回成功/失败摘要。 |

## 3. 审计日志检索

接口：`GET /api/v1/admin/iam/audit/logs?tenant_uuid=...&action=...`

1. 查询指定租户最近 50 条：

   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "$API_BASE/admin/iam/audit/logs?tenant_uuid=$TENANT_UUID"
   ```

2. 过滤成员相关操作（ resource=`iam.member` ）：

   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "$API_BASE/admin/iam/audit/logs?tenant_uuid=$TENANT_UUID&resource=iam.member"
   ```

3. 导出到 JSON 文件：

   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "$API_BASE/admin/iam/audit/logs?tenant_uuid=$TENANT_UUID" \
     -o /tmp/iam-audit-$TENANT_UUID.json
   ```

## 4. 快速验证

1. 锁定租户后立即尝试使用该租户成员刷新 Token，应得到 401。
2. 在 Web Admin 的“成员管理”中选中指定成员 → “禁用”，确认 UI 更新且 Playwright `iam-local` 用例通过。
3. 使用上述审计接口确认有 `iam.tenant`/`iam.member` 的 `create/update/delete` 记录。

## 5. 故障排查

| 现象 | 处理 |
| ---- | ---- |
| 锁定租户后成员仍可访问 | 检查是否使用 Delegated 模式；确认 Refresh Token 是否被清理，可查看 `iam_refresh_tokens` 表。 |
| 成员解锁后无法登录 | 确认密码是否正确，或重新执行 `px-plugin iam seed` 重置管理员/成员密码。 |
| 审计日志查询为空 | 确认 `includeIAM=true` 运行了最新迁移；在 PostgreSQL 中检查 `iam_audit_logs` 表是否有数据。 |

> 建议在每次操作后运行 `npm --prefix skeleton/web-admin/nuxt run test:e2e -- iam-local` 做冒烟验证，确保 UI + API 流程完整。

## 6. CLI 备份与管理员重置

1. **导出/备份**：使用 `px-plugin iam export` 从本地数据库导出租户、成员、角色与权限关系。

   ```bash
   # 默认 10 秒超时，支持 --tenant 过滤单个租户
   px-plugin iam export \
     --entry skeleton \
     --output /tmp/iam-backup.json \
     --pretty
   ```

   - `--tenant`：可传入 tenant key 或 UUID，便于针对性备份。
   - `--pretty`：格式化输出，方便人工 diff，若要压缩可去掉此 flag。
   - 导出的 JSON 可直接交给运维/审计，或导入灾备库进行对比。

2. **重置默认管理员**：当忘记管理员密码或需要快速初始化 Standalone 环境时，使用 `px-plugin iam seed`。命令会在事务里维护租户、账号、成员、角色与默认权限，若当前环境处于 Delegated，会给出提示并要求加 `--force`。

   ```bash
   px-plugin iam seed \
     --entry skeleton \
     --tenant-key 00000000-0000-0000-0000-000000000001 \
     --tenant-name "Local Tenant" \
     --admin-email admin@local.test \
     --admin-password S3cret!!
   ```

   - 建议在本地回归时修改 `--admin-email/--admin-password`，并记录在安全存储。
   - 如果 `POWERX_PROXY=1` 或 `POWERX_RBAC_DELEGATE=true`，命令会拒绝执行并提示加入 `--force`，避免误操作宿主环境。

3. **验证 CLI 结果**：执行完上述命令后，使用 `px-plugin iam export --tenant ...` 进行快速比对，或者直接运行 `npm --prefix skeleton/web-admin/nuxt run test:e2e -- auth-local` 验证登录流程。
