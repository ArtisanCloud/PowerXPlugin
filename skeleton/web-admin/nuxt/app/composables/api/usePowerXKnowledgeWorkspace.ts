import { useApiClient } from './_client'
import type { KnowledgeWorkspaceContext } from './useKnowledgeWorkspace'
import type { KnowledgeStrategyPackage, LocalKnowledgeSpace } from './useLocalKnowledge'

const base = '/admin/runtime/knowledge-lab'
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
function invalid(): never { throw { data: { code: 'KNOWLEDGE_INVALID_RESPONSE' } } }
function data(out: any) { return out?.data ?? out }
function unsupported(): never { throw { data: { code: 'KNOWLEDGE_UNSUPPORTED_CAPABILITY', error: 'knowledgeLab.workspace.unsupported' } } }
export function usePowerXKnowledgeWorkspace(basePath: string): KnowledgeWorkspaceContext {
  const { get, post } = useApiClient()
  const api: KnowledgeWorkspaceContext['api'] = {
    async spaces() {
      const out = data(await get(`${base}/spaces`))
      if (!Array.isArray(out?.spaces) || out.spaces.some((s: any) => !uuidPattern.test(s.id) || !s.name)) invalid()
      return { items: out.spaces.map((s: any): LocalKnowledgeSpace => ({ uuid: s.id, name: s.name, status: s.status, department_code: s.department_code || '', ingestion_profile_key: s.ingestion_profile_key || '', index_profile_key: s.index_profile_key || '', rag_profile_key: s.rag_profile_key || '', embedding_profile_key: '', feature_flags: s.feature_flags, created_at: '', updated_at: '' })) }
    },
    async strategyPackages() {
      const catalog = data(await get(`${base}/catalog`))
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
    capabilities: { documents: false, sources: false, strategyWrite: false, ingestionSettings: false, profiles: false, vectorStatus: false, realtime: false },
    async submitDocument(spaceUUID, value) {
      if (value.uuid || value.ingestion || value.effective_from || value.effective_to) unsupported()
      const out = data(await post(`${base}/spaces/${encodeURIComponent(spaceUUID)}/documents`, { title: value.title, content: value.content, tags: value.tags || [], content_type: value.content_type }))
      const job = out.item
      if (!uuidPattern.test(job?.job_id) || !uuidPattern.test(job?.document_id) || !['queued', 'running', 'succeeded', 'failed'].includes(job.status)) throw { data: { code: 'KNOWLEDGE_INVALID_RESPONSE' } }
      if (job.status === 'failed') throw { data: { code: job.error_code || 'KNOWLEDGE_INDEX_FAILED' } }
      return { documentUUID: job.document_id, jobUUID: job.job_id, status: job.status }
    },
    async indexJob(jobUUID) { const out = data(await get(`${base}/index-jobs/${encodeURIComponent(jobUUID)}`)); if (!out?.item || out.item.job_id !== jobUUID || !['queued','running','succeeded','failed'].includes(out.item.status)) invalid(); return out.item },
    async search(spaceUUID, query, limit) { return data(await post(`${base}/search`, { space_id: spaceUUID, query, limit })) },
  }
}
