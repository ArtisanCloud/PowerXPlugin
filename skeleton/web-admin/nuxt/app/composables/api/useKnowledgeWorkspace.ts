import { useApiClient } from './_client'
import { inject, type InjectionKey } from 'vue'
import { useLocalKnowledgeApi, type LocalKnowledgeDocument } from './useLocalKnowledge'

export type KnowledgeWorkspaceContext = {
  mode: 'local' | 'powerx'
  basePath: string
  api: ReturnType<typeof useLocalKnowledgeApi>
  capabilities: { documents: boolean; sources: boolean; strategyWrite: boolean; ingestionSettings: boolean; profiles: boolean; vectorStatus: boolean; realtime: boolean }
  submitDocument: (spaceUUID: string, value: Partial<LocalKnowledgeDocument> & { content_type?: string }) => Promise<{ documentUUID: string; jobUUID: string; status: string }>
  indexJob?: (jobUUID: string) => Promise<{ job_id: string; status: string; error_code?: string }>
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
    async submitDocument(spaceUUID, value) {
      const { content_type, ...documentInput } = value
      const document = await api.saveDocument(spaceUUID, documentInput)
      const job = await api.indexDocument(document.uuid)
      return { documentUUID: document.uuid, jobUUID: job.uuid, status: job.status }
    },
    async search(spaceUUID, query, limit) { const out: any = await post('/admin/runtime/knowledge/search', { space_id: spaceUUID, query, limit }); return out?.data ?? out },
  } satisfies KnowledgeWorkspaceContext
}
