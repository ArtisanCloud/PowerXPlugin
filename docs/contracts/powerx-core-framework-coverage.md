# PowerX Core 开放面与 Framework 覆盖台账

插件接入统一见 [Framework 业务模块接入指南](../guides/features/009-consume-powerx-capability/guide.md)。本台账只记录实现范围和验收证据，不替代操作手册或已发布版本说明。

## 目的与判定规则

本台账是 PowerXPlugin framework 对 PowerX Core 业务对象封装的唯一审计入口。它不以 package、接口或 Registry 的存在作为完成依据；只有同时具备稳定 DTO、业务接口、实际 transport、模式装配和合同测试，才可以标记为 `implemented`。

- **权威开放面**：以 Core `backend/internal/transport/http/openapi/routes.go` 实际注册的路由为准。
- **生成 Swagger 不是全量依据**：当前 `backend/api/openapi/swagger.json` 仅含模型路由，不能据此声明 SDK 已覆盖全量开放面。
- **规格 OpenAPI 不是自动上线证明**：`specs/*/contracts` 的条目必须另标注为 `active`、`design_only`、`admin_only` 或 `retired`。
- **状态定义**：`implemented`、`ready_for_integration`、`partial`、`contract_only`、`generic_only`、`not_started`。`ready_for_integration` 表示稳定 DTO、业务接口、实际 delegated transport、模式装配和单元/合同测试均已完成，真实业务插件安装联调后置。local 的权威持久化由插件注入同一 contract；Framework 只负责 Factory 选择，不能以空实现或内存 mock 冒充 local adapter。

## 当前运行时开放面（审计基线）

