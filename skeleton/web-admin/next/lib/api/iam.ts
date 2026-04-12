import { apiRequest } from './client'

export type IamTenant = {
  id: number
  uuid?: string
  key: string
  name: string
  status: string
  plan: string
  member_count?: number
  role_count?: number
  user_count?: number
  created_at?: string
  updated_at?: string
}

export type IamMember = {
  id: string | number
  member_id?: number
  user_id?: number
  tenant_uuid?: string
  username: string
  email?: string
  display_name?: string
  status?: number | string
}

export type IamRole = {
  id: number
  tenant_uuid?: string
  code: string
  name: string
  description?: string
  scope_type?: 'system' | 'tenant' | string
  member_count?: number
  builtin?: boolean
  permission_ids?: number[]
  member_ids?: number[]
}

export type IamPermission = {
  id: number
  resource: string
  action: string
  description?: string
}

export type IamDepartment = {
  id: number
  tenant_uuid: string
  name: string
  code?: string
  parent_id?: number | null
  sort_order?: number
  path?: string
}

type ListPayload<T> = {
  list?: T[]
  items?: T[]
  total?: number
}

function parseList<T>(payload: T[] | ListPayload<T>): { list: T[]; total: number } {
  if (Array.isArray(payload)) {
    return { list: payload, total: payload.length }
  }
  const list = Array.isArray(payload?.list)
    ? payload.list
    : Array.isArray(payload?.items)
      ? payload.items
      : []
  const total = typeof payload?.total === 'number' ? payload.total : list.length
  return { list, total }
}

export async function getIamOverview(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/iam/overview')
}

export async function listIamTenants(params: {
  status?: string
  q?: string
  page?: number
  page_size?: number
} = {}): Promise<{ list: IamTenant[]; total: number }> {
  const search = new URLSearchParams()
  if (params.status) search.set('status', params.status)
  if (params.q) search.set('q', params.q)
  if (typeof params.page === 'number') search.set('page', String(params.page))
  if (typeof params.page_size === 'number') search.set('page_size', String(params.page_size))
  const query = search.toString()
  const payload = await apiRequest<{ items?: IamTenant[]; total?: number }>(`/admin/iam/tenants${query ? `?${query}` : ''}`)
  const list = Array.isArray(payload?.items) ? payload.items : []
  const total = typeof payload?.total === 'number' ? payload.total : list.length
  return { list, total }
}

export async function createIamTenant(payload: {
  key: string
  name: string
  status?: string
  plan?: string
}): Promise<IamTenant> {
  return apiRequest<IamTenant>('/admin/iam/tenants', {
    method: 'POST',
    body: payload,
  })
}

export async function updateIamTenant(
  id: number,
  payload: {
    name?: string
    status?: string
    plan?: string
  },
): Promise<IamTenant> {
  return apiRequest<IamTenant>(`/admin/iam/tenants/${id}`, {
    method: 'PATCH',
    body: payload,
  })
}

export async function listIamMembers(params: {
  tenant_uuid?: string
  q?: string
  status?: string
  page?: number
  page_size?: number
} = {}): Promise<{ list: IamMember[]; total: number }> {
  const search = new URLSearchParams()
  if (params.tenant_uuid) search.set('tenant_uuid', params.tenant_uuid)
  if (params.q) search.set('q', params.q)
  if (params.status) search.set('status', params.status)
  if (typeof params.page === 'number') search.set('page', String(params.page))
  if (typeof params.page_size === 'number') search.set('page_size', String(params.page_size))
  const query = search.toString()
  const payload = await apiRequest<IamMember[] | ListPayload<IamMember>>(`/admin/iam/members${query ? `?${query}` : ''}`)
  return parseList(payload)
}

export async function listIamRoles(params: {
  tenant_uuid?: string
  q?: string
  scope_type?: string
} = {}): Promise<{ list: IamRole[]; total: number }> {
  const search = new URLSearchParams()
  if (params.tenant_uuid) search.set('tenant_uuid', params.tenant_uuid)
  if (params.q) search.set('q', params.q)
  if (params.scope_type) search.set('scope_type', params.scope_type)
  const query = search.toString()
  const payload = await apiRequest<IamRole[] | ListPayload<IamRole>>(`/admin/iam/roles${query ? `?${query}` : ''}`)
  return parseList(payload)
}

export async function getIamRole(id: number): Promise<IamRole> {
  return apiRequest<IamRole>(`/admin/iam/roles/${id}`)
}

export async function createIamRole(payload: {
  tenant_uuid: string
  code: string
  name: string
  description?: string
  scope_type?: 'system' | 'tenant'
  clone_role_id?: number
  permission_ids?: number[]
  member_ids?: number[]
}): Promise<IamRole> {
  return apiRequest<IamRole>('/admin/iam/roles', {
    method: 'POST',
    body: payload,
  })
}

export async function updateIamRole(
  id: number,
  payload: {
    name?: string
    description?: string
    scope_type?: 'system' | 'tenant'
  },
): Promise<IamRole> {
  return apiRequest<IamRole>(`/admin/iam/roles/${id}`, {
    method: 'PATCH',
    body: payload,
  })
}

export async function deleteIamRole(id: number): Promise<{ deleted?: boolean }> {
  return apiRequest<{ deleted?: boolean }>(`/admin/iam/roles/${id}`, {
    method: 'DELETE',
  })
}

export async function replaceIamRolePermissions(
  id: number,
  payload: { tenant_uuid: string; permission_ids: number[] },
): Promise<IamRole> {
  return apiRequest<IamRole>(`/admin/iam/roles/${id}/permissions`, {
    method: 'PUT',
    body: payload,
  })
}

export async function addIamRoleMembers(
  id: number,
  payload: { tenant_uuid: string; member_ids: number[] },
): Promise<{ ok?: boolean }> {
  return apiRequest<{ ok?: boolean }>(`/admin/iam/roles/${id}/members`, {
    method: 'POST',
    body: payload,
  })
}

export async function removeIamRoleMembers(
  id: number,
  payload: { tenant_uuid: string; member_ids: number[] },
): Promise<{ ok?: boolean }> {
  return apiRequest<{ ok?: boolean }>(`/admin/iam/roles/${id}/members`, {
    method: 'DELETE',
    body: payload,
  })
}

export async function listIamDepartments(tenantUuid: string): Promise<{ list: IamDepartment[]; total: number }> {
  const search = new URLSearchParams()
  if (tenantUuid) search.set('tenant_uuid', tenantUuid)
  const query = search.toString()
  const payload = await apiRequest<IamDepartment[] | ListPayload<IamDepartment>>(`/admin/iam/departments${query ? `?${query}` : ''}`)
  return parseList(payload)
}

export async function createIamDepartment(payload: {
  tenant_uuid: string
  name: string
  parent_id?: number
  code?: string
  sort_order?: number
  description?: string
}): Promise<IamDepartment> {
  return apiRequest<IamDepartment>('/admin/iam/departments', {
    method: 'POST',
    body: payload,
  })
}

export async function updateIamDepartment(
  id: number,
  payload: {
    name?: string
    parent_id?: number
    sort_order?: number
    description?: string
  },
): Promise<IamDepartment> {
  return apiRequest<IamDepartment>(`/admin/iam/departments/${id}`, {
    method: 'PATCH',
    body: payload,
  })
}

export async function listIamPermissions(): Promise<{ list: IamPermission[]; total: number }> {
  const payload = await apiRequest<IamPermission[] | ListPayload<IamPermission>>('/admin/iam/permissions')
  return parseList(payload)
}

export async function getIamSettings(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/iam/settings')
}
