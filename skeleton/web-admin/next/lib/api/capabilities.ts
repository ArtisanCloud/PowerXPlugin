import { apiRequest } from './client'

export async function getCapabilityLifecycle(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/capabilities/lifecycle')
}

export async function getCapabilityRegister(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/capabilities/register')
}

export async function invokeCapability(payload: Record<string, unknown>): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/capabilities/invoke', {
    method: 'POST',
    body: payload,
  })
}
