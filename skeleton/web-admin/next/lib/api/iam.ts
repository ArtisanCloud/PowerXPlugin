import { apiRequest } from './client'

export type IamMember = {
  id: string
  username: string
  display_name?: string
  status?: number
}

export type IamRole = {
  id: string
  code: string
  name: string
  description?: string
}

type ListPayload<T> = {
  list?: T[]
  total?: number
}

function parseList<T>(payload: T[] | ListPayload<T>): { list: T[]; total: number } {
  if (Array.isArray(payload)) {
    return { list: payload, total: payload.length }
  }
  const list = Array.isArray(payload?.list) ? payload.list : []
  const total = typeof payload?.total === 'number' ? payload.total : list.length
  return { list, total }
}

export async function getIamOverview(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/iam/overview')
}

export async function listIamMembers(): Promise<{ list: IamMember[]; total: number }> {
  const payload = await apiRequest<IamMember[] | ListPayload<IamMember>>('/admin/iam/members')
  return parseList(payload)
}

export async function listIamRoles(): Promise<{ list: IamRole[]; total: number }> {
  const payload = await apiRequest<IamRole[] | ListPayload<IamRole>>('/admin/iam/roles')
  return parseList(payload)
}

export async function getIamSettings(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/iam/settings')
}
