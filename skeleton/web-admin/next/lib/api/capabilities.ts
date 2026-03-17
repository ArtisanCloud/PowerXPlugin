import { apiRequest } from './client'

export async function getCapabilityLifecycle(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/capabilities/lifecycle')
}

export async function getCapabilityRegister(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/capabilities/register/template')
}

export async function listCapabilitiesCatalog(source = ''): Promise<Record<string, unknown>[]> {
  const query = source.trim() ? `?source=${encodeURIComponent(source.trim())}` : ''
  return apiRequest<Record<string, unknown>[]>(`/admin/capabilities${query}`)
}

export async function getCapabilityExposure(capabilityID: string): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>(`/admin/capabilities/exposure/${encodeURIComponent(capabilityID)}`)
}

export async function getCapabilityExposureTemplate(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/capabilities/exposure/template')
}

export async function upsertCapabilityExposure(
  capabilityID: string,
  payload: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>(`/admin/capabilities/exposure/${encodeURIComponent(capabilityID)}`, {
    method: 'PUT',
    body: payload,
  })
}

export async function invokeCapability(
  payload: Record<string, unknown>,
  headers: Record<string, string> = {},
): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/integration/capabilities/invoke', {
    method: 'POST',
    body: payload,
    headers,
  })
}
