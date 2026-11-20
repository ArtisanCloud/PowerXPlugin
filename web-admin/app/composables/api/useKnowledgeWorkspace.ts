import type {SemanticCapabilities,SemanticGeneration,SemanticQuery,SemanticResult,SemanticDocument} from './knowledgeSemanticTypes'
import type { HostIngestionSettings, HostIngestionCapabilities, HostIndexJob, HostIndexJobChunk, HostRebuildRequest } from './knowledgeHostTypes'
import { useApiClient } from './_client'
import { inject, type InjectionKey } from 'vue'
import { useLocalKnowledgeApi, type LocalKnowledgeDocument } from './useLocalKnowledge'
import { localContentOverview, type KnowledgeContentOverview, type KnowledgeContentRequest } from '../knowledge/contentOverview'

export type KnowledgeWorkspaceContext = {
  mode: 'local' | 'powerx'
  basePath: string
  api: ReturnType<typeof useLocalKnowledgeApi>
  capabilities: { documents: boolean; sources: boolean; strategyWrite: boolean; ingestionSettings: boolean; profiles: boolean; vectorStatus: boolean; realtime: boolean }
  submitDocument: (spaceUUID: string, value: Partial<LocalKnowledgeDocument> & { content_type?: string; host_ingestion?: HostIngestionSettings } & SemanticDocument) => Promise<{ documentUUID: string; jobUUID: string; status: string }>
  ingestionContract?: HostIngestionCapabilities
  semanticCapabilities?: (spaceUUID:string)=>Promise<SemanticCapabilities>
  contentOverview?: (spaceUUID: string, request: KnowledgeContentRequest) => Promise<KnowledgeContentOverview>
  configureSemantic?: (spaceUUID:string,input:{embedding_profile_key:string;expected_configuration_generation?:string})=>Promise<SemanticGeneration>
  semanticQuery?: (query:SemanticQuery)=>Promise<SemanticResult>
  setVisibility?: (spaceUUID:string,documentUUID:string,input:{queryable:boolean;expected_epoch?:string})=>Promise<{document_uuid:string;queryable:boolean;visibility_epoch:string}>
  indexJob?: (jobUUID: string) => Promise<HostIndexJob>
  rebuildIndex?: (spaceUUID: string, documentUUID: string | undefined, request: HostRebuildRequest) => Promise<HostIndexJob>
  indexJobChunks?: (jobUUID: string) => Promise<HostIndexJobChunk[]>
  search: (spaceUUID: string, query: string, limit: number) => Promise<any>
}
export const knowledgeWorkspaceKey: InjectionKey<KnowledgeWorkspaceContext> = Symbol('knowledge-workspace')
export function useKnowledgeWorkspace(): KnowledgeWorkspaceContext {
  const provided = inject(knowledgeWorkspaceKey, null)
  if (provided) return provided
  const config = useRuntimeConfig()
  const api = useLocalKnowledgeApi()
  const basePath = `${String(config.public?.pluginAdminBase || '/_p/com.powerx.plugins.base/admin/').replace(/\/+$/, '')}/knowledge`
  const { post } = useApiClient()
  return {
    mode: 'local', basePath, api,
    capabilities: { documents: true, sources: true, strategyWrite: true, ingestionSettings: true, profiles: true, vectorStatus: true, realtime: true },
    async contentOverview(spaceUUID) {
      const out = await api.documents(spaceUUID)
      return localContentOverview(spaceUUID, out.items)
    },
    async submitDocument(spaceUUID, value) {
      if (value.host_ingestion || value.indexing || value.external_ref || value.artifacts?.length) throw { data: { code: 'KNOWLEDGE_UNSUPPORTED_CAPABILITY' } }
      const { content_type, host_ingestion, ...documentInput } = value
      const document = await api.saveDocument(spaceUUID, documentInput)
      const job = await api.indexDocument(document.uuid)
      return { documentUUID: document.uuid, jobUUID: job.uuid, status: job.status }
    },
    async search(spaceUUID, query, limit) { const out: any = await post('/admin/runtime/knowledge/search', { space_id: spaceUUID, query, limit }); return out?.data ?? out },
  } satisfies KnowledgeWorkspaceContext
}
