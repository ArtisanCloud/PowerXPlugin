import { apiRequest } from './client'

export type KnowledgeProviderInfo = {
  provider?: string
  source?: string
  mode?: string
  capabilities?: Record<string, unknown>
  diagnostics?: Record<string, unknown>
}

export type KnowledgeSearchRequest = {
  tenant_uuid?: string
  query: string
  top_k?: number
  filters?: Record<string, unknown>
}

export type KnowledgeSearchResult = {
  query?: string
  results?: unknown[]
  citations?: unknown[]
  diagnostics?: Record<string, unknown>
}

export async function getKnowledgeProvider(): Promise<KnowledgeProviderInfo> {
  return apiRequest<KnowledgeProviderInfo>('/admin/runtime/knowledge/provider')
}

export async function searchKnowledge(payload: KnowledgeSearchRequest): Promise<KnowledgeSearchResult> {
  return apiRequest<KnowledgeSearchResult>('/admin/runtime/knowledge/search', {
    method: 'POST',
    body: payload,
  })
}
