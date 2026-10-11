# Framework Knowledge Runtime

`runtime/knowledge` provides provider-neutral contracts for plugin knowledge search, document indexing, Agent/Skill retrieval, diagnostics, and test fixtures.

The framework owns the generic runtime surface only:

- local provider for standalone development and repeatable tests
- delegated provider adapter contract for PowerX-hosted or proxy mode
- mock provider for deterministic plugin tests
- tenant, citation, redaction, and stable error handling

Production knowledge authority remains PowerX Core or a configured delegated provider. Local and mock providers are blocked in production unless a break-glass policy is explicitly enabled and auditable.

插件业务服务应只依赖 `KnowledgeProvider`；Provider 的 local/delegated 选择属于 Framework Runtime Factory，而不是页面或业务 service。历史 QA bridge 与正式 PowerX Host Contract 必须保持独立：前者不得被当作 delegated 生产能力的 fallback。共同规则见 [Framework 双模式业务模块规范](../../../../../docs/guides/develop/framework-dual-mode-business-modules.md)。

## Knowledge Host provisioning（2026-10-04）

按 Core `docs/contracts/knowledge-host-provisioning.md` 及 `specs/011-knowledge-space/contracts/host-provisioning.openapi.yaml` 接入正式服务合同：

- `runtime/powerx/knowledge.Client.GetKnowledgeCatalog`：固定 `GET /api/v1/tenant/knowledge/catalog`，读取 `data.catalog`，保留策略可用性、缺失原因、激活依赖、Profile UUID/版本、策略模板 UUID 和配额合同。
- `Client.CreateKnowledgeSpace`：固定 `POST /api/v1/tenant/knowledge/spaces`，发送 `CreateSpaceInput`，读取 `data.item`。必填 name/department_uuid/strategy_key；其他 UUID、scene_key、quotas 未设置则省略，由 Core 使用合同默认值。租户、actor、任意 endpoint/header 不属于请求 DTO。
- `DelegatedProvider` 声明 `catalog`/`create`；通过显式 `SpaceProvisioningProvider` 扩展调用 `CreateSpace`。Local/Mock 不自动获得 Core 创建或降级能力。
- Local+proxy 使用 `NewClientWithAPIKey`；Delegated 使用 `NewClientWithTokenProvider` 或已配置 STS 的 `NewClient`。capabilities 表示客户端实现了该操作，不代表凭证已授权。
- Go 错误读取 `CodeOf(err)`、`HTTPStatus(err)` 和 `Error.TraceID`；不要仅按 Code 推算原上游状态。Core 409/412、未知 machine code、503 和 Trace/request ID 原样保留。
- .NET `PowerXKnowledgeClient` 实现 `IKnowledgeProvisioningService`，提供 `GetCatalogAsync`/`CreateSpaceAsync`；DTO 在 `KnowledgeProvisioning.cs`。`KnowledgeException` 提供 `Code`、`StatusCode`、`TraceId`。凭证继续通过 `IServiceCredentialProvider` 注入 API Key 或 STS，不读取浏览器 token。

创建返回 `pending_iam` 只表示空间和 IAM 任务已创建，不表示 IAM 同步或索引激活完成。Framework 不代替 Core 授权、迁移、策略/Profile 创建或事务回滚。

### 文档配置快照边界（2026-10-06）

正式 Host 文档接口仍只接收 title/uri/content/content_type/checksum/version/tags。Go Host 客户端遇到非 nil `KnowledgeDocument.Ingestion` 时，在请求和凭证获取之前返回 `KNOWLEDGE_UNSUPPORTED_CAPABILITY`，不发送旧的顶层 camelCase 配置字段。仅文本提交继续使用正式 DTO；Local 的 `IngestionConfig` 不受影响。.NET 文档 DTO 当前未暴露快照参数，不新增未发布的 Host 字段。

