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
| Agent | 18（6 lifecycle + 12 Session） | `runtime/powerx/agent`、`runtime/agent` | ready_for_integration | SessionService 已接入独立 UUID-only STS Host：会话/消息管理、幂等 Invoke、查询/取消、state/final/error/end 订阅。Runtime.Sessions 选择插件 local 实现或 Core client，Skeleton 已装配。旧人工 Invoke/StreamSSE 在 delegated 下明确拒绝；不把订阅当执行，不提供 API Key Session 或 numeric 兼容。HTTP/Factory 测试已覆盖，Framework 安装态联调后置；Core P2 剩余授权审计不作已验收声明。 |
| AI | 11 | `runtime/powerx/ai`、`runtime/ai` | ready_for_integration | `GenerativeService` 已覆盖 LLM、模型、会话/会话流、Embedding、VLM、Image、Video、TTS；Factory 选择插件 local adapter 或 typed STS Core client，且无请求时模式切换。安装态 capability grant 联调后置。 |
| Capability Registry | 5 | `runtime/powerx/capability`、`runtime/capability` | ready_for_integration | 五项 typed client/Factory 已有，Core 已交付当前凭证目录过滤、实时目标 grant 与 trace 主体隔离；method/path 保持一致。Framework 不缓存授权，403 不回退 local。Core 部署及安装态撤权验收仍待执行。 |
| Integration Gateway | 3 | `runtime/powerx/integration`、`runtime/integration` | ready_for_integration | 三项 typed client/Factory 已有；Core 已交付路由目标 grant 与 tool grant 校验、目录过滤。Framework 403 保留且不回退 local；安装态授权验收后置，不编造独立网关 capability。 |
| Skills | 1 | `runtime/powerx/skills`、`runtime/skills` | ready_for_integration | 已覆盖 tenant Skill direct invoke；Factory、共享 STS transport、结构化 Core reason_code 及测试已齐全。local invoker 由插件注入。 |
| Notifications | 1 | `runtime/powerx/notifications`、`runtime/notifications` | ready_for_integration | 已覆盖 tenant 通知创建；Factory、共享 STS transport、结构化 Core reason_code 及测试已齐全。local publisher 由插件注入。 |
| Customer Auth | 3 | `runtime/customerfw` | ready_for_integration | 已覆盖 Core Shopify storefront register/login/validate 与 customer JWT 双凭证校验；`CustomerRuntime` 按可信模式选择插件 local store 或 delegated Core Auth/Membership adapter。delegated 不接受请求注入 tenant/customer UUID，也不回退插件客户表；客户端、credential 转发和 Factory 均有定向测试。 |
| Plugin Release | 5 | `runtime/pluginrelease`、`runtime/powerx/pluginrelease` | ready_for_integration | 已覆盖 UUID-only install session 与 import job Host Contract。delegated `Client` 从 STS 推导 service actor/plugin，使用 Core 托管 signing key 信任链；`Runtime` 选择插件注入 local `Service` 或 Core client，定向 HTTP/Factory 测试已覆盖。真实已安装插件 capability grant 联调后置。 |
| Plugin Runtime | 3 | `runtime/powerx/pluginruntime`、`runtime/pluginruntime` | ready_for_integration | 已覆盖知识空间列表、Agent 实例化和 Agent 列表的 UUID DTO；Factory、共享 STS transport、结构化 Core reason_code 及测试已齐全。local Service 由插件注入，安装态 capability grant 联调后置。 |
| Metadata Governance | 18 | `runtime/metadata` | ready_for_integration | 已覆盖正式 tenant Metadata Host Contract：字典、字典项、分类/节点、标签、UUID 化 tag binding、资源类型及已声明的 PATCH 操作。`HostClient` 仅携带 STS 服务凭证调用 `/api/v1/tenant/metadata/*`，不再调用 admin 路由；`Runtime` 由可信 `ProviderMode` 选择插件注入 local `Service` 或 delegated client。`METADATA_*` 稳定错误与 UUID tag-binding 创建/删除已有定向测试；真实已安装插件的 capability grant 联调后置。 |

## 消费者审计补充（2026-09-08）

IAM 上表的 ready_for_integration **仅指目录读取、授权判定与身份上下文**，不包括部门/成员组织写入和后台登录/刷新/登出。SCRM delegated 组织写入当前缺 Core Host Contract，也缺 Framework 写 contract/Factory，状态为 not_started；返回明确不可用是安全措施，不算功能完成。

Media、组织写入和 Capability/Integration 的精确范围及 Core/Framework/插件分工见 [消费者缺口任务单](framework-consumer-contract-gaps.md)。本次纠正先前“Variant 全量”“只等插件安装联调”的过宽声明；不撤销已有操作的代码与定向测试成果。

## 现有非本表运行时开放面的客户端

2026-09-09 新增 Cache / TaskCenter 正式 Host Contract 对齐（安装态验收后置）：

| 模块 | 包 | 状态 | 已交付 / 未交付 |
|---|---|---|---|
| Cache | runtime/cache | ready_for_integration | 三项正式 tenant Host 操作、STS 凭证租户匹配、typed DTO、错误与信封校验、单选 Factory 和合同测试已交付。local 持久化由插件实现；Core 部署与安装态验收后置。 |
| TaskCenter | runtime/taskcenter | ready_for_integration | 三项正式 tenant Host 操作、STS 凭证租户匹配、typed DTO、错误与信封校验、单选 Factory 和合同测试已交付。local 持久化由插件实现；Core 部署与安装态验收后置。 |

精确语义、Core 交付和电商适配要求见 [Cache / TaskCenter 合同任务单](cache-taskcenter-host-requirements.md)。delegated transport 已实现；目标环境仍需部署、授权和安装态验收。不使用 taskqueue 代替任务记录，不猜内部 URL。

`runtime/scheduler`、`runtime/wsbus`、`runtime/customerfw` 的 Admin 等表外客户端、`runtime/aisettings`、`runtime/taskqueue` 仍需保留独立审计行。它们可能面向 Core Admin 或运行时接口，但不得被计入上表“当前 OpenAPI 开放面已覆盖”。

## 发布门槛

新增或变更 PowerX Core 对接时，必须同步更新本台账，并满足：

1. Core 权威路由或正式 OpenAPI 已确认；
2. UUID DTO、鉴权、租户隔离和错误码已冻结；
3. local/delegated 的支持范围分别记录；
4. transport 与合同测试均已落地；
5. 业务插件不直接查询宿主 IAM 表、不解析 Gateway 原始 `map[string]any`、不以 UUID 充当显示名称。
