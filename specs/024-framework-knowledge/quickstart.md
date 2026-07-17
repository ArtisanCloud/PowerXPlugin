# Quickstart: Framework Knowledge Base

## 1. Review Existing Knowledge Touchpoints

```bash
rg "knowledge|knowledge_base|知识库|rag|embedding|vector" framework skeleton docs specs \
  -g '!**/node_modules/**' -g '!**/.nuxt/**' -g '!**/.output/**'
```

Expected current baseline:

- Operations support playbook has `knowledge_base` links.
- Framework has no dedicated runtime knowledge package yet.
- Agent/Skill bridge does not consume a framework knowledge retriever yet.

## 2. Target Package

Planned framework package:

```text
framework/backend/go/runtime/knowledge
```

Initial public API should expose:

- `KnowledgeProvider`
- `KnowledgeQuery`
- `KnowledgeSearchResult`
- `KnowledgeDocument`
- `KnowledgeCitation`
- `ProviderCapabilities`
- `RAGRetriever`
- `MockProvider`

## 3. Local Provider Smoke

After implementation:

```bash
cd framework/backend/go
go test ./runtime/knowledge -run 'TestLocalProvider|TestMockProvider|TestRAGRetriever' -count=1
```

Expected:

- Upsert fixture markdown document.
- Search returns at least one chunk.
- Result includes citation.
- Delete removes the document from later search.

## 4. Delegated Provider Contract

After delegated adapter exists:

```bash
cd framework/backend/go
go test ./runtime/knowledge -run 'TestDelegatedProvider' -count=1
```

Expected:

- 401/403/404/429/5xx map to stable framework errors.
- Provider response normalizes to `KnowledgeSearchResult`.
- No delegated failure falls back to local.

## 5. Skeleton Adapter Verification

After skeleton wiring:

```bash
cd skeleton/backend/go-gin
go test ./internal/services/admin/knowledge ./internal/config -count=1
```

Expected:

- Config chooses `local` in standalone/dev.
- Config chooses `delegated` in host/proxy production.
- Production local/mock fails unless break-glass is explicit.

## 6. Template Parity

After scaffold/CLI templates are updated:

```bash
npm test
```

Expected:

- Skeleton, scaffold, and CLI embedded Go-Gin templates expose the same knowledge config keys.
- Generated plugin compiles without requiring external vector DB.

## 7. Manual Agent RAG Probe

After Agent RAG helper exists:

1. Seed local fixture document.
2. Invoke a skill with query text.
3. Confirm retriever returns snippets and citations.
4. Confirm answer generation remains outside framework knowledge package.

## Current Validation

The following checks passed on 2026-06-30:

```bash
go test ./framework/backend/go/runtime/knowledge ./skeleton/backend/go-gin/internal/services/admin/knowledge ./skeleton/backend/go-gin/internal/config -count=1
ruby -e 'require "yaml"; YAML.load_file("specs/024-framework-knowledge/contracts/knowledge.openapi.yaml")'
bash scripts/contracts/validate-framework-knowledge-boundary.sh
```

Local fixture benchmark sample on Apple M1 Max:

```text
BenchmarkLocalProviderSearchFixture-10  100  88273 ns/op
```
