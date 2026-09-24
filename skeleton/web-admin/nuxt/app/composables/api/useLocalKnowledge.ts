import { useApiClient } from './_client'

export type KnowledgeProfileKind = 'ingestion' | 'index' | 'rag'
export type KnowledgeProfileVersion = { uuid: string; profile_key: string; version: number; status: 'draft' | 'published' | 'archived'; display_name: string; config: Record<string, unknown>; rollback_from_uuid?: string; created_at: string; updated_at: string }
export type LocalKnowledgeSpace = { uuid: string; name: string; department_code: string; ingestion_profile_key: string; index_profile_key: string; rag_profile_key: string; embedding_profile_key: string; active_vector_index_key?: string; feature_flags?: string[]; status: string; created_at: string; updated_at: string }
export type LocalKnowledgeVectorIndex = { uuid: string; index_key: string; table_name: string; dimensions: number; embedding_provider: string; embedding_model: string; embedding_profile_ref: string; status: 'creating' | 'active' | 'retired' | 'failed'; last_error?: string; updated_at: string }
export type LocalKnowledgeVectorIndexStatus = { embedding_profile_key?: string; active_vector_index_key?: string; active?: LocalKnowledgeVectorIndex | null; indexes: LocalKnowledgeVectorIndex[] }
export type KnowledgeStrategyPackage = { key: string; profile_key: string; dependencies: { index: string[]; runtime: string[]; assets: string[] }; bundle_prerequisites: string[]; ready: boolean; missing_requirements: string[] }
export type SaveKnowledgeSpaceInput = { uuid?: string; name: string; department_code: string; strategy_package_key: string }
export type LocalKnowledgeDocument = { uuid: string; space_uuid: string; title: string; content: string; source_type?: 'manual' | 'upload'; tags?: string[]; effective_from?: string; effective_to?: string; ingestion?: LocalKnowledgeIngestionSnapshot; status: string; chunk_count: number; updated_at: string }
export type LocalKnowledgeIngestionSnapshot = { ingestion_profile: string; processor_profile: string; masking_profile?: string; priority: 'normal' | 'high'; rag_scene_key?: string; rag_bundle_key?: string; rag_primary?: string; segment_mode: 'unit' | 'heading' | 'clause' | 'semantic' | 'table_row' | 'code_block' | 'conversation'; chunk_size: number; chunk_overlap: number; segment_size_policy: 'cap' | 'target'; segment_order: string[]; separators: string[]; page_priority: boolean; anchor_heading_path: boolean; anchor_clause_id: boolean; anchor_row_number: boolean; anchor_speaker: boolean; anchor_sentence_index: boolean }
export type LocalKnowledgeChunkListItem = { uuid: string; ordinal: number; kind: string; char_count: number }
export type LocalKnowledgeChunk = { uuid: string; ordinal: number; kind: string; content: string; metadata?: Record<string, unknown> }
export type LocalKnowledgeChunkPage = { items: LocalKnowledgeChunkListItem[]; page: number; page_size: number; total: number }
export type LocalKnowledgeDocumentInspection = { document: LocalKnowledgeDocument; persisted_chunk_count: number; vector_count: number; vector_dimensions?: number; vector_index_key?: string; vector_status: 'verified' | 'mismatch' | 'unavailable'; vector_reason?: string }
export type LocalKnowledgeJob = { uuid: string; document_uuid: string; operation: string; status: string; error_code?: string; created_at: string }
export type LocalKnowledgeIngestionJob = { uuid: string; space_uuid: string; source_id: string; source_type: 'manual' | 'upload'; status: string; priority: string; retry_count: number; chunk_total: number; summary_chunk_count: number; paragraph_chunk_count: number; chunk_covered_pct: number; embedding_success_pct: number; progress_percent: number; error_code?: string; blocked_reason?: string; submitted_by?: string; started_at?: string; completed_at?: string; created_at: string; metrics_snapshot?: { ingestion?: LocalKnowledgeIngestionSnapshot; index_job_uuid?: string; source_title?: string } }
export type LocalKnowledgeIngestionJobPage = { items: LocalKnowledgeIngestionJob[]; page: number; page_size: number; total: number }
export type LocalKnowledgeJobChunk = { uuid: string; ordinal: number; kind: string; content: string; metadata?: Record<string, unknown> }
export type LocalKnowledgeJobChunkPage = { items: LocalKnowledgeJobChunk[]; page: number; page_size: number; total: number }
export type LocalKnowledgeSpaceSource = { uuid: string; connector_uuid: string; provider: 'notion' | 'feishu'; credential_uuid: string; credential_name: string; credential_status: string; sync_mode: 'full_then_incremental' | 'incremental'; schedule?: string; status: 'blocked' | 'active' | 'failed'; scope: Record<string, unknown>; last_error?: string; created_at: string; updated_at: string }
export type SaveLocalKnowledgeSpaceSource = { provider: 'notion' | 'feishu'; credential_uuid?: string; auth_type?: 'token' | 'oauth'; credential_name?: string; secret_ref?: string; scope: Record<string, unknown>; sync_mode: 'full_then_incremental' | 'incremental'; schedule?: string }
export type LocalKnowledgeRoute = { uuid: string; router_space_uuid: string; target_space_uuid: string; keywords: string[]; priority: number; enabled: boolean }