完整向导的接入顺序和 Core 验收清单见 [Knowledge Lab Core 要求](../../../../../docs/contracts/knowledge-lab-core-requirements.md#2026-10-06入库配置快照交付清单)。最终 Core OpenAPI/DTO 交付后，再补充两种 Framework typed 映射、快照回读和插件提交验收。传输成功或额外 JSON 字段被忽略均不构成快照支持。

合同测试覆盖两种服务鉴权、固定路由/请求 envelope、UUID/128-byte 名称/配额校验、可选字段省略、不可用策略、不支持能力 fail-closed、错误状态/机器码/Trace ID。这些使用测试 transport，不等于真实 STS Exchange 或浏览器验收。运行验收仍需 Core migrate/restart、两项显式 grant、必要时租户能力版本升级，再执行真实 catalog → create → 空间目录复读。


### 文档配置快照 v1（2026-10-07）

Core 已发布 `powerx.knowledge.document-ingestion/v1`，本节取代前节的合同缺失状态。Go `KnowledgeDocument.HostIngestion` 使用 `DocumentIngestionSettings`（精确 Profile UUID/整数版本及可选指针字段），正式请求序列化为嵌套 snake_case `ingestion`。Local `IngestionConfig` 不转换为 Host DTO；错误使用 locale key 和机器码。原文不 trim，保留 UTF-8 SHA-256 一致性。

`KnowledgeCatalog.DocumentIngestion` 保留真实支持范围/默认值；`KnowledgeIndexJob` 保留 requested/effective config、snapshot checksum、chunk count、priority、trace 和时间。`DelegatedProvider` 实现显式 `IndexJobChunksReader`，固定读取 `/api/v1/tenant/knowledge/index-jobs/:uuid/chunks`，没有 Local fallback。Local 拒绝 Host 配置。

.NET 对齐 `DocumentIngestionSettings`、任务快照、`IKnowledgeJobChunksService` 和目录合同。非默认 false、0、空数组/顺序由两种 Framework transport 测试覆盖。实际 Plugin Local+proxy 浏览器提交/分块/检索证据见 [验收记录](../../../../../docs/contracts/evidence/knowledge-lab-snapshot-20261007.json)。当前执行 lexical chunks，不等于向量 embedding；没有新增实时事件或自动轮询。

### Host 入库快照与重建（2026-10-07）

Core 正式文档接口已接受嵌套 `ingestion`。Go 使用 `KnowledgeDocument.HostIngestion`（`DocumentIngestionSettings`）提交；Local 的 `IngestionConfig` 使用 Profile key，不能作为 Host UUID/版本配置提交。指针字段保留显式 false、0、空数组和顺序。任务与分块回读保留实际配置、源版本/checksum、分块数、trace 和冻结 Profile 引用。

重建调用 `ReindexInput.HostRebuild`，显式指定 `Mode`、`IdempotencyKey`：

- `reuse_snapshot` 不允许携带 `Ingestion`，由 Core 使用上次成功任务的冻结配置。
- `apply_config` 必须携带 typed `Ingestion`，Core 重新校验当前空间策略、Profile UUID/版本和支持范围。
- 有 `DocumentID`：固定 POST `/api/v1/tenant/knowledge/spaces/{space_uuid}/documents/{document_uuid}/indexes:rebuild`。
- 无 `DocumentID`：固定 POST `/api/v1/tenant/knowledge/spaces/{space_uuid}/indexes:rebuild`。

相同逻辑请求重试时保留幂等键，变更配置或明确开始一次新操作时生成新键。SDK 不自动改写配置、不把文档失败改为空间重建、不自行重试冲突。任务详情提供 `rebuild_mode`、`document_count`、`processed_documents` 和逐文档 `document_configs`。

原文冻结、并发去重、旧索引保留及成功后的原子切换由 Core 实现。SDK/页面测试只能证明请求、范围和响应适配；“其他文档不受影响”“失败后旧索引可检索”仍需通过真实 Core 数据验收。

### 严格语义合同（2026-10-08）

`SemanticProvider(provider)` 取得可选 `SemanticKnowledgeProvider`：GetSemanticCapabilities、ConfigureSemanticIndex、QuerySemanticKnowledge、SetDocumentVisibility。Local 未实现时明确不支持；Delegated 凭据固定调用正式 Core capability，不回退到 Search/Local。

请求与响应类型见 `semantic.go`。查询使用完整原文和 typed filters，不应用旧 MinScore、按文档去重、转换概率或计算客户 fit score。保留 configuration/corpus/bundle generation，pending 模型必须由明确空间重建发布。`SourceText` 对不可变全文按 Unicode codepoint 校验来源位置。

文档与重建追加 Indexing、Artifacts、ExternalRef、IdempotencyKey，job 回读追加实际 artifact/vector 数量、模型快照与逐文档配置。真实 SDK 证据与页面说明见 `docs/contracts/knowledge-semantic-framework.md`。
