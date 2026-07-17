# Research: Framework Knowledge Base

## Decision 1: 新建 024，而不是并入 Agent、Operations 或 Capability

**Decision**: 新建 `024-framework-knowledge`。

**Rationale**: 知识库是横切 runtime 能力，Agent/Skill 是主要消费者但不是唯一消费者；Operations Support 里的 `knowledge_base` 是运营配置，不等价于 runtime retrieval/index contract；Capability 只描述能力暴露，不负责知识检索。

**Alternatives considered**:

- 放入 `021-powerx-agent-skill-bridge`：会把知识库封装绑死在 Agent UI/调试页。
- 放入 Operations Support：会把链接管理误当成检索/索引 runtime。
- 放入 Capability：无法表达 provider、index、citation、RAG helper 的运行时语义。

## Decision 2: Provider contract first

**Decision**: Framework 首先定义 provider-neutral 的 `KnowledgeProvider`、实体、错误和 capability inspection。

**Rationale**: 当前最大风险是 local、PowerX delegated、未来向量库/第三方检索接口形状不同。先定义 provider-neutral contract，才能让插件代码不依赖具体存储。

**Alternatives considered**:

- 直接接 PowerX Core API：host 模式快，但 standalone/dev 和测试无法稳定运行。
- 直接做本地向量库：会引入非必要依赖，并偏离生产委托模型。
- 只提供 HTTP client：Agent/Skill 和后端 service 仍需手写业务封装。

## Decision 3: 生产默认 delegated，local/mock 仅 dev/test/break-glass

**Decision**: host/proxy/production 默认使用 PowerX Core delegated provider；local/mock 默认只允许 dev/test，production 使用需显式 break-glass 并审计。

**Rationale**: 知识库内容可能跨租户和包含敏感资料。生产绕过平台 provider 会绕过权限、索引治理、审计和数据生命周期。

**Alternatives considered**:

- 允许生产 local：部署简单，但形成第二知识权威源。
- delegated 不可用时 fallback local：看似高可用，实际会造成数据不一致和越权风险。
- 完全禁止 local：不利于插件 standalone/dev 和单元测试。

## Decision 4: MVP local provider 不强制向量库

**Decision**: local provider MVP 可使用 in-memory/file/SQLite + 简单全文/metadata 匹配；不要求 embedding 或向量 DB。

**Rationale**: 024 的目标是统一 framework contract 和 provider 边界，不是一次性实现搜索质量最优。向量库会引入额外部署、模型、索引和依赖复杂度。

**Alternatives considered**:

- 首版集成向量库：成本高，CI/开发环境不稳定。
- 完全没有 local provider：无法做 standalone 和 contract test。
- local provider 只 mock：不能验证 upsert/search/delete 的基本行为。

## Decision 5: Agent RAG helper 只输出片段和引用，不负责生成答案

**Decision**: Framework RAG helper 负责检索、过滤、redaction、citation 和 trace；不负责 LLM prompt 策略或最终答案生成。

**Rationale**: 生成策略属于 Agent Runtime/Skill 语义。Framework 应提供可治理的知识上下文，而不是把业务回答逻辑写进基础层。

**Alternatives considered**:

- Framework 直接生成回答：会侵入 Agent 责任边界。
- Skill 自己调用 provider：会重复实现权限、引用、redaction 和错误映射。
- 只返回 raw provider response：会把 provider shape 泄露给业务。

## Decision 6: Citation 是强制治理字段

**Decision**: Search result 必须包含可追踪 citation/source 信息；缺失时返回 validation error 或标记为不可引用来源。

**Rationale**: Agent 回答、管理员排障和审计都需要知道答案来自哪个文档/片段。没有 citation 的知识结果难以治理。

**Alternatives considered**:

- citation 可选：实现轻松但审计不可用。
- 只记录 provider trace id：不能定位具体文档。
- 只保留 document title：不足以做 chunk/版本级追踪。