| Core 模块 | 当前路由数 | Framework 对应包 | 状态 | 事实与缺口 |
|---|---:|---|---|---|
| Media | 13 | `runtime/media`、`runtime/powerx/media` | ready_for_integration | Asset 与 Variant 三项传输已接入：创建必填 checksum，响应 status/completed_at；票据 TTL、完成响应与错误传播有测试。local Service 新增三项方法，需插件补齐。Core 已交付代码但尚未迁移/seed/重启，实际传输、票据撤销与安装验收后置。 |
| IAM | 10 | `iam/contracts`、`iam/adapters` | ready_for_integration | 已覆盖 credential-scoped tenant、显式分页成员目录、单成员/严格批量/容错批量 UUID 查询、精确显示名解析、部门、角色、权限与 `authorization:check`。Framework local/delegated DTO、Bootstrap、Gateway API Key/STS transport 和错误映射均有合同测试；Skeleton local 管理 API、模型迁移和部门/成员/角色/权限关联均已 UUID-only。真实已安装插件的 capability grant 联调后置。 |
| Knowledge Space | 9 | `runtime/knowledge`、`runtime/powerx/knowledge` | ready_for_integration | 已覆盖 QA Retrieval Plan、Memory Snapshot，以及正式 tenant Host Contract 的空间列表、Search、文档异步写入/删除、空间索引重建和索引任务查询。Skeleton delegated 模式复用共享 STS token，Host DTO/错误映射/本地模式任务语义均有测试；搜索筛选与文档级重建未被 Core 声明时明确失败。2026-09-03：Core 已发布三项 capability 与 API-Key 精确 scope；当日 Skeleton 开发 Key 返回 `403 KNOWLEDGE_FORBIDDEN`，该历史结果不代表当前 Key 状态。安装态由 capabilities.required + 插件升级同步 STS grant 后另行验收，API Key 验证单列记录。 |
| Agent | 18（6 lifecycle + 12 Session） | `runtime/powerx/agent`、`runtime/agent` | partial | .NET 已有 Lifecycle 与 Session UUID DTO、显式 `ILocalAgentLifecycleStore` / `ILocalAgentStore`、启动期 local/delegated 选择点；Session 的 N402 delegated 12 项 transport 与 SSE `Last-Event-ID` 恢复已实现。Lifecycle 六项 delegated typed client 已对齐 Core OpenAPI 路由、STS Bearer 服务凭证、请求体和响应 DTO；定向 HTTP 测试覆盖六项操作、租户阻断、403、非法响应和取消。Framework 本身不提供插件的生产 local store，尚缺实际插件消费、安装态 grant/撤权及跨租户联调，故整体保持 partial，不以 local fallback 冒充成功。 |
| AI | 11 | `runtime/powerx/ai`、`runtime/ai` | ready_for_integration | `GenerativeService` 已覆盖 LLM、模型、会话/会话流、Embedding、VLM、Image、Video、TTS；Factory 选择插件 local adapter 或 typed STS Core client，且无请求时模式切换。安装态 capability grant 联调后置。 |
| Capability Registry | 5 | `runtime/powerx/capability`、`runtime/capability` | ready_for_integration | 五项 typed client/Factory 已有，Core 已交付当前凭证目录过滤、实时目标 grant 与 trace 主体隔离；method/path 保持一致。Framework 不缓存授权，403 不回退 local。Core 部署及安装态撤权验收仍待执行。 |
| Integration Gateway | 3 | `runtime/powerx/integration`、`runtime/integration` | ready_for_integration | 三项 typed client/Factory 已有；Core 已交付路由目标 grant 与 tool grant 校验、目录过滤。Framework 403 保留且不回退 local；安装态授权验收后置，不编造独立网关 capability。 |
| Skills | 1 | `runtime/powerx/skills`、`runtime/skills` | ready_for_integration | 已覆盖 tenant Skill direct invoke；Factory、共享 STS transport、结构化 Core reason_code 及测试已齐全。local invoker 由插件注入。 |
| Notifications | 1 | `runtime/powerx/notifications`、`runtime/notifications` | ready_for_integration | 已覆盖 tenant 通知创建；Factory、共享 STS transport、结构化 Core reason_code 及测试已齐全。local publisher 由插件注入。 |
| Customer Auth | 3 | `runtime/customerfw` | ready_for_integration | 已覆盖 Core Shopify storefront register/login/validate 与 customer JWT 双凭证校验；`CustomerRuntime` 按可信模式选择插件 local store 或 delegated Core Auth/Membership adapter。delegated 不接受请求注入 tenant/customer UUID，也不回退插件客户表；客户端、credential 转发和 Factory 均有定向测试。 |
| Plugin Release | 5 | `runtime/pluginrelease`、`runtime/powerx/pluginrelease` | ready_for_integration | 已覆盖 UUID-only install session 与 import job Host Contract。delegated `Client` 从 STS 推导 service actor/plugin，使用 Core 托管 signing key 信任链；`Runtime` 选择插件注入 local `Service` 或 Core client，定向 HTTP/Factory 测试已覆盖。真实已安装插件 capability grant 联调后置。 |
| Plugin Runtime | 3 | `runtime/powerx/pluginruntime`、`runtime/pluginruntime` | ready_for_integration | 已覆盖知识空间列表、Agent 实例化和 Agent 列表的 UUID DTO；Factory、共享 STS transport、结构化 Core reason_code 及测试已齐全。local Service 由插件注入，安装态 capability grant 联调后置。 |
| Metadata Governance | 18 | `runtime/metadata` | ready_for_integration | 已覆盖正式 tenant Metadata Host Contract：字典、字典项、分类/节点、标签、UUID 化 tag binding、资源类型及已声明的 PATCH 操作。`HostClient` 仅携带 STS 服务凭证调用 `/api/v1/tenant/metadata/*`，不再调用 admin 路由；`Runtime` 由可信 `ProviderMode` 选择插件注入 local `Service` 或 delegated client。`METADATA_*` 稳定错误与 UUID tag-binding 创建/删除已有定向测试；真实已安装插件的 capability grant 联调后置。 |

### Agent Lifecycle Go/.NET 合同对照（2026-10-01）

以下六项使用 Core `agent_lifecycle.yaml` 的 `/api/v1/openapi/agents/{agent_id}` 路由。
Go `runtime/powerx/agent/lifecycle.go` 与 .NET `PowerXAgentLifecycleClient` 均从启动期持有的
服务凭证发包，不发送请求提供的 tenant。Core `requireServiceTenantAgent` 要求 STS service actor，
并从可信 claims 取得 tenant；.NET 另在发包前核对调用 scope 与已配置 tenant。

