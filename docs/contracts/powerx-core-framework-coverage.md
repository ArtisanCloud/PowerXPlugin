# PowerX Core 开放面与 Framework 覆盖台账

## 目的与判定规则

本台账是 PowerXPlugin framework 对 PowerX Core 业务对象封装的唯一审计入口。它不以 package、接口或 Registry 的存在作为完成依据；只有同时具备稳定 DTO、业务接口、实际 transport、模式装配和合同测试，才可以标记为 `implemented`。

- **权威开放面**：以 Core `backend/internal/transport/http/openapi/routes.go` 实际注册的路由为准。
- **生成 Swagger 不是全量依据**：当前 `backend/api/openapi/swagger.json` 仅含模型路由，不能据此声明 SDK 已覆盖全量开放面。
- **规格 OpenAPI 不是自动上线证明**：`specs/*/contracts` 的条目必须另标注为 `active`、`design_only`、`admin_only` 或 `retired`。
- **状态定义**：`implemented`、`ready_for_integration`、`partial`、`contract_only`、`generic_only`、`not_started`。`ready_for_integration` 表示稳定 DTO、业务接口、实际 transport、模式装配和单元/合同测试均已完成，真实业务插件安装联调后置；禁止以空实现、内存 mock 或调用方自行注入的接口标记为该状态。

## 当前运行时开放面（审计基线）

| Core 模块 | 当前路由数 | Framework 对应包 | 状态 | 事实与缺口 |
|---|---:|---|---|---|
| Media | 11 | `media` | implemented | 已覆盖资产创建、列表、查询、更新、删除、预签名和 Variant 创建/预签名；二进制读取统一通过预签名 URL 的 `DownloadBytes`，插件不直接拼资源路径。 |
| IAM | 10 | `iam/contracts`、`iam/adapters` | ready_for_integration | 已覆盖 credential-scoped tenant、显式分页成员目录、单成员/严格批量/容错批量 UUID 查询、精确显示名解析、部门、角色、权限与 `authorization:check`。Framework local/delegated DTO、Bootstrap、Gateway API Key/STS transport 和错误映射均有合同测试；Skeleton local 管理 API、模型迁移和部门/成员/角色/权限关联均已 UUID-only。真实已安装插件的 capability grant 联调后置。 |
| Knowledge Space | 9 | `runtime/knowledge`、`runtime/powerx/knowledge` | ready_for_integration | 已覆盖 QA Retrieval Plan、Memory Snapshot，以及正式 tenant Host Contract 的空间列表、Search、文档异步写入/删除、空间索引重建和索引任务查询。Skeleton delegated 模式复用共享 STS token，Host DTO/错误映射/本地模式任务语义均有测试；搜索筛选与文档级重建未被 Core 声明时明确失败。2026-09-03：Core 已发布三项 capability 与 API-Key 精确 scope；Skeleton 当前开发 Key 返回 `403 KNOWLEDGE_FORBIDDEN`，须改用已授权专用 Key，或在安装态由清单 consumes + 插件升级同步 STS grant 后再做真实联调。 |
| Agent | 6 | `runtime/powerx/agent` | ready_for_integration | 已覆盖健康摘要/历史、Bridge 状态、冻结、恢复与扩缩容；Skeleton 已装配 tenant-scoped STS client，并有配置、SSE、WS、生命周期与路由回归。真实安装后的 Host capability/授权联调后置。 |
| AI | 11 | `runtime/powerx/ai` | ready_for_integration | 已覆盖 LLM、模型、会话、Embedding、VLM/图像/视频/TTS 与两个 SSE 流接口；Skeleton 已注入 STS token provider，typed transport 测试通过。具体业务消费和 Core 会话持久性属于后续联调，不阻塞 Framework 代码版本。 |
| Capability Registry | 5 | `runtime/powerx/capability` | ready_for_integration | 已覆盖租户 capability 目录、REST 路由解析、调用及调用追踪；Skeleton delegated 模式已注入共享 STS service token，transport 测试覆盖成功与 403 错误信封。具体 capability 的 payload schema 和已安装插件授权联调后置。 |
| Integration Gateway | 3 | `runtime/powerx/integration` | ready_for_integration | 已有路由列表、详情和调用的强类型 tenant SDK 与 transport 测试；真实 capability grant 和已安装插件联调后置。 |
| Skills | 1 | `runtime/powerx/skills` | ready_for_integration | 已覆盖 tenant skill direct invoke，外层 DTO 固定，payload/context/result 保持 capability/skill 自身声明的 JSON 对象；Skeleton delegated 模式已注入共享 STS service token，transport 测试覆盖成功与 403 错误信封。具体 skill 注册和安装授权联调后置。 |
| Notifications | 1 | `runtime/powerx/notifications` | ready_for_integration | 已覆盖 tenant 通知创建；Skeleton delegated 模式已注入共享 STS service token，transport 测试覆盖成功与 403 错误信封。通知文案与 metadata 由调用插件的业务 i18n/schema 决定，真实权限授权联调后置。 |
| Plugin Release | 5 | 无对应 tenant release client | partial | Core 已注册本地安装会话与离线导入路由，但现行 Host Contract 仍接收 numeric `developerId`、信任 body `tenant_uuid`，且 GET 未按 tenant 过滤对象；不符合 UUID-only/tenant-scoped 约束，Framework 不会固化该协议。修订项见 `powerx-plugin-release-host-contract-requirements.md`。 |
| Plugin Runtime | 3 | `runtime/powerx/pluginruntime` | ready_for_integration | 已有知识空间列表、Agent 实例化与 Agent 列表的强类型客户端及 UUID DTO transport 测试；正式 capability grant 和安装联调后置。 |

## 现有非本表运行时开放面的客户端

`runtime/metadata`、`runtime/scheduler`、`runtime/wsbus`、`runtime/customerfw`、`runtime/aisettings`、`runtime/taskqueue` 仍需保留独立审计行。它们可能面向 Core Admin 或运行时接口，但不得被计入上表“当前 OpenAPI 开放面已覆盖”。

## 发布门槛

新增或变更 PowerX Core 对接时，必须同步更新本台账，并满足：

1. Core 权威路由或正式 OpenAPI 已确认；
2. UUID DTO、鉴权、租户隔离和错误码已冻结；
3. local/delegated 的支持范围分别记录；
4. transport 与合同测试均已落地；
5. 业务插件不直接查询宿主 IAM 表、不解析 Gateway 原始 `map[string]any`、不以 UUID 充当显示名称。
