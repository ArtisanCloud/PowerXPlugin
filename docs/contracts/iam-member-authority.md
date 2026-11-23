# .NET 成员组织关系依据

`IAMRegistry.GetMemberAuthorityDirectory()` 获取所选 IAM adapter 的正式组织事实接口；没有绑定时明确失败，不回退本地身份或 Admin API。Local提供方实现 `IMemberAuthorityDirectory`，Delegated 使用 `MemberAuthorityClient`，通过固定 `com.corex.iam.organization.service_read`、`core://iam/organization`、`INVOKE`、typed `read_member_authority` 调用。

请求 `powerx.iam.member-authority-query.v1` 包含1至200个唯一规范成员 UUID，租户来自配置服务身份。客户端不发送用户 bearer、调用者 tenant/plugin、动态 HTTP 地址、headers或代理参数。超时、响应大小与查询数量有界。

响应 `powerx.iam.member-authority.v1` 必须逐一覆盖请求成员，校验租户、UUID、实际用户、明确角色、部门闭包、祖先层级/循环及上级/负责人状态。字段缺失、未知 schema、重复/越界、不完整active身份或矛盾关系明确失败，不能把截断结果当作事实。保留源 HTTP错误码、可用trace/request关联；角色事实不是业务授权，调用方继续执行原业务范围和独立性检查。

未配置的关系以unset表达，不从唯一候选、数字ID、姓名或部门父级推测负责人。Core具体服务最小授权、Admin明确维护版本与审计、schema/回退边界见 Core `docs/contracts/iam-member-authority.md` 与OpenAPI。

## Core 错误响应与关联信息

非成功 HTTP 响应统一抛出 `MemberAuthorityException`，`Status` 保留实际收到的 HTTP 状态。只读取正式顶层机器字段：`OriginalCode` 对应 `error_code`，`ReasonCode` 对应 `reason_code`，缺失字段保持 `null`；`Code` 优先取 `reason_code`，其次取 `error_code`。数值型 `code`、字符串 `error`、`message` 和 `details.code` 都不能替代机器错误码，也不读取旧的对象型 `error.code`。异常不包含服务端自由文本或响应正文。

`TraceId` 优先取响应头 `X-Trace-ID`，头部缺失时读取正式响应体的 `details.trace_id`；`RequestId` 优先取 `X-Request-ID`，头部缺失时读取顶层 `request_id`。这些字段仅用于错误关联，不触发重试、身份切换或其他传输方式。

无效 JSON、非对象根节点、错误字段类型、空白机器字段、缺少两种机器错误码或正文超过 1 MiB 时，`Code` 明确为 `IAM_ORGANIZATION_RESPONSE_INVALID`，仍保留收到的 HTTP 状态、头部关联信息及可读取的正式诊断字段。不会把这类响应当成空组织事实，也不会静默归类为普通依赖不可用。

Framework 无法还原被 Core 调用入口改写的 HTTP 状态。Core 的 `IAM_ORGANIZATION_*` 状态映射属于独立修复项；例如实际收到 502 且业务码为 `IAM_ORGANIZATION_SCHEMA_NOT_READY` 时，本客户端保留两者，不自行推测或改成 503。当前契约测试覆盖不同 HTTP 状态、正式字符串错误响应、头部与正文关联、独立错误码以及无效和超限响应。

2026-11-15：成员组织关系测试 37 项通过，完整 .NET 测试集 336 项通过、2 项依赖独立 Core 通知环境的测试跳过。另通过临时 Go overlay 启动实际 Core Gin 能力调用路由，以受控调用器注入组织类 400／403／503 错误，并由本 SDK 通过真实回环 HTTP 发起 6 个场景，分别覆盖头部和正文关联信息，全部通过；当前 Core 路由确实将这些错误映射为 502，Framework 保留收到的状态及原业务码。该验证不连接数据库，不证明真实服务凭证、授权或 CRM 工作流验收；未修改 Core 源码、日常服务和业务数据，临时进程已退出、6 个监听端口已释放，未发布 SDK 包。

2026-11-12：Framework新增13项组织契约用例通过，全量297项通过，1项实际Core通知fixture用例在常规全量中跳过并在临时服务中单独运行通过。CRM实际LocalRegistry关系配置→接收人重读及原生关系审计回滚已验证；未发布SDK包、修改日常组织/授权或完成真实宿主/客户验收。
