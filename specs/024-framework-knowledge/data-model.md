# Data Model: Framework Knowledge Base

## KnowledgeProvider

Runtime provider abstraction.

- `name`: provider identifier, e.g. `local`, `powerx_delegated`, `mock`.
- `mode`: `local` | `delegated` | `mock` | `third_party`.
- `capabilities`: supported operations and limits.
- `health`: ready/degraded/unavailable with diagnostic reason.

Rules:

- Provider selection is explicit.
- Production local/mock requires break-glass.
- Delegated provider unavailable must not fallback silently.

## KnowledgeSpace

Logical searchable namespace.

- `space_id`: stable provider-neutral identifier.
- `plugin_id`: owning plugin.
- `tenant_uuid`: optional for global spaces, required for tenant spaces.
- `visibility`: `private` | `tenant` | `plugin` | `public`.
- `locale`: optional default locale.
- `agent_ids`: optional agent bindings.
- `skill_ids`: optional skill bindings.
- `metadata`: provider-neutral metadata.

Rules:

- Tenant spaces require tenant context before search/index.
- Visibility must be enforced before provider call where possible.

## KnowledgeDocument

Source document submitted for indexing.

- `document_id`: stable document id.
- `space_id`: target knowledge space.
- `title`: display title.
- `uri`: source URI or logical path.
- `content`: text or markdown payload for inline documents.
- `content_type`: `text/plain` | `text/markdown` | `text/html` | `application/json`.
- `checksum`: content checksum for idempotency.
- `version`: document version.
- `tags`: searchable tags.
- `metadata`: provider-neutral metadata.
- `visibility`: optional override.

Rules:

- Content or URI is required.
- Secrets in metadata/content must be redacted in diagnostics.
- Upsert is idempotent by `(space_id, document_id, version/checksum)` where provider supports it.

## KnowledgeChunk

Searchable document fragment.

- `chunk_id`: provider-neutral chunk id.
- `document_id`: source document id.
- `space_id`: source space.
- `text`: chunk text.
- `score`: provider score normalized to 0..1 where possible.
- `position`: optional page/section/offset info.
- `metadata`: chunk metadata.
- `citation`: required citation reference.

Rules:

- Results without text are ignored or returned as invalid provider response.
- Cross-tenant chunks must be rejected before caller receives results.

## KnowledgeCitation

Traceable source reference.

- `document_id`
- `chunk_id`
- `title`
- `uri`
- `version`
- `position`
- `provider`
- `retrieved_at`

Rules:

- Citation must not include secrets or raw signed URLs unless explicitly allowed by caller policy.
- UI and Agent responses should use citation rather than provider-specific fields.

## KnowledgeQuery

Normalized retrieval input.

- `query`: required text.
- `space_ids`: optional knowledge spaces.
- `tenant_uuid`: required for tenant-scoped spaces.
- `plugin_id`
- `agent_id` / `agent_uuid`
- `skill_id`
- `caller_type`: `member` | `customer` | `agent` | `system`.
- `locale`
- `tags`
- `limit`
- `min_score`
- `filters`
- `trace_id`

Rules:

- Empty query is invalid unless operation is retrieve-by-id.
- Tenant and caller scope must be resolved before delegated calls.

## KnowledgeSearchResult

Provider-neutral search output.

- `query_id`
- `provider`
- `space_id`
- `chunks`
- `citations`
- `total`
- `diagnostics`
- `trace_id`

Rules:

- Citations should correspond to returned chunks.
- Provider latency and error category must be available in diagnostics.

## KnowledgeIndexJob

Async indexing/reindexing state.

- `job_id`
- `space_id`
- `document_id`
- `operation`: `upsert` | `delete` | `reindex`
- `status`: `queued` | `running` | `succeeded` | `failed` | `cancelled`
- `error_code`
- `message`
- `started_at`
- `finished_at`

Rules:

- Synchronous providers may return completed job state.
- Failed jobs must expose stable error codes.

## KnowledgeProviderError

Stable error model.

- `code`
- `message`
- `provider`
- `operation`
- `retryable`
- `trace_id`
- `safe_details`

Standard codes:

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