function dataOf<T>(out: any): T { return (out?.data ?? out) as T }
export function useLocalKnowledgeApi() {
  const { client } = useApiClient()
  return {
    async profileVersions(kind: KnowledgeProfileKind) { return dataOf<{ items: KnowledgeProfileVersion[] }>(await client(`admin/local-knowledge/profile-versions/${kind}`)) },
    async createProfileVersion(kind: KnowledgeProfileKind, value: Pick<KnowledgeProfileVersion, 'profile_key' | 'display_name' | 'config'>) { return dataOf<KnowledgeProfileVersion>(await client(`admin/local-knowledge/profile-versions/${kind}`, { method: 'POST', body: value })) },
    async publishProfileVersion(kind: KnowledgeProfileKind, uuid: string) { await client(`admin/local-knowledge/profile-versions/${kind}/${uuid}/publish`, { method: 'POST' }) },
    async rollbackProfileVersion(kind: KnowledgeProfileKind, uuid: string) { return dataOf<KnowledgeProfileVersion>(await client(`admin/local-knowledge/profile-versions/${kind}/${uuid}/rollback`, { method: 'POST' })) },
    async spaces() { return dataOf<{ items: LocalKnowledgeSpace[] }>(await client('admin/local-knowledge/spaces')) },
    async strategyPackages() { return dataOf<{ items: KnowledgeStrategyPackage[] }>(await client('admin/local-knowledge/strategy-packages')) },
    async routes() { return dataOf<{ items: LocalKnowledgeRoute[] }>(await client('admin/local-knowledge/routes')) },
    async saveRoute(value: Omit<LocalKnowledgeRoute, 'uuid' | 'enabled'>) { return dataOf<LocalKnowledgeRoute>(await client('admin/local-knowledge/routes', { method: 'POST', body: value })) },
    async deleteRoute(uuid: string) { await client(`admin/local-knowledge/routes/${uuid}`, { method: 'DELETE' }) },
    async saveSpace(value: SaveKnowledgeSpaceInput) { const method = value.uuid ? 'PUT' : 'POST'; const path = value.uuid ? `admin/local-knowledge/spaces/${value.uuid}` : 'admin/local-knowledge/spaces'; return dataOf<LocalKnowledgeSpace>(await client(path, { method, body: value })) },
    async deleteSpace(uuid: string) { await client(`admin/local-knowledge/spaces/${uuid}`, { method: 'DELETE' }) },
    async vectorIndexStatus(spaceUUID: string) { return dataOf<LocalKnowledgeVectorIndexStatus>(await client(`admin/local-knowledge/spaces/${spaceUUID}/vector-index`)) },
    async activateVectorIndex(spaceUUID: string, embeddingProfileKey?: string) { return dataOf<{ active: LocalKnowledgeVectorIndex; created_table: boolean }>(await client(`admin/local-knowledge/spaces/${spaceUUID}/vector-index/activate`, { method: 'POST', body: { embedding_profile_key: embeddingProfileKey || '' } })) },
    async documents(spaceUUID: string) { return dataOf<{ items: LocalKnowledgeDocument[] }>(await client(`admin/local-knowledge/spaces/${spaceUUID}/documents`)) },
    async inspectDocument(uuid: string) { return dataOf<LocalKnowledgeDocumentInspection>(await client(`admin/local-knowledge/documents/${uuid}/inspection`)) },
    async documentChunks(uuid: string, page: number, pageSize: number) { return dataOf<LocalKnowledgeChunkPage>(await client(`admin/local-knowledge/documents/${uuid}/chunks?page=${page}&page_size=${pageSize}`)) },
    async documentChunk(uuid: string, chunkUUID: string) { return dataOf<LocalKnowledgeChunk>(await client(`admin/local-knowledge/documents/${uuid}/chunks/${chunkUUID}`)) },
	async updateDocumentChunk(uuid: string, chunkUUID: string, content: string) { return dataOf<LocalKnowledgeChunk>(await client(`admin/local-knowledge/documents/${uuid}/chunks/${chunkUUID}`, { method: 'PATCH', body: { content } })) },
    async sources(spaceUUID: string) { return dataOf<{ items: LocalKnowledgeSpaceSource[] }>(await client(`admin/local-knowledge/spaces/${spaceUUID}/sources`)) },
    async createSource(spaceUUID: string, value: SaveLocalKnowledgeSpaceSource) { return dataOf<LocalKnowledgeSpaceSource>(await client(`admin/local-knowledge/spaces/${spaceUUID}/sources`, { method: 'POST', body: value })) },
    async saveDocument(spaceUUID: string, value: Partial<LocalKnowledgeDocument>) { const method=value.uuid?'PUT':'POST'; const path=value.uuid?`admin/local-knowledge/documents/${value.uuid}`:`admin/local-knowledge/spaces/${spaceUUID}/documents`; return dataOf<LocalKnowledgeDocument>(await client(path,{method,body:value})) },
    async deleteDocument(uuid: string) { await client(`admin/local-knowledge/documents/${uuid}`, { method: 'DELETE' }) },
    async indexDocument(uuid: string) { return dataOf<LocalKnowledgeJob>(await client(`admin/local-knowledge/documents/${uuid}/index`, { method: 'POST' })) },
    async jobs(spaceUUID: string) { return dataOf<{ items: LocalKnowledgeJob[] }>(await client(`admin/local-knowledge/spaces/${spaceUUID}/jobs`)) },
    async ingestionJobs(spaceUUID: string, page = 1, pageSize = 25, sourceID?: string) { const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) }); if (sourceID) query.set('source_id', sourceID); return dataOf<LocalKnowledgeIngestionJobPage>(await client(`admin/local-knowledge/spaces/${spaceUUID}/ingestion-jobs?${query}`)) },
    async ingestionJob(spaceUUID: string, jobUUID: string) { return dataOf<LocalKnowledgeIngestionJob>(await client(`admin/local-knowledge/spaces/${spaceUUID}/ingestion-jobs/${jobUUID}`)) },
    async ingestionJobChunks(spaceUUID: string, jobUUID: string, page = 1, pageSize = 25) { return dataOf<LocalKnowledgeJobChunkPage>(await client(`admin/local-knowledge/spaces/${spaceUUID}/ingestion-jobs/${jobUUID}/chunks?page=${page}&page_size=${pageSize}`)) }
  }
}