| 操作 | Core 后缀 | Go 方法/响应 | .NET 方法/响应 |
|---|---|---|---|
| GET 健康摘要 | `/health/summary` | `GetHealthSummary` / `HealthSummary` | `GetHealthSummaryAsync` / `AgentHealthSummary` |
| GET 健康历史 | `/health/history` | `ListHealthHistory` / `HealthHistory` | `ListHealthHistoryAsync` / `AgentHealthHistory` |
| GET 桥接状态 | `/bridge/state` | `GetBridgeState` / `json.RawMessage` | `GetBridgeStateAsync` / `AgentBridgeState`，无快照时 `Health=null` |
| POST 冻结 | `/bridge/freeze` | `Freeze` / `BridgeLifecycleResult` | `FreezeAsync` / `AgentBridgeLifecycleResult` |
| POST 恢复 | `/bridge/recover` | `Recover` / `BridgeLifecycleResult` | `RecoverAsync` / `AgentBridgeLifecycleResult` |
| POST 重平衡 | `/bridge/rebalance` | `Rebalance` / `BridgeLifecycleResult` | `RebalanceAsync` / `AgentBridgeLifecycleResult` |

Go 定向测试 `TestLifecycleClientUsesTenantScopedCoreRoutes` 覆盖摘要、历史和冻结，
`TestLifecycleClientMapsHostAuthorizationAndAvailabilityErrors` 覆盖 403/503。
.NET `PowerXAgentLifecycleClientTests` 覆盖六项路由、请求体、返回字段、跨租户拒绝、
403、非法响应、STS-only 凭证和取消；`AgentRuntimeTests` 覆盖 local/delegated 单选、
缺失 adapter 和 delegated 403 零 local 调用。错误分别保持 Go `ErrCodeForbidden` /
`ErrCodeUnavailable` 与 .NET `FRAMEWORK_AGENT_FORBIDDEN` /
`FRAMEWORK_AGENT_UPSTREAM_DEPENDENCY`，不将传输错误回退到 local。

## 消费者审计补充（2026-09-08）

IAM 上表的 ready_for_integration **仅指目录读取、授权判定与身份上下文**，不包括部门/成员组织写入和后台登录/刷新/登出。SCRM delegated 组织写入当前缺 Core Host Contract，也缺 Framework 写 contract/Factory，状态为 not_started；返回明确不可用是安全措施，不算功能完成。

成员状态以 Core 的传输契约为准：成员列表、详情和严格批量查询的 `status` 是 JSON 数字（`1=启用、2=停用`）。Go/.NET Framework 的 delegated transport 使用独立数字 DTO 接收，再转换为公开 `Member.Status` 的字符串表示 `"1"`/`"2"`，插件无需自行兼容数字和字符串。宿主返回字符串状态视为契约错误。Go 的容错批量解析 `members:batch-resolve` 按 Core 的精简 DTO 接收 UUID、用户 UUID、名称和缺失列表；租户使用调用时可信租户上下文，未返回的状态保持空值。数字响应、错误格式和跨租户响应有定向测试；本次修正尚未完成真实 API Key/STS 验收。

Media、组织写入和 Capability/Integration 的精确范围及 Core/Framework/插件分工见 [消费者缺口任务单](framework-consumer-contract-gaps.md)。本次纠正先前“Variant 全量”“只等插件安装联调”的过宽声明；不撤销已有操作的代码与定向测试成果。

## 现有非本表运行时开放面的客户端

2026-09-09 新增 Cache / TaskCenter 正式 Host Contract 对齐（安装态验收后置）：

| 模块 | 包 | 状态 | 已交付 / 未交付 |
|---|---|---|---|
| Cache | runtime/cache | ready_for_integration | 三项正式 tenant Host 操作、STS 凭证租户匹配、typed DTO、错误与信封校验、单选 Factory 和合同测试已交付。local 持久化由插件实现；Core 部署与安装态验收后置。 |
| TaskCenter | runtime/taskcenter | ready_for_integration | 三项正式 tenant Host 操作、STS 凭证租户匹配、typed DTO、错误与信封校验、单选 Factory 和合同测试已交付。local 持久化由插件实现；Core 部署与安装态验收后置。 |

精确语义、Core 交付和电商适配要求见 [Cache / TaskCenter 合同任务单](cache-taskcenter-host-requirements.md)。delegated transport 已实现；目标环境仍需部署、授权和安装态验收。不使用 taskqueue 代替任务记录，不猜内部 URL。

`runtime/scheduler`、`runtime/wsbus`、`runtime/customerfw` 的 Admin 等表外客户端、`runtime/aisettings`、`runtime/taskqueue` 仍需保留独立审计行。它们可能面向 Core Admin 或运行时接口，但不得被计入上表“当前 OpenAPI 开放面已覆盖”。

