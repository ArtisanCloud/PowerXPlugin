import type {SemanticCapabilities,SemanticIndexSnapshot} from './knowledgeSemanticTypes'
import { reactive, ref } from 'vue'
import { documentIngestionSchema, type HostIngestionCapabilities } from './knowledgeHostTypes'
import { useApiClient } from './_client'
import type { KnowledgeWorkspaceContext } from './useKnowledgeWorkspace'
import type { KnowledgeStrategyPackage, LocalKnowledgeSpace } from './useLocalKnowledge'

const base = '/admin/runtime/knowledge-lab'
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
function invalid(): never { throw { data: { code: 'KNOWLEDGE_INVALID_RESPONSE' } } }
function data(out: any) { return out?.data ?? out }
function unsupported(): never { throw { data: { code: 'KNOWLEDGE_UNSUPPORTED_CAPABILITY', error: 'knowledgeLab.workspace.unsupported' } } }
function snapshotValid(value:any) {
 if(value?.schema!==documentIngestionSchema || !['lexical_chunks','semantic','hybrid'].includes(value.index_mode) || value.length_unit!=='unicode_rune')return false
 if(value.index_mode==='lexical_chunks')return !value.indexing
 const index=value.indexing as SemanticIndexSnapshot | undefined
 return Boolean(index && index.mode===value.index_mode && uuidPattern.test(index.embedding_profile?.uuid) && Number.isInteger(index.embedding_profile?.version) && index.embedding_profile.version>0 && index.dimensions>0 && index.model_revision && uuidPattern.test(index.configuration_generation))
}
export function usePowerXKnowledgeWorkspace(basePath: string): KnowledgeWorkspaceContext {
  const { get, post, patch } = useApiClient()
  const contract = ref<HostIngestionCapabilities>()
  const capabilities = reactive({ documents: false, sources: false, strategyWrite: false, ingestionSettings: false, profiles: false, vectorStatus: false, realtime: false })
  const api: KnowledgeWorkspaceContext['api'] = {
    async spaces() {
      const out = data(await get(`${base}/spaces`))
      if (!Array.isArray(out?.spaces) || out.spaces.some((s: any) => !uuidPattern.test(s.id) || !s.name)) invalid()
      return { items: out.spaces.map((s: any): LocalKnowledgeSpace => ({ uuid: s.id, name: s.name, status: s.status, department_code: s.department_code || '', ingestion_profile_key: s.ingestion_profile_key || '', index_profile_key: s.index_profile_key || '', rag_profile_key: s.rag_profile_key || '', embedding_profile_key: '', feature_flags: s.feature_flags, created_at: '', updated_at: '' })) }
    },
    async strategyPackages() {
      const catalog = data(await get(`${base}/catalog`))
      contract.value = undefined
      capabilities.ingestionSettings = false
      const published = catalog.document_ingestion
      if (published) {
        if (published.schema !== documentIngestionSchema || !['lexical_chunks','semantic','hybrid'].includes(published.index_mode) || published.length_unit !== 'unicode_rune' || !Array.isArray(published.segment_modes) || !Array.isArray(published.segment_order_items) || !Array.isArray(published.priorities) || !Array.isArray(published.content_types) || !Array.isArray(published.segment_size_policies) || !Number.isInteger(published.chunk_size_max) || !Array.isArray(published.defaults?.segment_order) || !Array.isArray(published.defaults?.separators) || !published.anchor_mode_requirements) invalid()
        contract.value = published
        capabilities.ingestionSettings = true
      }
      return { items: catalog.strategy_packages.map((s: any): KnowledgeStrategyPackage => ({ key: s.key, label: s.label, scene_labels: (s.recommended_scenes || []).map((key: string) => catalog.scenes?.find((scene: any) => scene.key === key)?.label).filter((label: unknown) => typeof label === 'string'), profile_key: s.recommended_profile_key, dependencies: s.dependencies, bundle_prerequisites: s.activation_dependencies || [], ready: s.available, missing_requirements: s.unavailable_reasons })) }
    },
    profileVersions: unsupported, createProfileVersion: unsupported, publishProfileVersion: unsupported, rollbackProfileVersion: unsupported,
    routes: unsupported, saveRoute: unsupported, deleteRoute: unsupported, saveSpace: unsupported, deleteSpace: unsupported,
    vectorIndexStatus: unsupported, activateVectorIndex: unsupported, documents: unsupported, inspectDocument: unsupported,
    documentChunks: unsupported, documentChunk: unsupported, updateDocumentChunk: unsupported, sources: unsupported, createSource: unsupported,
    saveDocument: unsupported, deleteDocument: unsupported, indexDocument: unsupported, jobs: unsupported,
    ingestionJobs: unsupported, ingestionJob: unsupported, ingestionJobChunks: unsupported,
  }
  return {
    mode: 'powerx', basePath, api,
    capabilities,
    get ingestionContract() { return contract.value },
    async submitDocument(spaceUUID, value) {
      if (value.uuid || value.ingestion || value.effective_from || value.effective_to) unsupported()
      if (value.host_ingestion && (!capabilities.ingestionSettings || value.host_ingestion.schema !== documentIngestionSchema)) unsupported()
      const out = data(await post(`${base}/spaces/${encodeURIComponent(spaceUUID)}/documents`, { title: value.title, content: value.content, tags: value.tags || [], content_type: value.content_type, ...(value.host_ingestion ? { ingestion: value.host_ingestion } : {}),...(value.indexing?{indexing:value.indexing}:{}),...(value.artifacts?{artifacts:value.artifacts}:{}),...(value.external_ref?{external_ref:value.external_ref}:{}),...(value.idempotency_key?{idempotency_key:value.idempotency_key}:{}),...(value.uri?{uri:value.uri}:{}),...(value.version?{version:value.version}:{}),...(value.checksum?{checksum:value.checksum}:{}) }))
      const job = out.item
      if (!uuidPattern.test(job?.job_id) || !uuidPattern.test(job?.document_id) || !['queued', 'running', 'succeeded', 'failed'].includes(job.status)) throw { data: { code: 'KNOWLEDGE_INVALID_RESPONSE' } }
      if (job.status === 'failed') throw { data: { code: job.error_code || 'KNOWLEDGE_INDEX_FAILED' } }
      return { documentUUID: job.document_id, jobUUID: job.job_id, status: job.status }
    },
    async rebuildIndex(spaceUUID, documentUUID, request) {
      if (!uuidPattern.test(spaceUUID) || documentUUID !== undefined && !uuidPattern.test(documentUUID) || !request.idempotency_key.trim() || request.idempotency_key !== request.idempotency_key.trim() || request.idempotency_key.length > 128 || !['reuse_snapshot','apply_config'].includes(request.mode) || request.mode === 'reuse_snapshot' && (request.ingestion || (request as any).indexing) || request.mode === 'apply_config' && (!request.ingestion && !request.indexing || request.ingestion && request.ingestion.schema !== documentIngestionSchema)) throw {data:{code:'KNOWLEDGE_INVALID_ARGUMENT'}}
      const path = `${base}/spaces/${encodeURIComponent(spaceUUID)}${documentUUID ? '/documents/' + encodeURIComponent(documentUUID) : ''}/indexes:rebuild`
      const out = data(await post(path, request)); const job = out.item
      if (!uuidPattern.test(job?.job_id) || (job.document_id || undefined) !== documentUUID || !['queued','running','succeeded','failed'].includes(job.status)) invalid()
      return job
    },
    async indexJob(jobUUID) { const out = data(await get(`${base}/index-jobs/${encodeURIComponent(jobUUID)}`)); if (!out?.item || out.item.job_id !== jobUUID || !['queued','running','succeeded','failed'].includes(out.item.status)) invalid(); const job = out.item
      if (job.effective_config && (!snapshotValid(job.effective_config) || !Number.isInteger(job.chunk_count) || job.chunk_count < 0 || !/^[a-f0-9]{64}$/.test(job.snapshot_checksum))) invalid()
      if (job.rebuild_mode) {
        if (!['reuse_snapshot','apply_config'].includes(job.rebuild_mode) || !Number.isInteger(job.document_count) || job.document_count < 1 || !Number.isInteger(job.processed_documents) || job.processed_documents < 0 || job.processed_documents > job.document_count || !Array.isArray(job.document_configs) || job.document_configs.length !== job.document_count || new Set(job.document_configs.map((item:any)=>item.document_uuid)).size !== job.document_count || job.document_configs.some((item:any)=>!uuidPattern.test(item.document_uuid) || !snapshotValid(item.effective_config))) invalid()
        if (job.document_id && (job.document_count !== 1 || job.document_configs[0].document_uuid !== job.document_id)) invalid()
      }
      return job },
    async indexJobChunks(jobUUID) { const out = data(await get(`${base}/index-jobs/${encodeURIComponent(jobUUID)}/chunks`)); if (!Array.isArray(out.items) || out.items.some((c: any) => !uuidPattern.test(c.uuid) || c.job_uuid !== jobUUID || !c.checksum)) invalid(); return out.items },
    async semanticCapabilities(spaceUUID) {
      if(!uuidPattern.test(spaceUUID))invalid()
      const out=data(await get(`${base}/spaces/${encodeURIComponent(spaceUUID)}/semantic-index`));const item=out.capabilities as SemanticCapabilities
      if(item?.schema!=='powerx.knowledge.semantic-capabilities/v1'||typeof item.configured!=='boolean'||typeof item.ready!=='boolean'||typeof item.indexed!=='boolean'||!Array.isArray(item.query_modes)||item.indexed&&!item.ready||item.configured&&!item.generation)invalid()
      return item
    },
    async configureSemantic(spaceUUID,input){const out=data(await post(`${base}/spaces/${encodeURIComponent(spaceUUID)}/semantic-index`,input));if(out.generation?.space_uuid!==spaceUUID||!uuidPattern.test(out.generation?.embedding_profile?.uuid)||!out.generation?.model_revision)invalid();return out.generation},
    async semanticQuery(request){const out=data(await post(`${base}/retrieval/query`,request));const result=out.result;if(result?.schema!=='powerx.knowledge.retrieval-result/v1'||result.requested_mode!==request.mode||result.effective_mode!==request.mode||!Array.isArray(result.items)||!Array.isArray(result.generations)||result.items.some((item:any)=>!request.space_uuids.includes(item.space_uuid)||!uuidPattern.test(item.chunk_uuid)||!Array.isArray(item.scores)))invalid();return result},
    async setVisibility(spaceUUID,documentUUID,input){const out=data(await patch(`${base}/spaces/${encodeURIComponent(spaceUUID)}/documents/${encodeURIComponent(documentUUID)}/visibility`,input));if(out.item?.document_uuid!==documentUUID||out.item?.queryable!==input.queryable||!uuidPattern.test(out.item?.visibility_epoch))invalid();return out.item},
    async search(spaceUUID, query, limit) { return data(await post(`${base}/search`, { space_id: spaceUUID, query, limit })) },
  }
}
