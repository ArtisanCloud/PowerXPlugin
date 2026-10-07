import { apiGet, apiPost, apiPut } from './_client'
import type { ApiResponse } from './_base'
export type LocalKind = 'agents' | 'skills'
export interface LocalDefinition {
  uuid: string
  key: string
  name: string
  description: string
  status: 'active' | 'inactive'
  model_key: string
  prompt: string
  updated_at: string
  version?: string
  executor?: string
  persona?: string
  skill_uuids?: string[]
}
export interface LocalModel { model_key: string; label?: string; configured: boolean }
export interface LocalPage { items: LocalDefinition[]; total: number; page: number; page_size: number }
export function useLocalIntelligence() {
  const base = 'admin/local-intelligence'
  return {
    list: (kind: LocalKind, query: Record<string, unknown>) => apiGet<ApiResponse<LocalPage>>(`${base}/${kind}`, query),
    models: () => apiGet<ApiResponse<{ items: LocalModel[] }>>(`${base}/models`),
    save: (kind: LocalKind, id: string, body: Record<string, unknown>) => id
      ? apiPut<ApiResponse<LocalDefinition>>(`${base}/${kind}/${encodeURIComponent(id)}`, body)
      : apiPost<ApiResponse<LocalDefinition>>(`${base}/${kind}`, body),
    run: (kind: LocalKind, id: string, body: Record<string, unknown>) =>
      apiPost<ApiResponse<Record<string, unknown>>>(`${base}/${kind}/${encodeURIComponent(id)}/${kind === 'agents' ? 'debug' : 'invoke'}`, body),
  }
}