2026-10-02 .NET 表外 local 补齐：`LocalEventFabricAdapter` 在序列化 payload 前按
元数据 tenant 筛选订阅；`LocalEventFabricAdapterTests` 覆盖共享主题的双租户订阅、
缺租户元数据不投递及其他租户 payload 零读取。CRM local 消费改用 `IEventRuntime`，
绑定配置的可信 tenant 并再次检查 payload tenant。`LocalScheduler` 按 tenant + job_id
寻址，`LocalSchedulerTests` 覆盖 Get/Update/Pause/Resume/Trigger 的跨租户拒绝、同名
任务独立及缺失/非法 tenant UUID 的明确错误。Go 对照为 `runtime/scheduler/local_provider.go`
的 GetJob/setPaused/TriggerJob 租户检查；.NET 同名任务额外按租户保存，避免覆盖。
`ListJobsAsync(null)` 保留 local runner 枚举全部任务的内部用法，不计作租户 API 验收。
本条不代表 Scheduler Admin 接口的服务身份授权或 Core Event Fabric 安装态已通过。

## 发布门槛

2026-10-03 .NET local Scheduler runner 补齐：任务失败相互隔离，失败保留 scheduled
time 与幂等键，once 成功后 completed，once 时间规范为 UTC。`SchedulerRunnerTests`
覆盖失败重试、其他租户继续、一次性完成、时区和取消；CRM
`PoolReclaimFrameworkIntegrationTests` 覆盖 local + proxy=1 的真实本地派发链路，
以及模拟 Core RPC 的 Ack 丢失重连、去重和 Nack。这不是持久化 Scheduler store
或真实 Core 安装态验收；当前 LocalScheduler 任务仍存于进程内存。

同日后续已增加 `ILocalSchedulerStore` 与 revision CAS 的 local Scheduler 存储接口。
CRM 通过显式 local store factory 注入 PostgreSQL `CrmSchedulerStore`；任务 UUID、
payload、暂停/完成、到期与运行时间持久化，失败重启后复用幂等键。独立快照不再
暴露内存对象作为写接口。真实 PostgreSQL 测试仅创建/清理临时 schema，验证恢复与
并发版本检查；默认内存 factory 仍仅提供开发/测试基线。不代表服务重启、安装态或
多副本调度已经验收。

2026-10-02 .NET IAM 消费补充：`PowerXIamClient.ListMembers` 按正式 Core
`items/pagination` 遍历所有 200 条分页。`PowerXIamClientTests` 验证 201 成员跨页、
缺分页、重复、截断、总数变化、错误页号和后续页 403，失败不返回部分目录。
CRM `ICrmIamDirectory` 保留内部数字键，通过 Registry 获取名称和有效状态；
delegated 使用显式 UUID 映射，已释放 DbContext 的测试证明没有 local IAM 读取。
CRM 本机经完整备份独立修复 9 行历史 UUID，六类复核均无缺失、非法或重复。
完整回归 Framework 92/92、CRM 95/95；这些结果不改变安装态 `ready_for_integration`
边界，实际 Core grant/撤权和跨租户联调尚未完成。

新增或变更 PowerX Core 对接时，必须同步更新本台账，并满足：

1. Core 权威路由或正式 OpenAPI 已确认；
2. UUID DTO、鉴权、租户隔离和错误码已冻结；
3. local/delegated 的支持范围分别记录；
4. transport 与合同测试均已落地；
5. 业务插件不直接查询宿主 IAM 表、不解析 Gateway 原始 `map[string]any`、不以 UUID 充当显示名称。

### Knowledge Host provisioning 接入补充（2026-10-04）

Go / .NET 已按 Core `knowledge-host-provisioning.md` 对齐 tenant catalog/create 服务合同：typed DTO、正式 REST transport、API Key/STS 凭证注入、catalog/create 能力声明、策略可用性/Profile/模板 UUID/配额映射、原 HTTP 状态与 machine code/Trace ID，以及合同测试。Local 不添加 Core 创建 fallback。Knowledge 仍为 ready_for_integration；Core 运行迁移、显式 grant、真实 STS Exchange/创建复读与 Skeleton 浏览器写入闭环尚待完成。
