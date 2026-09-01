# PowerX Plugin Release Host Contract 修订要求

Framework 暂不为当前 Plugin Release tenant 路由提供客户端。原因不是缺少 transport，而是现有接口违反插件间的 UUID-only 与 tenant-scoped 合同，直接封装会把安全和数据模型问题带入已发布 SDK。

## 必须由 PowerX 修订

1. `POST /api/v1/tenant/plugin-release/local/sessions` 不再接收 `tenant_uuid` 与 numeric `developerId`。
   - tenant 只能从已认证的 STS service actor / Gateway credential 上下文获得。
   - developer 身份应由 service actor 推导；若确实需要显式目标，字段必须为 `developer_member_uuid`，并验证属于当前 tenant。
2. `LocalInstallSession.DeveloperID uint64` 和关联的 repository/service/permission/cache 键迁移为 `developer_member_uuid`；不新增 numeric 兼容输入。
3. `GET` / `DELETE /api/v1/tenant/plugin-release/local/sessions/{session_uuid}` 必须以 credential tenant 查询并验证归属。跨 tenant 与不存在统一返回 404，避免泄露对象存在性。
4. `POST /api/v1/tenant/offline-imports` 也不得接受 body `tenant_uuid`；`GET /api/v1/tenant/offline-imports/{job_uuid}` 必须用 tenant 范围查询。返回字段统一为 `job_uuid`，不得暴露非 UUID 的临时 ID。
5. 所有五个路由使用稳定错误信封和 `reason_code`：至少覆盖 401 未认证、403 capability 未授予、404 对象不存在/跨租户、409 活跃本地安装会话、422 签名或校验失败、503 依赖不可用。
6. 发布独立 capability：目录读取与有副作用的本地安装/离线导入必须分离，并确认 `sts_direct: true` 同时执行 tenant registration/grant 检查。
7. 更新 OpenAPI 到最终 UUID DTO，并增加 Host 合同测试：成功、跨 tenant、无 capability、body 伪造 tenant、numeric developer 输入拒绝、无效 UUID、状态查询与停止操作。

## Framework 后续接入范围

PowerX 修订并发布上述合同后，Framework 将新增 `runtime/powerx/pluginrelease`：本地安装会话创建/查询/停止与离线导入创建/查询，复用 Skeleton delegated 模式的共享 STS token provider，并补齐 transport 测试。不会为当前 numeric/body-tenant 协议提供兼容分支。
