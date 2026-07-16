# Framework Knowledge Runtime

`runtime/knowledge` provides provider-neutral contracts for plugin knowledge search, document indexing, Agent/Skill retrieval, diagnostics, and test fixtures.

The framework owns the generic runtime surface only:

- local provider for standalone development and repeatable tests
- delegated provider adapter contract for PowerX-hosted or proxy mode
- mock provider for deterministic plugin tests
- tenant, citation, redaction, and stable error handling

Production knowledge authority remains PowerX Core or a configured delegated provider. Local and mock providers are blocked in production unless a break-glass policy is explicitly enabled and auditable.
