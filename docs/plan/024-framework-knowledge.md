# PowerXPlugin Framework Knowledge Base 统一规划

本文定义 PowerXPlugin framework 中面向插件、智能体和运行时检索场景的通用知识库封装。该能力解决插件如何以一致方式接入本地知识库、PowerX Core delegated 知识库和测试 mock provider，并为 Agent/Skill 提供可治理的 RAG 上下文获取机制。

## 1. 背景

插件和智能体会持续需要读取知识：

1. 插件 README、FAQ、产品文档、运营 playbook。
2. 租户级知识库、客服知识库、帮助中心。
3. Agent/Skill 执行任务前需要的检索片段。
4. Web Admin 或调试工具需要展示引用、trace 和检索诊断。

当前仓库里已经存在一些知识相关概念，例如 Operations Support Playbook 的 `knowledge_base` 链接，但它本质是运营配置，不是 framework 级 runtime retrieval/index contract。

如果每个插件自行接向量库、对象存储、PowerX Core API 或本地 SQLite，最终会出现：

1. standalone 与 host/proxy 行为不一致。
2. Agent/Skill 重复实现检索、过滤、引用和错误映射。
3. tenant、plugin、agent、skill 作用域散落在业务代码中。
4. citation、trace、redaction 和审计不可统一。
5. production delegated provider 不可用时错误地 fallback 到 local。

因此需要把知识库访问上提为 framework runtime 能力。

## 2. 目标

1. 在 framework 中提供统一 `KnowledgeProvider` 契约。
2. 支持 local/dev provider、PowerX delegated provider 和 mock provider。
3. 提供统一 search、retrieve、upsert document、delete document、reindex、health/capability inspection。
4. 为 Agent/Skill 提供 RAG helper，只返回 snippets、citations、diagnostics，不负责生成最终答案。
5. 保证 tenant/plugin/agent/skill/caller 作用域在 provider 调用前被解析和校验。
6. 统一 citation/source 输出，确保知识结果可追踪。
7. 统一 provider 错误映射和 redaction，避免泄露 token、secret、raw credential。
8. skeleton、scaffold、CLI 模板保持知识库配置与 adapter 一致。

## 3. 非目标

1. 不在 framework 里实现行业知识模型。
2. 不定义课程、患者记录、订单、会员权益、训练计划、客服工单等业务实体。
3. 不强制首版引入向量数据库、embedding 模型或外部搜索引擎。
4. 不把 Operations Support Playbook 的 `knowledge_base` 链接直接当作 runtime 知识库权威源。
5. 不在 framework 内生成 Agent 最终回答；答案生成属于 Agent Runtime 或 Skill 业务逻辑。
6. 不允许 production delegated provider 失败后静默 fallback 到 local provider。

## 4. Provider 模式

Framework 首版 provider 模式：

| Mode | 用途 | 生产默认允许 | 说明 |
| --- | --- | --- | --- |
| `local` | standalone/dev/local smoke | 否 | 可用 in-memory/file/SQLite + 简单全文检索，不要求向量库 |
| `delegated` | host/proxy/production | 是 | 委托 PowerX Core 或平台知识库能力 |
| `mock` | 单元测试/合同测试 | 否 | 用 fixture 构造 search/upsert/delete/reindex 场景 |
| `third_party` | 后续扩展 | 受配置控制 | 预留给外部搜索或向量服务 adapter |

生产约束：

1. `production + local/mock` 默认启动失败。
2. break-glass 必须显式配置，并输出审计/诊断标记。
3. delegated 不可用时返回稳定错误，不回退 local。
4. provider capability 必须可查询，调用方不能猜测 provider 支持哪些操作。

## 5. Framework 包边界

建议新增 framework Go 包：

```text
github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge
```

该包负责：

1. 通用类型定义。
2. provider interface 与 capability inspection。
3. stable errors 与 HTTP/status mapping。
4. local provider MVP。
5. delegated provider adapter contract。
6. Agent/Skill RAG helper。
7. source policy 与 production guard。
8. diagnostics/log fields。
9. mock provider 与测试 fixture。

不负责：

1. 业务文档编辑 UI。
2. 行业实体建模。
3. LLM prompt 生成策略。
4. PowerX Core 内部知识库实现。

## 6. 核心类型

### 6.1 KnowledgeProvider

```go
type KnowledgeProvider interface {
    Capabilities(ctx context.Context) (ProviderCapabilities, error)
    Health(ctx context.Context) (ProviderHealth, error)
    Search(ctx context.Context, query KnowledgeQuery) (KnowledgeSearchResult, error)
    Retrieve(ctx context.Context, ref KnowledgeReference) (KnowledgeChunk, error)
    UpsertDocument(ctx context.Context, doc KnowledgeDocument) (KnowledgeIndexJob, error)
    DeleteDocument(ctx context.Context, ref KnowledgeDocumentRef) (KnowledgeIndexJob, error)
    Reindex(ctx context.Context, input ReindexInput) (KnowledgeIndexJob, error)
}
```

首版可以按 provider capabilities 部分实现；不支持的操作返回 `KNOWLEDGE_UNSUPPORTED_CAPABILITY`。

### 6.2 KnowledgeQuery

```go
type KnowledgeQuery struct {
    Query      string
    SpaceIDs   []string
    TenantUUID string
    PluginID   string
    AgentUUID  string
    SkillID    string
    CallerType string // member, customer, agent, system
    Locale     string
    Tags       []string
    Limit      int
    MinScore   float64
    Filters    map[string]any
    TraceID    string
}
```

### 6.3 KnowledgeSearchResult

