import { apiRequest } from './client'

export async function getIntegrationSettings(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/integration/settings')
}

export async function getOperationsOverview(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/operations/overview')
}

export async function getSecurityOverview(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/security/overview')
}
