# Tasks: Framework Knowledge Base

**Input**: Design documents from `/specs/024-framework-knowledge/`  
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/knowledge.openapi.yaml`, `quickstart.md`  
**Tests**: This feature requires framework unit/contract tests, skeleton config/adapter tests, template parity checks, and provider policy tests. Browser E2E is not required for MVP CI.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行，前提是任务改动不同文件且不依赖未完成任务。
- **[Story]**: 仅用户故事阶段使用，格式为 `[US1]`、`[US2]` 等。
- 每个任务必须包含明确文件路径。

## Phase 1: Setup

- [x] T001 Confirm current knowledge touchpoints with `rg "knowledge|knowledge_base|知识库|rag|embedding|vector" framework skeleton docs specs`.
- [x] T002 [P] Create package README in `framework/backend/go/runtime/knowledge/README.md`.
- [x] T003 [P] Add package-level documentation to `framework/backend/go/runtime/knowledge/types.go` when the concrete entity types are implemented; do not create placeholder-only package files.
- [x] T004 [P] Align stable knowledge error codes and provider capability fields in `specs/024-framework-knowledge/contracts/knowledge.openapi.yaml`.

---

## Phase 2: Foundational Framework Contracts

**CRITICAL**: Complete this phase before user-story implementation.

- [x] T005 Create core entity types in `framework/backend/go/runtime/knowledge/types.go`.
- [x] T006 [P] Define `KnowledgeProvider`, `ProviderCapabilities`, and operation options in `framework/backend/go/runtime/knowledge/provider.go`.
- [x] T007 [P] Define stable errors and redaction-safe error mapping in `framework/backend/go/runtime/knowledge/errors.go`.
- [x] T008 [P] Define provider source policy and production local/mock guard in `framework/backend/go/runtime/knowledge/source_policy.go`.
- [x] T009 [P] Define diagnostics fields and log helpers in `framework/backend/go/runtime/knowledge/diagnostics.go`.
- [x] T010 [P] Add unit tests for entity validation in `framework/backend/go/runtime/knowledge/types_test.go`.
- [x] T011 [P] Add unit tests for error mapping and redaction in `framework/backend/go/runtime/knowledge/errors_test.go`.

**Checkpoint**: Framework has provider-neutral knowledge contracts without skeleton dependencies.

---

## Phase 3: User Story 1 - 插件统一检索知识库 (Priority: P1)

**Goal**: Same framework search call works against local and delegated providers.

**Independent Test**: Local and delegated mock providers return the same normalized `KnowledgeSearchResult`.

### Tests

- [x] T012 [P] [US1] Add provider contract tests in `framework/backend/go/runtime/knowledge/provider_contract_test.go`.
- [x] T013 [P] [US1] Add local provider search tests in `framework/backend/go/runtime/knowledge/local_provider_test.go`.
- [x] T014 [P] [US1] Add delegated provider normalized search success, tenant-mismatch, citation-required, and timeout tests in `framework/backend/go/runtime/knowledge/delegated_provider_test.go` and `framework/backend/go/runtime/knowledge/redaction_test.go`.

### Implementation

- [x] T015 [US1] Implement local/dev provider in `framework/backend/go/runtime/knowledge/local_provider.go`.
- [x] T016 [US1] Implement provider capability inspection in `framework/backend/go/runtime/knowledge/provider.go`.
- [x] T017 [US1] Implement search result normalization helpers in `framework/backend/go/runtime/knowledge/normalize.go`.

**Checkpoint**: Local provider can upsert fixture documents and search with citations.

---

## Phase 4: User Story 2 - 智能体通过 framework 做 RAG 上下文获取 (Priority: P1)

**Goal**: Agent/Skill code retrieves knowledge snippets through framework helper, not provider-specific APIs.

**Independent Test**: RAG helper accepts runtime context and query, returns snippets/citations and stable errors.

### Tests

- [x] T018 [P] [US2] Add RAG helper success, tenant-required, and caller-context propagation tests in `framework/backend/go/runtime/knowledge/rag_test.go`.
- [x] T019 [P] [US2] Add redaction tests for RAG output in `framework/backend/go/runtime/knowledge/redaction_test.go`.

### Implementation

- [x] T020 [US2] Implement `RAGRetriever` and context input types in `framework/backend/go/runtime/knowledge/rag.go`.
- [x] T021 [US2] Implement result redaction helpers in `framework/backend/go/runtime/knowledge/redaction.go`.
- [x] T022 [US2] Add skeleton Agent/Skill adapter example in `skeleton/backend/go-gin/internal/services/admin/knowledge/rag_adapter.go`.

**Checkpoint**: Agent/Skill can request knowledge snippets without importing provider-specific packages.

---

## Phase 5: User Story 3 - 统一知识库索引与文档同步契约 (Priority: P2)

**Goal**: Framework supports upsert/delete/reindex contracts across providers.

**Independent Test**: Upsert/delete fixture document and verify search visibility changes.

### Tests

- [x] T023 [P] [US3] Add document upsert/delete tests in `framework/backend/go/runtime/knowledge/document_test.go`.
- [x] T024 [P] [US3] Add index job status tests in `framework/backend/go/runtime/knowledge/index_job_test.go`.

### Implementation

- [x] T025 [US3] Implement document validation in `framework/backend/go/runtime/knowledge/document.go`.
- [x] T026 [US3] Implement local provider upsert/delete/reindex behavior in `framework/backend/go/runtime/knowledge/local_provider.go`.
- [x] T027 [US3] Add support playbook source adapter design notes in `docs/guides/develop/knowledge/framework-knowledge.md`.

**Checkpoint**: MVP local provider supports basic document lifecycle.

---

## Phase 6: User Story 4 - provider 模式与生产安全边界 (Priority: P2)

**Goal**: Production defaults to delegated provider and rejects unsafe local/mock modes.

**Independent Test**: production + local fails; break-glass allows with diagnostics; delegated unavailable maps stable error.

### Tests

- [x] T028 [P] [US4] Add source policy tests in `framework/backend/go/runtime/knowledge/source_policy_test.go`.
- [x] T029 [P] [US4] Add delegated provider error mapping tests in `framework/backend/go/runtime/knowledge/delegated_provider_test.go`.
- [x] T030 [P] [US4] Add skeleton config tests in `skeleton/backend/go-gin/internal/config/knowledge_test.go`.

### Implementation

- [x] T031 [US4] Implement delegated provider client adapter skeleton and STS-compatible gateway contract boundary in `framework/backend/go/runtime/knowledge/delegated_provider.go`.
- [x] T032 [US4] Add knowledge config structs and validation in `skeleton/backend/go-gin/internal/config/config.go`.
- [x] T033 [US4] Add skeleton provider factory in `skeleton/backend/go-gin/internal/services/admin/knowledge/provider_factory.go`.

**Checkpoint**: Provider mode is explicit and production unsafe modes fail fast.

---

## Phase 7: User Story 5 - 提供测试工具和模板对齐 (Priority: P3)

**Goal**: Tests and generated plugins can use framework knowledge helpers without external services.

**Independent Test**: Mock provider fixture covers success and provider failure.

### Tests

- [x] T034 [P] [US5] Add mock provider tests covering success, empty result, unsupported operation, access denied, and source unavailable in `framework/backend/go/runtime/knowledge/testing_test.go`.
- [x] T035 [P] [US5] Add skeleton adapter tests using mock provider in `skeleton/backend/go-gin/internal/services/admin/knowledge/provider_factory_test.go`.

### Implementation

- [x] T036 [US5] Implement `MockProvider`, fixture document helpers, and assertion helpers for success, empty result, unsupported operation, access denied, and source unavailable in `framework/backend/go/runtime/knowledge/testing.go`.
- [x] T037 [US5] Mirror skeleton config and provider factory in `scaffold/templates/backend/go-gin/internal/config/config.go.tmpl` and `scaffold/templates/backend/go-gin/internal/services/admin/knowledge/provider_factory.go.tmpl`.
- [x] T038 [US5] Mirror skeleton config and provider factory in `tools/cli/internal/templates/data/backend/go-gin/internal/config/config.go.tmpl` and `tools/cli/internal/templates/data/backend/go-gin/internal/services/admin/knowledge/provider_factory.go.tmpl`.
- [x] T039 [US5] Update framework knowledge docs in `docs/guides/develop/knowledge/framework-knowledge.md`.
- [x] T039A [US5] Add admin runtime knowledge debug endpoints in `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/knowledge_handler.go` and mirror them to scaffold and CLI templates.
- [x] T039B [US5] Add Nuxt Knowledge Lab page, sidebar entry, i18n/menu metadata, and plugin manifest menu entries in skeleton, scaffold templates, and CLI templates.

**Checkpoint**: New generated plugins have a consistent knowledge provider config surface.

---

## Final Phase: Polish & Validation

- [x] T040 [P] Add quickstart verification results to `specs/024-framework-knowledge/quickstart.md`.
- [x] T041 [P] Add static boundary check for no industry-specific knowledge model in framework package in `scripts/contracts/validate-framework-knowledge-boundary.sh`.
- [x] T042 Wire boundary check into `package.json` or existing validation target.
- [x] T043 Add local provider p95 fixture benchmark and delegated timeout tests in `framework/backend/go/runtime/knowledge/local_provider_test.go` and `framework/backend/go/runtime/knowledge/delegated_provider_test.go`.
- [x] T044 Run `go test ./framework/backend/go/runtime/knowledge ./skeleton/backend/go-gin/internal/services/admin/knowledge ./skeleton/backend/go-gin/internal/config -count=1`.
- [x] T045 Run template parity checks for scaffold and CLI embedded templates.
- [x] T046 Review diagnostics to ensure no raw token/secret/content leak is logged.
- [x] T047 Run admin runtime knowledge handler tests and web-admin Nuxt build for Knowledge Lab route coverage.

---

## Dependencies & Execution Order

1. Phase 1 Setup has no dependencies.
2. Phase 2 Foundational Contracts blocks all stories.
3. US1 and US2 are P1 and should be implemented before index lifecycle work.
4. US3 depends on provider/document contracts from US1.
5. US4 depends on provider contracts and should be completed before production rollout.
6. US5 depends on finalized contracts and skeleton config shape.
7. Final Phase depends on all story phases.
