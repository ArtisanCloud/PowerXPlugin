# Framework Knowledge Runtime

`runtime/knowledge` provides provider-neutral contracts for plugin knowledge search, document indexing, Agent/Skill retrieval, diagnostics, and test fixtures.

The framework owns the generic runtime surface only:

- local provider for standalone development and repeatable tests
- delegated provider adapter contract for PowerX-hosted or proxy mode
- mock provider for deterministic plugin tests
- tenant, citation, redaction, and stable error handling

Production knowledge authority remains PowerX Core or a configured delegated provider. Local and mock providers are blocked in production unless a break-glass policy is explicitly enabled and auditable.

插件业务服务应只依赖 `KnowledgeProvider`；Provider 的 local/delegated 选择属于 Framework Runtime Factory，而不是页面或业务 service。历史 QA bridge 与正式 PowerX Host Contract 必须保持独立：前者不得被当作 delegated 生产能力的 fallback。共同规则见 [Framework 双模式业务模块规范](../../../../../docs/guides/develop/framework-dual-mode-business-modules.md)。
