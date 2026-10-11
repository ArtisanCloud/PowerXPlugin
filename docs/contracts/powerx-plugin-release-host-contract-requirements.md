# PowerX Plugin Release Host Contract 修订要求

Framework 暂不为当前 Plugin Release tenant 路由提供客户端。原因不是缺少 transport，而是现有接口违反插件间的 UUID-only 与 tenant-scoped 合同，直接封装会把安全和数据模型问题带入已发布 SDK。

## 2026-09-03 复核结果

Core 已完成一部分 UUID/tenant 修订：本地安装的创建请求不再接收 `tenant_uuid` 或 numeric `developerId`，而是从请求上下文取得 tenant 与当前 member；本地安装 session 和离线导入 job 的持久化标识也已经是 UUID，并以 tenant 范围查询。

但该路由组仍不是可供插件 service actor 调用的正式 Host Contract：没有独立 capability/grant 校验；本地安装要求请求上下文中存在人类 member UUID；离线导入仍以 `jobId`、camelCase 字段和非稳定错误文本对外；五个路由未形成统一 OpenAPI/错误信封合同。因此 Framework 继续保持不接入。

## 仍必须由 PowerX 完成

1. 明确 service actor 的业务语义：不能从 STS 插件凭证假定存在人类 `member_uuid`。创建本地安装应由 service actor 作为审计主体；如需管理员审批，改为显式的 member-scoped Admin Contract，而不是让插件伪造 member。
2. 发布独立 capability，并按最小权限拆分只读状态查询与有副作用的本地安装/离线导入；Gateway API Key 与 STS 均必须校验发布记录、tenant registration 和 credential grant。`sts_direct: true` 不能替代 grant 校验。
3. 统一全部五个路由的 UUID DTO 与命名：路径参数为 `session_uuid`/`job_uuid`，响应字段同名；不得再对外输出 `sessionId`、`jobId` 或 body tenant。
4. 所有路由使用稳定错误信封和 `reason_code`：至少覆盖 401 未认证、403 capability 未授予、404 对象不存在/跨租户、409 活跃本地安装会话、422 签名或校验失败、503 依赖不可用。
5. 发布最终 OpenAPI 和 Host 合同测试：成功、跨 tenant、无 capability、service actor 调用、body 伪造 tenant、无效 UUID、状态查询与停止操作。

## Framework 后续接入范围

PowerX 修订并发布上述合同后，Framework 将新增 `runtime/powerx/pluginrelease`：本地安装会话创建/查询/停止与离线导入创建/查询，复用 Skeleton delegated 模式的共享 STS token provider，并补齐 transport 测试。不会为当前 numeric/body-tenant 协议提供兼容分支。
