# PowerX IAM Phase 2 Host Contract Handoff

## 目标

让 Framework IAM delegated 模式能够以 UUID-only、tenant-scoped 的 Host Contract 完成部门、角色、权限目录读取与实时授权判定。插件不得读取或写入 PowerX IAM 数据库。

## 前置数据迁移

先完成数据模型迁移，再发布 API；不接受 numeric ID 作为跨服务输入或返回字段。

1. `departments` 增加并回填 `department_uuid`；将 `parent_id`、`leader_member_id` 分别迁移为 `parent_department_uuid`、`leader_member_uuid`。
2. `permissions` 增加并回填稳定 `permission_uuid`。
3. `role_permissions` 使用 `role_uuid + permission_uuid` 关联；若该关联对象需要独立审计或状态，增加自身 UUID。
4. 所有成员目录 DTO 返回 `department_uuids`，不返回内部部门 numeric ID。
5. 迁移校验必须拒绝缺失 UUID；不保留 numeric ID 兼容请求入口。

## Host API

所有接口使用 PowerX tenant-scoped STS service actor。目录与授权能力分别最小授权；不得复用后台 admin-user capability。

| API | Capability | 成功语义 |
| --- | --- | --- |
| `GET /api/v1/tenant/iam/departments` | `com.corex.iam.directory.read` | 返回当前 tenant 的 UUID 部门目录 |
| `GET /api/v1/tenant/iam/roles` | `com.corex.iam.directory.read` | 返回完整角色目录，不是仅 provisioning 可选集 |
| `GET /api/v1/tenant/iam/permissions` | `com.corex.iam.directory.read` | 返回 UUID permission 目录 |
| `POST /api/v1/tenant/iam/authorization:check` | `com.corex.iam.authorization.check` | 返回 `200 { allowed: boolean, reason_code }` |

成员接口继续使用既有 `com.corex.iam.members.read`；最终可演进为 `com.corex.iam.directory.read`，但演进必须显式发布并更新所有 plugin credential grant。

## 授权检查请求与语义

```json
{
  "member_uuid": "<uuid>",
  "user_uuid": "<uuid>",
  "resource": "iam.member",
  "action": "read"
}
```

- `resource/action` 必须命中 PowerX 正式 Permission/Capability 目录；拒绝任意字符串探测。
- `member_uuid` 与 `user_uuid` 同时提供时，必须属于同一 tenant member；否则明确失败 `IAM_SUBJECT_MISMATCH`。
- 目标成员存在但不具有该权限是 `200` + `allowed:false`，并返回稳定 `reason_code`。
- 调用插件服务未被授予 Host capability 是 `403 IAM_FORBIDDEN`。

## 稳定错误合同

错误响应必须同时包含 `error_code` 与 `reason_code`：

| 场景 | HTTP | reason_code |
| --- | ---: | --- |
| 缺失或无效身份 | 401 | `IAM_UNAUTHORIZED` |
| 调用主体无 Host capability | 403 | `IAM_FORBIDDEN` |
| 跨 tenant 或不存在成员 | 404 | `IAM_MEMBER_NOT_FOUND` |
| 重复/非法请求 UUID | 400 | `IAM_INVALID_ARGUMENT` |
| `member_uuid`/`user_uuid` 不一致 | 400 | `IAM_SUBJECT_MISMATCH` |
| 数据库或依赖不可用 | 502 或 503 | `IAM_UPSTREAM_DEPENDENCY` |

批量成员查询必须先校验全部 UUID 与重复项；重复 UUID 返回 `400 IAM_INVALID_ARGUMENT`，不得静默去重。

## Capability 与 STS 验收

每个 API 都必须同时验证：

1. tenant STS service actor；
2. 已发布 Capability Record；
3. 当前 tenant 已发布 registration；
4. 调用插件 credential 的 `allowed_capabilities` 明确含对应 capability。

`sts_direct: true` 不能仅作为路由直连 allowlist；必须经过上述 grant 校验。

## PowerX 验收测试

至少覆盖：目录成功、批量漏项、重复 UUID、跨 tenant、缺失/无效身份 401、未授予 capability 403、目标成员 deny、subject mismatch、上游 502/503。完成后运行 PowerX 的 capability 校验与 IAM OpenAPI 合同测试。

## Framework 接续范围

PowerX 发布并提供可调用环境后，Framework 立即将 delegated adapter 的 `ListDepartments`、`ListRoles`、`ListPermissions` 和 `Authorize` 接到以上 API，并增加跨模式合同测试。发布前保持明确的 `IAM_UPSTREAM_DEPENDENCY`，不回退插件本地表。
