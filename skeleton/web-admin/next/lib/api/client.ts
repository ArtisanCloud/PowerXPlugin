import { normalizeApiError } from './normalizeApiError'
import { resolveApiBase } from './baseUrl'
import { getAccessToken } from '../auth/session'

export type ApiEnvelope<T> = {
  code: number
  message: string
  data: T
}

export type ApiRequestOptions = {
  pathname?: string
  headers?: Record<string, string>
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  body?: unknown
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const base = resolveApiBase(options.pathname)
  const token = getAccessToken()

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }

  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(`${base}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
    credentials: 'include',
  })

  if (!response.ok) {
    throw await normalizeApiError(response)
  }

  const envelope = (await response.json()) as ApiEnvelope<T>
  return envelope.data
}
