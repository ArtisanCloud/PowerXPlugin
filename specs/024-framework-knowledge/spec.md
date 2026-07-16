# Feature Specification: Framework Knowledge Base

**Feature Branch**: `024-framework-knowledge`  
**Created**: 2026-06-30  
**Status**: Draft  
**Input**: User description: "基于 docs/plan/024-framework-knowledge.md，为 PowerXPlugin Framework 定义智能体知识库封装，支持 local 与 PowerX delegated 调用，并服务 Agent/Skill 的知识检索场景。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 统一检索插件知识 (Priority: P1)

插件开发者需要通过一套统一的知识检索能力读取插件知识、租户知识、运营文档或帮助文档，而不用关心这些知识当前来自本地开发环境还是 PowerX 平台托管服务。

**Why this priority**: 统一检索是知识库封装的最小可用价值。没有这一层，插件会继续直接接不同知识源，导致本地模式、宿主模式和生产模式行为不一致。

**Independent Test**: 使用同一条知识检索请求，在本地知识源和 PowerX delegated 知识源下都能得到相同结构的结果，并包含可追踪来源。

**Acceptance Scenarios**:

1. **Given** 插件运行在本地开发模式且已准备测试文档，**When** 开发者检索一个关键词，**Then** 系统返回匹配片段、来源引用和检索诊断。
2. **Given** 插件运行在 PowerX 托管或代理模式，**When** 开发者发起同样的检索，**Then** 系统委托平台知识服务并返回相同结构的结果。
3. **Given** 当前知识源不支持某项操作，**When** 插件请求该操作，**Then** 系统返回明确的“不支持”结果，而不是静默忽略或换用其他知识源。

---

### User Story 2 - 智能体获取可引用知识上下文 (Priority: P1)

智能体或 Skill 在执行任务前需要获取和用户问题相关的知识片段，并保留这些片段来自哪个文档、哪个版本、哪个位置，便于生成回答、排障和审计。

**Why this priority**: Agent/Skill 是知识库的核心消费方。知识结果如果没有统一上下文和引用，会导致回答不可追踪，也无法判断是否发生跨租户或越权检索。

**Independent Test**: 构造一个带租户、插件和智能体上下文的问题，系统能返回按相关性排序的知识片段和引用；缺少必要上下文时拒绝检索。

**Acceptance Scenarios**:

1. **Given** 智能体任务携带有效租户和问题，**When** 它请求知识上下文，**Then** 系统返回相关片段、引用和 trace 信息。
2. **Given** 请求缺少租户上下文但目标知识空间是租户级，**When** 智能体请求知识上下文，**Then** 系统拒绝请求并说明需要租户上下文。
3. **Given** 知识结果包含敏感凭证或不可公开字段，**When** 系统返回给智能体，**Then** 结果被脱敏或拒绝，并保留安全诊断信息。

---

### User Story 3 - 管理知识文档生命周期 (Priority: P2)

插件开发者需要把 README、FAQ、产品文档、运营 playbook 或业务说明同步进知识库，并能更新、删除或重新索引这些文档，保证检索结果反映最新内容。

**Why this priority**: 只有检索能力不足以支撑真实知识库。文档进入、更新、撤回和重建索引必须有一致语义，否则不同插件会产生不同的文档管理方式。

**Independent Test**: 提交一份文档后可以被检索；更新文档后检索结果反映新版本；删除文档后不再返回旧片段。

**Acceptance Scenarios**:

1. **Given** 开发者提交一份带标题、内容和标签的文档，**When** 文档同步完成，**Then** 后续检索能返回该文档的片段和引用。
2. **Given** 文档内容更新，**When** 开发者重新提交同一文档，**Then** 系统记录新版本并使检索结果使用新内容。
3. **Given** 文档被删除或撤回，**When** 用户再次检索相关关键词，**Then** 系统不再返回该文档的片段。

---

### User Story 4 - 控制本地与平台知识源边界 (Priority: P2)

平台治理负责人需要明确本地知识源、测试知识源和 PowerX delegated 知识源的使用边界，避免生产环境绕过平台权限、审计和租户隔离。

