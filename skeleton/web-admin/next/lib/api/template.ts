import { apiRequest } from './client'

export type Template = {
  id: number
  name: string
  description: string
  content: string
  created_at?: string
  updated_at?: string
}

export type TemplatePayload = {
  name: string
  description: string
  content: string
}

type TemplateListEnvelope = {
  list?: Template[]
  total?: number
}

function parseListPayload(payload: Template[] | TemplateListEnvelope): { list: Template[]; total: number } {
  if (Array.isArray(payload)) {
    return { list: payload, total: payload.length }
  }

  const list = Array.isArray(payload?.list) ? payload.list : []
  const total = typeof payload?.total === 'number' ? payload.total : list.length
  return { list, total }
}

export async function listTemplates(page = 1, pageSize = 20, q = ''): Promise<{ list: Template[]; total: number }> {
  const params = new URLSearchParams()
  params.set('page', String(page))
  params.set('page_size', String(pageSize))
  if (q.trim()) {
    params.set('q', q.trim())
  }

  const payload = await apiRequest<Template[] | TemplateListEnvelope>(`/templates?${params.toString()}`)
  return parseListPayload(payload)
}

export async function createTemplate(payload: TemplatePayload): Promise<Template> {
  return apiRequest<Template>('/templates', {
    method: 'POST',
    body: payload,
  })
}

export async function updateTemplate(id: number | string, payload: TemplatePayload): Promise<Template> {
  return apiRequest<Template>(`/templates/${id}`, {
    method: 'PUT',
    body: payload,
  })
}

export async function deleteTemplate(id: number | string): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>(`/templates/${id}`, {
    method: 'DELETE',
  })
}
