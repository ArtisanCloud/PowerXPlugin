# Implementation Plan: Framework Knowledge Base

**Branch**: `024-framework-knowledge` | **Date**: 2026-06-30 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/024-framework-knowledge/spec.md`

## Summary

将智能体知识库能力抽象为 PowerXPlugin Framework 的通用 runtime contract。Framework 负责 provider 选择、统一 search/retrieve/upsert/delete/reindex 契约、local/delegated provider 边界、tenant/plugin/agent/skill 作用域、引用/citation、稳定错误、诊断和测试 helper。Skeleton 保留装配、示例和 runtime debug 入口，scaffold/CLI 模板同步；生产默认委托 PowerX Core，不允许静默 local fallback。

## Technical Context

**Language/Version**: Go 1.24, TypeScript 4.x for docs/template validation and web-admin template compatibility  
**Primary Dependencies**: framework runtime common logging/errors, customer/member tenant context, skeleton Agent/Skill bridge patterns, PowerX delegated gateway + STS-compatible adapter contract  
**Storage**: Framework 不强制新增生产持久化；local/dev provider 可使用 in-memory/file/SQLite；生产知识库由 PowerX Core 或配置的 delegated provider 权威维护  
**Testing**: Go unit/contract tests for provider contracts, local/mock provider tests, delegated timeout/error/tenant-mismatch tests, skeleton adapter tests, citation-required checks, Nuxt build, template parity checks  
**Target Platform**: Plugin standalone backend + PowerX host/proxy runtime + Agent/Skill invocation path  
**Project Type**: backend framework runtime + skeleton adapter/config + docs/templates  
**Performance Goals**: local search p95 < 50ms for a small fixture corpus of up to 100 documents or 1 MiB total text; delegated search timeout configurable and fail-fast; result normalization O(result count)  
**Constraints**: Tenant scope required for tenant knowledge; delegated PowerX calls must be made through an injected adapter that can use STS short-lived credentials; production local/mock disabled by default; no raw token/secret logging; no industry-specific knowledge model in framework  
**Scale/Scope**: MVP covers framework Go runtime contracts, local/mock provider, delegated provider interface, skeleton config adapter, Agent RAG helper, Nuxt admin debug endpoint/page, docs and template alignment

## Constitution Check

### Pre-Design Gate

1. Host Contract First: PASS  
Production knowledge authority is PowerX Core/delegated provider. Framework defines contracts and adapter injection points without coupling to host internals; any concrete PowerX adapter must use gateway/STS short-lived credentials.

2. Tenant Isolation & Zero Trust: PASS  
Tenant-scoped knowledge requires resolved tenant and provider calls carry scope. Missing/mismatched tenant fails before retrieval.

3. Service-Centric Architecture: PASS  
Framework owns provider contracts and helpers; skeleton wires config/adapters and thin admin debug handlers; Agent/Skill consumes service interfaces.

4. Observable & Testable Delivery: PASS  
Stable errors, diagnostics, mock provider, fixture helpers, local/delegated contract tests are required.

5. Event Contracts & TaskBus Readiness: PASS（N/A）  
Index jobs may later emit task events, but MVP does not require new event topics.

6. Minimal Footprint & Versioned Releases: PASS  
No new external service requirement; vector DB is not mandatory in MVP; provider capability inspection prevents hidden assumptions.

### Post-Design Gate

Phase 1 documents completed with no constitution violations. Production local/mock break-glass must remain explicit and auditable.

## Project Structure

### Documentation (this feature)

```text
specs/024-framework-knowledge/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/
│   └── requirements.md
├── contracts/
│   └── knowledge.openapi.yaml
└── tasks.md

docs/plan/
└── 024-framework-knowledge.md
```

### Source Code (planned)

```text
framework/backend/go/runtime/knowledge/
├── types.go             # core entities and DTOs
├── provider.go          # KnowledgeProvider contract and capabilities
├── errors.go            # stable errors and mappings
├── local_provider.go    # local/dev provider MVP
├── delegated_provider.go# PowerX Core delegated adapter
├── rag.go               # Agent/Skill RAG helper
├── diagnostics.go       # structured logging/diagnostics
├── source_policy.go     # production local/mock guard
└── testing.go           # mock provider and fixtures

skeleton/backend/go-gin/internal/
├── config/              # knowledge provider config
├── services/admin/knowledge/  # provider factory/adapters
└── transport/http/...   # admin/runtime knowledge debug routes

skeleton/web-admin/
└── nuxt/app/pages/powerx/knowledge-lab.vue

scaffold/templates/
└── ...                  # mirror skeleton backend config/provider/debug UI wiring

tools/cli/internal/templates/data/
└── ...                  # mirror CLI generated output
```

**Structure Decision**: 新建 `framework/backend/go/runtime/knowledge`，避免把知识库能力混入 Agent、Capability、Operations Support 或 Customer Auth 包。

## Phase 0: Research Output

已生成 [research.md](./research.md)，覆盖：
1. 新建 024 而不是塞进 Agent 或 Operations；
2. Provider 契约优先；
3. delegated provider 是生产权威；
4. local provider 的 MVP 边界；
5. Agent RAG helper 与 provider 低耦合；
6. citation/redaction/diagnostics 治理。

## Phase 1: Design Output

已生成：
1. [data-model.md](./data-model.md)
2. [contracts/knowledge.openapi.yaml](./contracts/knowledge.openapi.yaml)
3. [quickstart.md](./quickstart.md)

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 无 | N/A | N/A |