**Why this priority**: 知识库可能包含租户资料、客户资料、运营资料或内部文档。错误的知识源选择会造成数据泄露和审计缺口。

**Independent Test**: 生产环境默认拒绝使用本地或测试知识源；平台知识源不可用时返回明确错误；显式应急模式会产生可审计诊断。

**Acceptance Scenarios**:

1. **Given** 生产环境配置为本地知识源，**When** 系统启动或执行知识操作，**Then** 系统拒绝并提示该配置不允许用于生产。
2. **Given** 平台 delegated 知识源不可用，**When** 插件执行知识检索，**Then** 系统返回明确的上游不可用错误，不自动回退到本地知识源。
3. **Given** 管理员显式开启受控应急模式，**When** 系统使用本地知识源，**Then** 所有相关操作都带有可审计标记。

---

### User Story 5 - 使用标准测试知识源 (Priority: P3)

插件测试编写者需要用标准测试知识源覆盖检索成功、无结果、越权、知识源不可用和文档更新等场景，而不依赖真实 PowerX 平台或外部检索服务。

**Why this priority**: 知识库是横切能力。标准测试工具能让插件测试稳定、快速，并避免每个插件重复定义不同的知识结果格式。

**Independent Test**: 使用测试知识源加载 fixture 文档，并在单元或集成测试中稳定返回预期片段、引用和错误。

**Acceptance Scenarios**:

1. **Given** 测试准备了 fixture 文档，**When** 测试执行检索，**Then** 系统返回确定的片段和引用。
2. **Given** 测试模拟知识源不可用，**When** 插件执行检索，**Then** 系统返回稳定错误而不是 panic 或空结果。
3. **Given** 新插件由模板生成，**When** 开发者查看测试样例，**Then** 可以看到统一知识源 fixture 和断言方式。

### Edge Cases

- 请求没有租户上下文，但目标知识空间属于租户级范围。
- 请求携带的租户与知识源返回的租户不一致。
- 知识源支持检索但不支持文档写入、删除或重新索引。
- 文档刚更新但索引仍是旧版本。
- 文档过大、内容为空、格式不支持或元数据缺失。
- 检索结果为空、重复、没有分数、没有引用或引用不完整。
- 平台知识源返回未授权、禁止访问、限流、超时或不可解析响应。
- 知识片段、引用或错误详情中包含密钥、令牌、手机号、邮箱等敏感信息。
- 智能体请求 customer 侧知识，但当前知识空间只允许后台成员访问。
- 生产环境意外配置本地或测试知识源。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide one unified knowledge access surface for searching knowledge regardless of whether the active source is local, delegated to PowerX, or a test source.
- **FR-002**: System MUST return search results in a consistent structure containing matched snippets, source citations, relevance information, provider/source identity, and diagnostic trace data.
- **FR-003**: System MUST preserve citations for every returned knowledge snippet so users and operators can trace the source document, version, and location.
- **FR-004**: System MUST allow callers to inspect which knowledge operations are supported by the active knowledge source before using them.
- **FR-005**: System MUST clearly report unsupported operations instead of silently ignoring the request or falling back to another source.
- **FR-006**: System MUST support document submission, update, deletion, and reindex status where the active knowledge source declares those operations available.
- **FR-007**: System MUST prevent tenant-scoped knowledge access when tenant context is missing.
- **FR-008**: System MUST reject knowledge results or operations when tenant scope conflicts are detected.
- **FR-009**: System MUST include caller context such as tenant, plugin, intelligent agent, skill, locale, visibility, and caller type when evaluating knowledge access.
- **FR-010**: System MUST support local knowledge sources for standalone development and repeatable tests.
- **FR-011**: System MUST support delegated PowerX knowledge sources for hosted, proxy, and production usage through an injected delegated client contract that is compatible with the PowerX gateway and STS short-lived credential pattern; framework code MUST NOT store long-lived tokens or depend on host internals.
- **FR-012**: System MUST NOT silently fall back from delegated knowledge access to local knowledge access after delegated failure.
- **FR-013**: System MUST reject local or test knowledge sources in production by default unless an explicit audited exception is configured.
- **FR-014**: System MUST provide a safe knowledge retrieval path for intelligent agents and skills that returns snippets, citations, and diagnostics without exposing source-specific response shapes.
- **FR-015**: System MUST redact secrets, raw credentials, and sensitive operational details from logs, diagnostics, citations, metadata, and user-visible errors.
- **FR-016**: System MUST expose stable error outcomes for source unavailable, unauthorized, forbidden, not found, rate limited, unsupported operation, tenant required, tenant mismatch, invalid document, index failed, and redaction required.
- **FR-017**: System MUST provide a standard test knowledge source that can simulate successful retrieval, empty retrieval, unsupported operations, access denial, and source unavailability.
- **FR-018**: System MUST keep knowledge framework entities generic and MUST NOT define industry-specific models such as course, patient record, order, membership benefit, training plan, support ticket, or customer profile.
- **FR-019**: System MUST document the boundary between framework knowledge access, PowerX platform knowledge authority, plugin business content, and operations support links.
- **FR-020**: System MUST keep generated plugin templates aligned with the same knowledge source configuration, testing expectations, and optional admin debug endpoints/pages for provider inspection and search diagnostics.