```go
type KnowledgeSearchResult struct {
    QueryID     string
    Provider    string
    SpaceID     string
    Chunks      []KnowledgeChunk
    Citations   []KnowledgeCitation
    Total       int
    Diagnostics map[string]any
    TraceID     string
}
```

### 6.4 KnowledgeDocument

```go
type KnowledgeDocument struct {
    DocumentID  string
    SpaceID     string
    Title       string
    URI         string
    Content     string
    ContentType string
    Checksum    string
    Version     string
    Tags        []string
    Visibility  string
    Metadata    map[string]any
}
```

### 6.5 KnowledgeCitation

```go
type KnowledgeCitation struct {
    DocumentID  string
    ChunkID     string
    Title       string
    URI         string
    Version     string
    Position    map[string]any
    Provider    string
    RetrievedAt time.Time
}
```

Citation 是强制治理字段。没有 citation 的 provider 响应必须被标记为不可引用来源或映射为稳定错误。

## 7. Agent / Skill RAG 边界

Agent/Skill 不直接调用 provider。推荐链路：

```text
Agent Invocation / Skill Prepare
        ↓
knowledge.RAGRetriever
        ↓
KnowledgeProvider.Search
        ↓
snippets + citations + diagnostics
        ↓
Agent Runtime / Skill executor 生成最终 response
```

Framework RAG helper 负责：

1. 组装 tenant/plugin/agent/skill/caller context。
2. 校验 scope 和 provider capability。
3. 调用 provider search。
4. 做结果裁剪、redaction、citation 保留。
5. 输出 trace 和 diagnostics。

Framework RAG helper 不负责：

1. 决定最终回答语气。
2. 编排多 Agent 任务。
3. 执行业务 capability。
4. 生成 prompt 模板。

## 8. 与既有 feature 的关系

依赖：

1. `021-powerx-agent-skill-bridge`：Agent/Skill 是 knowledge RAG 的主要消费方。
2. `022-framework-realtime-transport`：后续知识检索或索引进度可通过 realtime transport 暴露，但 MVP 不强制。
3. `023-framework-customer-auth`：C 端 customer 场景可作为 caller context，但不混入知识实体。
4. `018-framework-iam-unification`：后台 member 场景可作为 caller context 和权限来源。

不归入：

1. Operations Support：support playbook 可作为 knowledge source，但不是 runtime provider contract。
2. Capability：knowledge provider 可以被 capability 消费，但不是 capability manifest 本身。
3. Agent Bridge：Agent 使用 knowledge，但 knowledge 不应绑定某个 Agent UI。

## 9. 配置建议

```yaml
knowledge:
  provider: delegated # local | delegated | mock
  allow_local_in_production: false
  delegated:
    base_url: ""
    auth_scheme: sts
    timeout_ms: 3000
  local:
    storage: memory # memory | file | sqlite
    path: tmp/knowledge
  retrieval:
    default_limit: 8
    max_limit: 50
    min_score: 0.0
```

运行时环境覆盖：

```text
POWERX_KNOWLEDGE_PROVIDER
POWERX_KNOWLEDGE_DELEGATED_BASE_URL
POWERX_KNOWLEDGE_ALLOW_LOCAL_IN_PRODUCTION
```

## 10. 错误码

建议稳定错误码：

```text
KNOWLEDGE_PROVIDER_UNAVAILABLE
KNOWLEDGE_UNAUTHORIZED
KNOWLEDGE_FORBIDDEN
KNOWLEDGE_NOT_FOUND
KNOWLEDGE_RATE_LIMITED
KNOWLEDGE_UNSUPPORTED_CAPABILITY
KNOWLEDGE_TENANT_REQUIRED
KNOWLEDGE_TENANT_MISMATCH
KNOWLEDGE_INVALID_DOCUMENT
KNOWLEDGE_INDEX_FAILED
KNOWLEDGE_REDACTION_REQUIRED
```

## 11. 落地阶段

### Phase 1: Framework Contract

1. 新增 `framework/backend/go/runtime/knowledge`。
2. 定义 core types、provider interface、capabilities、errors、diagnostics。
3. 增加 unit tests。

### Phase 2: Local / Mock Provider

1. 实现 local provider MVP。
2. 实现 mock provider 与 fixture helper。
3. 覆盖 search/upsert/delete/reindex 基础测试。

### Phase 3: Delegated Provider

1. 定义 PowerX Core delegated adapter。
2. 实现错误映射和 timeout/fail-fast。
3. 禁止 delegated 失败后 fallback local。

### Phase 4: Agent RAG Helper

1. 实现 `RAGRetriever`。
2. 接入 Agent/Skill invocation context。
3. 输出 snippets/citations/diagnostics。

### Phase 5: Skeleton / Template Alignment

1. skeleton 增加 knowledge config 和 provider factory。
2. scaffold 模板同步。
3. CLI embedded templates 同步。
4. 文档和 quickstart 更新。

## 12. 验证策略

CI 默认不跑复杂浏览器 E2E。本 feature 的默认验证应以稳定后端测试为主：

```bash
go test ./framework/backend/go/runtime/knowledge -count=1
go test ./skeleton/backend/go-gin/internal/services/knowledge ./skeleton/backend/go-gin/internal/config -count=1
npm test
```

复杂 Agent UI / browser E2E 只作为显式 opt-in：

```bash
REGRESSION_RUN_E2E=1 make test-regression
```

## 13. 后续 spec

详细 feature spec 位于：

```text
specs/024-framework-knowledge/
```

该目录负责拆分 user stories、data model、OpenAPI contract、quickstart 和 tasks。
