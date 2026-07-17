# Framework Knowledge 使用说明

PowerXPlugin Framework 的 knowledge runtime 提供统一的知识检索、文档同步、引用、诊断和测试封装。它不负责生产知识库权威存储；生产环境默认应委托 PowerX Core 或配置的 delegated provider。

## 边界

- Framework 负责通用 contract：`KnowledgeProvider`、`KnowledgeQuery`、`KnowledgeDocument`、`KnowledgeSearchResult`、`KnowledgeCitation`、稳定错误码、脱敏和 source policy。
- Skeleton 负责装配：读取 `knowledge` 配置，构造 local/mock/delegated provider。
- PowerX Core 负责生产权威知识服务、平台审计、跨插件治理和托管知识库；插件侧 concrete delegated adapter 必须沿用 gateway + STS 短期凭证模式，不能使用长效 token 或直接耦合宿主内部实现。
- Agent/Skill 只消费 framework RAG helper 返回的 snippets、citations、diagnostics，不直接依赖 provider 私有响应。

## 配置

开发默认 local：

```yaml
knowledge:
  mode: local
  require_tenant: true
  delegate_timeout: 3s
```

测试可使用 mock：

```yaml
knowledge:
  mode: mock
  require_tenant: true
```

生产或 PowerX host/proxy 模式应使用 delegated：

```yaml
knowledge:
  mode: delegated
  delegate_endpoint: https://powerx.example.com/api/v1/framework/knowledge
  delegate_timeout: 3s
  require_tenant: true
```

delegated provider 的实际 HTTP/gRPC client 应通过 PowerX gateway adapter 注入，并由 adapter 在请求前完成 STS token exchange。Framework 只定义 `DelegatedClient` contract 和注入边界，不保存长效凭据。

生产环境默认拒绝 `local` 和 `mock`。如确需应急，必须显式开启 break-glass 并填写原因：

```yaml
knowledge:
  mode: local
  break_glass_local: true
  break_glass_reason: emergency local replay for incident PX-000
```

## 后端使用

构造 provider：

```go
factory := knowledge.NewProviderFactory(cfg, delegatedClient, nil)
provider, err := factory.Build()
```

Agent/Skill 检索知识上下文：

```go
retriever := knowledge.NewRAGRetriever(provider)
result, err := retriever.Retrieve(ctx, fwknowledge.RAGContext{
    TenantUUID: tenantUUID,
    PluginID: pluginID,
    AgentUUID: agentUUID,
    SkillID: skillID,
    CallerType: fwknowledge.CallerTypeAgent,
    Visibility: fwknowledge.VisibilityTenant,
}, question)
```

返回结果必须使用 `result.Chunks` 和 `result.Citations`。不要把 provider 原始响应透传给 Agent 或前端。

## 管理端调试

Skeleton 和生成模板提供 Knowledge Lab 调试入口：

- 后端：`GET /admin/runtime/knowledge/provider`
- 后端：`POST /admin/runtime/knowledge/search`
- 前端：`/powerx/knowledge-lab`

这些入口用于开发/运维诊断 provider、source policy、tenant context、citation 和 error mapping。产品打包时可按 manifest 策略隐藏菜单，但 scaffold/CLI 默认保持与 skeleton 一致。

## 测试 fixture

用于手工验证知识空间创建、文档入库和检索的最小样本文档：

- `docs/guides/develop/knowledge/fixtures/after-sales-refund-sop.md`

推荐创建知识空间时选择：

- 知识库类型：SOP / 制度 / 产品说明
- 检索策略：层次索引

上传文档后，优先用以下问题验证章节召回：

- 定制商品可以无理由退款吗？
- 已经发货的订单怎么退款？
- 退款多久到账？
- 客服审核退款时要检查哪些信息？

## 文档同步

local provider 支持基础 upsert/delete/reindex，用于开发、fixture 和文档生命周期测试：

```go
_, err := provider.UpsertDocument(ctx, fwknowledge.KnowledgeDocument{
    SpaceID: "plugin-help",
    DocumentID: "faq",
    Title: "FAQ",
    Content: "常见问题...",
    Visibility: fwknowledge.VisibilityTenant,
    TenantUUID: tenantUUID,
})
```

生产 delegated provider 的实际索引语义由 PowerX Core 决定，但必须返回 framework 标准 `KnowledgeIndexJob`。

## 稳定错误

调用方应基于 framework error code 分支处理：

- `KNOWLEDGE_PROVIDER_UNAVAILABLE`
- `KNOWLEDGE_UNAUTHORIZED`
- `KNOWLEDGE_FORBIDDEN`
- `KNOWLEDGE_NOT_FOUND`
- `KNOWLEDGE_RATE_LIMITED`
- `KNOWLEDGE_UNSUPPORTED_CAPABILITY`
- `KNOWLEDGE_TENANT_REQUIRED`
- `KNOWLEDGE_TENANT_MISMATCH`
- `KNOWLEDGE_INVALID_DOCUMENT`
- `KNOWLEDGE_INDEX_FAILED`
- `KNOWLEDGE_REDACTION_REQUIRED`

## 验证

```bash
go test ./framework/backend/go/runtime/knowledge ./skeleton/backend/go-gin/internal/services/admin/knowledge ./skeleton/backend/go-gin/internal/config -count=1
```

边界检查：

```bash
rg -n "course|patient|order|membership benefit|training plan|support ticket|customer profile" framework/backend/go/runtime/knowledge
```

该检查应无业务模型命中。脱敏测试中出现 `secret`、`token` 等词是允许的。