### Key Entities *(include if feature involves data)*

- **Knowledge Source**: The active location or service that supplies searchable knowledge. It may be local for development, delegated to PowerX for hosted/production use, or mocked for tests.
- **Knowledge Space**: A logical namespace for searchable content, scoped by tenant, plugin, visibility, and optionally intelligent agent or skill usage.
- **Knowledge Document**: A source item submitted to the knowledge system, such as README, FAQ, help page, playbook, or product documentation.
- **Knowledge Snippet**: A matched fragment returned by search, carrying text, relevance information, and a citation.
- **Knowledge Citation**: A traceable reference that identifies the source document, version, location, and retrieval source for a snippet.
- **Knowledge Query**: A retrieval request containing user intent, scope, filters, locale, caller context, and trace information.
- **Index Job**: A tracked document indexing or reindexing operation with status and failure reason.
- **Knowledge Error**: A stable, safe error result that callers can handle without parsing source-specific failures.

### Assumptions

- PowerX delegated knowledge service is the production authority when the plugin runs under PowerX host or proxy mode.
- Delegated PowerX calls are performed by an injected adapter that follows the existing plugin gateway/STS pattern; the framework contract does not embed PowerX host internals or long-lived credentials.
- Local knowledge sources are intended for development, standalone demos, and automated tests, not as production authority.
- First delivery may use simple text matching for local knowledge; semantic vector search can be added behind the same source contract later.
- Intelligent agents consume retrieved knowledge snippets and citations, but final answer generation remains outside this feature.
- Operations support playbook links may become knowledge document sources, but they are not themselves the framework knowledge authority.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can switch between local and delegated knowledge sources without changing plugin business logic.
- **SC-002**: 100% of tenant-scoped knowledge requests without tenant context are rejected before any knowledge result is returned.
- **SC-003**: 100% of returned knowledge snippets include a citation or are rejected as invalid results.
- **SC-004**: 100% of production configurations using local or test knowledge sources are rejected unless an explicit audited exception is present.
- **SC-005**: Standard test knowledge source covers at least five outcomes: success, empty result, unsupported operation, access denied, and source unavailable.
- **SC-006**: Intelligent agent retrieval returns relevant snippets and citations in a single standard result shape for all supported knowledge sources.
- **SC-007**: No framework knowledge entity introduces industry-specific business models.
- **SC-008**: Documentation enables a plugin developer to identify when to use local, delegated, or test knowledge source without reading implementation code.
- **SC-009**: Generated plugin templates expose the same knowledge provider config, backend debug endpoints, and web-admin Knowledge Lab entry as the skeleton unless explicitly disabled by product packaging policy and recorded in `plugin.yaml` menu configuration or release notes.
