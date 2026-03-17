'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import {
  addIamRoleMembers,
  createIamRole,
  deleteIamRole,
  getIamRole,
  listIamMembers,
  listIamPermissions,
  listIamRoles,
  listIamTenants,
  removeIamRoleMembers,
  replaceIamRolePermissions,
  updateIamRole,
  type IamMember,
  type IamPermission,
  type IamRole,
  type IamTenant,
} from '@/lib/api/iam'

type RoleScope = 'system' | 'tenant'

export default function IamRolesPage() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const [tenants, setTenants] = useState<IamTenant[]>([])
  const [selectedTenant, setSelectedTenant] = useState('')
  const [roles, setRoles] = useState<IamRole[]>([])

  const [search, setSearch] = useState('')
  const [scopeFilter, setScopeFilter] = useState<'all' | RoleScope>('all')
  const [typeFilter, setTypeFilter] = useState<'all' | 'builtin' | 'custom'>('all')

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const [roleModalOpen, setRoleModalOpen] = useState(false)
  const [editingRoleId, setEditingRoleId] = useState<number | null>(null)
  const [cloneSourceId, setCloneSourceId] = useState<number | null>(null)
  const [permissionDrawerOpen, setPermissionDrawerOpen] = useState(false)
  const [membersDrawerOpen, setMembersDrawerOpen] = useState(false)
  const [activeRole, setActiveRole] = useState<IamRole | null>(null)
  const [permissionSearch, setPermissionSearch] = useState('')
  const [permissionLoading, setPermissionLoading] = useState(false)
  const [permissionSaving, setPermissionSaving] = useState(false)
  const [permissionList, setPermissionList] = useState<IamPermission[]>([])
  const [permissionSelection, setPermissionSelection] = useState<number[]>([])
  const [membersSearch, setMembersSearch] = useState('')
  const [membersLoading, setMembersLoading] = useState(false)
  const [membersSaving, setMembersSaving] = useState(false)
  const [memberCandidates, setMemberCandidates] = useState<IamMember[]>([])
  const [memberSelection, setMemberSelection] = useState<number[]>([])
  const [initialMemberSelection, setInitialMemberSelection] = useState<number[]>([])
  const [roleName, setRoleName] = useState('')
  const [roleCode, setRoleCode] = useState('')
  const [roleScope, setRoleScope] = useState<RoleScope>('tenant')
  const [roleDesc, setRoleDesc] = useState('')
  const [roleTenant, setRoleTenant] = useState('')

  const resetRoleForm = useCallback(() => {
    setEditingRoleId(null)
    setCloneSourceId(null)
    setRoleName('')
    setRoleCode('')
    setRoleScope('tenant')
    setRoleDesc('')
    setRoleTenant(selectedTenant)
  }, [selectedTenant])

  const loadRoles = useCallback(async () => {
    if (!selectedTenant) {
      setRoles([])
      return
    }
    setLoading(true)
    setError('')
    try {
      const payload = await listIamRoles({
        tenant_uuid: selectedTenant,
        q: search.trim() || undefined,
        scope_type: scopeFilter === 'all' ? undefined : scopeFilter,
      })
      setRoles(payload.list || [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载角色失败')
    } finally {
      setLoading(false)
    }
  }, [scopeFilter, search, selectedTenant])

  const loadInitial = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const tenantPayload = await listIamTenants({ page: 1, page_size: 50 })
      const nextTenants = tenantPayload.list || []
      setTenants(nextTenants)
      const firstTenant = nextTenants[0]?.uuid || nextTenants[0]?.key || ''
      setSelectedTenant(firstTenant)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载角色失败')
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadInitial()
  }, [loadInitial])

  useEffect(() => {
    void loadRoles()
  }, [loadRoles])

  useEffect(() => {
    setPage(1)
  }, [search, scopeFilter, typeFilter, selectedTenant, pageSize])

  const filteredRoles = useMemo(() => {
    if (typeFilter === 'all') return roles
    const builtin = typeFilter === 'builtin'
    return roles.filter((item) => Boolean(item.builtin) === builtin)
  }, [roles, typeFilter])

  const totalPages = useMemo(() => Math.max(1, Math.ceil(filteredRoles.length / Math.max(1, pageSize))), [filteredRoles.length, pageSize])
  const pageItems = useMemo(() => {
    if (totalPages <= 7) return Array.from({ length: totalPages }, (_, i) => i + 1)
    if (page <= 4) return [1, 2, 3, 4, 5, totalPages]
    if (page >= totalPages - 3) return [1, totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages]
    return [1, page - 1, page, page + 1, totalPages]
  }, [page, totalPages])
  const pageRows = useMemo(() => {
    const start = (page - 1) * pageSize
    return filteredRoles.slice(start, start + pageSize)
  }, [filteredRoles, page, pageSize])

  useEffect(() => {
    if (page > totalPages) {
      setPage(totalPages)
    }
  }, [page, totalPages])

  const openCreateRole = () => {
    resetRoleForm()
    setRoleModalOpen(true)
  }

  const openEditRole = (role: IamRole) => {
    setEditingRoleId(role.id)
    setCloneSourceId(null)
    setRoleName(role.name || '')
    setRoleCode(role.code || '')
    setRoleScope(((role.scope_type || 'tenant') as RoleScope) || 'tenant')
    setRoleDesc(role.description || '')
    setRoleTenant(role.tenant_uuid || selectedTenant)
    setRoleModalOpen(true)
  }

  const openCloneRole = (role: IamRole) => {
    setEditingRoleId(null)
    setCloneSourceId(role.id)
    setRoleName(`${role.name || ''} Copy`)
    setRoleCode(`${role.code || ''}_copy`)
    setRoleScope(((role.scope_type || 'tenant') as RoleScope) || 'tenant')
    setRoleDesc(role.description || '')
    setRoleTenant(role.tenant_uuid || selectedTenant)
    setRoleModalOpen(true)
  }

  const resolveMemberId = (member: IamMember) => Number(member.member_id ?? member.id ?? 0)

  const openPermissionDrawer = async (role: IamRole) => {
    if (!selectedTenant) return
    setActiveRole(role)
    setPermissionLoading(true)
    setPermissionSearch('')
    setPermissionDrawerOpen(true)
    try {
      const [catalog, detail] = await Promise.all([
        listIamPermissions(),
        getIamRole(role.id),
      ])
      setPermissionList(catalog.list || [])
      setPermissionSelection(detail.permission_ids || [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载角色权限失败')
      setPermissionDrawerOpen(false)
    } finally {
      setPermissionLoading(false)
    }
  }

  const openMembersDrawer = async (role: IamRole) => {
    if (!selectedTenant) return
    setActiveRole(role)
    setMembersLoading(true)
    setMembersSearch('')
    setMembersDrawerOpen(true)
    try {
      const [membersPayload, detail] = await Promise.all([
        listIamMembers({ tenant_uuid: selectedTenant, page: 1, page_size: 200 }),
        getIamRole(role.id),
      ])
      const members = membersPayload.list || []
      const selected = detail.member_ids || []
      setMemberCandidates(members)
      setMemberSelection(selected)
      setInitialMemberSelection(selected)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载角色成员失败')
      setMembersDrawerOpen(false)
    } finally {
      setMembersLoading(false)
    }
  }

  const togglePermission = (id: number) => {
    setPermissionSelection((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const savePermissionSelection = async () => {
    if (!activeRole || !selectedTenant) return
    setPermissionSaving(true)
    setError('')
    try {
      await replaceIamRolePermissions(activeRole.id, {
        tenant_uuid: selectedTenant,
        permission_ids: permissionSelection,
      })
      setPermissionDrawerOpen(false)
      await loadRoles()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '保存角色权限失败')
    } finally {
      setPermissionSaving(false)
    }
  }

  const toggleMember = (memberId: number) => {
    setMemberSelection((prev) => (prev.includes(memberId) ? prev.filter((x) => x !== memberId) : [...prev, memberId]))
  }

  const saveMembersSelection = async () => {
    if (!activeRole || !selectedTenant) return
    const prev = new Set(initialMemberSelection)
    const next = new Set(memberSelection)
    const toAdd = [...next].filter((id) => !prev.has(id))
    const toRemove = [...prev].filter((id) => !next.has(id))
    setMembersSaving(true)
    setError('')
    try {
      if (toAdd.length) {
        await addIamRoleMembers(activeRole.id, { tenant_uuid: selectedTenant, member_ids: toAdd })
      }
      if (toRemove.length) {
        await removeIamRoleMembers(activeRole.id, { tenant_uuid: selectedTenant, member_ids: toRemove })
      }
      setMembersDrawerOpen(false)
      await loadRoles()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '保存角色成员失败')
    } finally {
      setMembersSaving(false)
    }
  }

  const submitRole = async () => {
    const tenantUuid = roleTenant || selectedTenant
    if (!tenantUuid || !roleName.trim() || !roleCode.trim()) {
      setError('请填写必填项（租户/角色名称/角色编码）')
      return
    }
    setSaving(true)
    setError('')
    try {
      if (editingRoleId) {
        await updateIamRole(editingRoleId, {
          name: roleName.trim(),
          description: roleDesc.trim() || undefined,
          scope_type: roleScope,
        })
      } else {
        await createIamRole({
          tenant_uuid: tenantUuid,
          code: roleCode.trim(),
          name: roleName.trim(),
          description: roleDesc.trim() || undefined,
          scope_type: roleScope,
          clone_role_id: cloneSourceId ?? undefined,
        })
      }
      setRoleModalOpen(false)
      resetRoleForm()
      await loadRoles()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '保存角色失败')
    } finally {
      setSaving(false)
    }
  }

  const removeRole = async (roleId: number) => {
    if (!window.confirm('确认删除该角色吗？')) return
    setError('')
    try {
      await deleteIamRole(roleId)
      await loadRoles()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '删除角色失败')
    }
  }

  const filteredPermissions = useMemo(() => {
    const q = permissionSearch.trim().toLowerCase()
    if (!q) return permissionList
    return permissionList.filter((item) => {
      const text = `${item.resource || ''} ${item.action || ''} ${item.description || ''}`.toLowerCase()
      return text.includes(q)
    })
  }, [permissionList, permissionSearch])

  const filteredMemberCandidates = useMemo(() => {
    const q = membersSearch.trim().toLowerCase()
    if (!q) return memberCandidates
    return memberCandidates.filter((item) => {
      const text = `${item.display_name || ''} ${item.username || ''} ${item.email || ''}`.toLowerCase()
      return text.includes(q)
    })
  }, [memberCandidates, membersSearch])

  return (
    <main className="px-admin-page" data-testid="iam-roles-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <section className="px-cap-hero">
            <p className="px-cap-kicker">IAM Roles</p>
            <h1 className="px-cap-page-title">角色权限</h1>
            <p className="px-cap-page-desc">管理角色、作用域与成员授权。</p>
          </section>

          <div className="px-admin-toolbar" style={{ justifyContent: 'space-between', marginTop: 12 }}>
            <select className="px-select" value={selectedTenant} onChange={(e) => setSelectedTenant(e.target.value)} style={{ minWidth: 280 }}>
              {tenants.map((item) => (
                <option key={item.uuid || item.key} value={item.uuid || item.key}>
                  {item.name}
                </option>
              ))}
            </select>
            <button type="button" className="px-btn" onClick={openCreateRole}>新增角色</button>
          </div>

          <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1.6fr 0.8fr 0.8fr auto', gap: 10 }}>
            <input className="px-field" value={search} placeholder="搜索角色名称/编码" onChange={(e) => setSearch(e.target.value)} />
            <select className="px-select" value={scopeFilter} onChange={(e) => setScopeFilter(e.target.value as 'all' | RoleScope)}>
              <option value="all">全部作用域</option>
              <option value="system">系统</option>
              <option value="tenant">租户</option>
            </select>
            <select className="px-select" value={typeFilter} onChange={(e) => setTypeFilter(e.target.value as 'all' | 'builtin' | 'custom')}>
              <option value="all">全部类型</option>
              <option value="builtin">系统内置</option>
              <option value="custom">自定义</option>
            </select>
            <button type="button" className="px-btn-ghost" onClick={() => { setSearch(''); setScopeFilter('all'); setTypeFilter('all') }}>重置</button>
          </div>

          {error ? <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>{error}</p> : null}

          <div className="px-table-wrap" style={{ marginTop: 12 }}>
            <table className="px-table">
              <thead>
                <tr>
                  <th>角色名称</th>
                  <th>角色编码</th>
                  <th>作用域</th>
                  <th>成员数</th>
                  <th>描述</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={6} className="px-admin-card-text">加载中...</td></tr>
                ) : pageRows.length === 0 ? (
                  <tr><td colSpan={6} className="px-admin-card-text">暂无角色数据</td></tr>
                ) : (
                  pageRows.map((item) => (
                    <tr key={item.id}>
                      <td>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <span style={{ fontWeight: 600 }}>{item.name || '-'}</span>
                          {item.builtin ? <span className="px-cap-tag">系统</span> : null}
                        </div>
                      </td>
                      <td>{item.code || '-'}</td>
                      <td>{item.scope_type === 'system' ? '系统' : '租户'}</td>
                      <td>{item.member_count ?? 0}</td>
                      <td>{item.description || '-'}</td>
                      <td>
                        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                          <button
                            type="button"
                            className="px-btn-ghost"
                            style={{ height: 26, padding: '0 8px', fontSize: 12 }}
                            onClick={() => openPermissionDrawer(item)}
                          >
                            权限
                          </button>
                          <button
                            type="button"
                            className="px-btn-ghost"
                            style={{ height: 26, padding: '0 8px', fontSize: 12 }}
                            onClick={() => openMembersDrawer(item)}
                          >
                            成员
                          </button>
                          <button type="button" className="px-btn-ghost" style={{ height: 26, padding: '0 8px', fontSize: 12 }} onClick={() => openCloneRole(item)}>克隆</button>
                          <button type="button" className="px-btn-ghost" style={{ height: 26, padding: '0 8px', fontSize: 12 }} onClick={() => openEditRole(item)}>编辑</button>
                          {!item.builtin ? (
                            <button type="button" className="px-btn-ghost" style={{ height: 26, padding: '0 8px', fontSize: 12, color: '#ef4444' }} onClick={() => void removeRole(item.id)}>删除</button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {filteredRoles.length > 0 ? (
            <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
              <div className="px-admin-card-text">共 {filteredRoles.length} 条</div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <select className="px-select" value={String(pageSize)} onChange={(e) => setPageSize(Number(e.target.value) || 10)} style={{ width: 92 }}>
                  <option value="10">10 / 页</option>
                  <option value="20">20 / 页</option>
                  <option value="50">50 / 页</option>
                </select>
                <button type="button" className="px-btn-ghost" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>上一页</button>
                {pageItems.map((p) => (
                  <button key={p} type="button" className={p === page ? 'px-btn' : 'px-btn-ghost'} onClick={() => setPage(p)}>
                    {p}
                  </button>
                ))}
                <button type="button" className="px-btn-ghost" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>下一页</button>
              </div>
            </div>
          ) : null}
        </article>

        {roleModalOpen ? (
          <div
            role="dialog"
            aria-modal="true"
            style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }}
            onClick={() => setRoleModalOpen(false)}
          >
            <article className="px-admin-card" style={{ width: 'min(860px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'flex-start' }}>
                <div>
                  <h3 className="px-admin-card-title">{editingRoleId ? '编辑角色' : '新建角色'}</h3>
                  <p className="px-admin-card-text">填写角色名称、作用域与所属租户。</p>
                </div>
                <button type="button" className="px-btn-ghost" onClick={() => setRoleModalOpen(false)}>✕</button>
              </div>

              <div style={{ marginTop: 12, borderTop: '1px solid #e5e7eb', paddingTop: 14, display: 'grid', gap: 14 }}>
                <label>
                  <div className="px-admin-card-text">角色名称 *</div>
                  <input className="px-field" value={roleName} onChange={(e) => setRoleName(e.target.value)} />
                </label>
                <label>
                  <div className="px-admin-card-text">角色代码 *</div>
                  <input className="px-field" value={roleCode} onChange={(e) => setRoleCode(e.target.value)} readOnly={Boolean(editingRoleId)} />
                </label>
                <label>
                  <div className="px-admin-card-text">作用域</div>
                  <select className="px-select" value={roleScope} onChange={(e) => setRoleScope(e.target.value as RoleScope)}>
                    <option value="tenant">租户角色</option>
                    <option value="system">系统角色</option>
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">归属租户 *</div>
                  <select className="px-select" value={roleTenant} onChange={(e) => setRoleTenant(e.target.value)}>
                    {tenants.map((item) => (
                      <option key={item.uuid || item.key} value={item.uuid || item.key}>
                        {item.key || item.uuid || item.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">角色描述</div>
                  <textarea value={roleDesc} onChange={(e) => setRoleDesc(e.target.value)} style={{ width: '100%', minHeight: 96, borderRadius: 10, border: '1px solid #cbd5e1', padding: 10, fontSize: 14 }} />
                </label>
              </div>

              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setRoleModalOpen(false)}>取消</button>
                <button type="button" className="px-btn" disabled={saving} onClick={() => void submitRole()}>{saving ? '保存中' : '保存'}</button>
              </div>
            </article>
          </div>
        ) : null}

        {permissionDrawerOpen ? (
          <div
            role="dialog"
            aria-modal="true"
            style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }}
            onClick={() => setPermissionDrawerOpen(false)}
          >
            <article className="px-admin-card" style={{ width: 'min(920px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <h3 className="px-admin-card-title">{activeRole?.name || '角色权限'}</h3>
              <div style={{ marginTop: 10 }}>
                <input
                  className="px-field"
                  value={permissionSearch}
                  placeholder="搜索权限资源、动作、描述"
                  onChange={(e) => setPermissionSearch(e.target.value)}
                />
              </div>
              <div style={{ marginTop: 10, maxHeight: 420, overflow: 'auto', border: '1px solid #e5e7eb', borderRadius: 10 }}>
                {permissionLoading ? (
                  <div className="px-admin-card-text" style={{ padding: 12 }}>加载中...</div>
                ) : filteredPermissions.length === 0 ? (
                  <div className="px-admin-card-text" style={{ padding: 12 }}>暂无权限数据</div>
                ) : (
                  filteredPermissions.map((item) => {
                    const code = `${item.resource || '-'}:${item.action || '-'}`
                    return (
                      <label key={item.id} style={{ display: 'flex', gap: 10, alignItems: 'flex-start', padding: '10px 12px', borderBottom: '1px solid #f1f5f9', cursor: 'pointer' }}>
                        <input type="checkbox" checked={permissionSelection.includes(item.id)} onChange={() => togglePermission(item.id)} />
                        <div>
                          <div style={{ fontWeight: 600, fontSize: 13 }}>{code}</div>
                          <div className="px-admin-card-text">{item.description || '-'}</div>
                        </div>
                      </label>
                    )
                  })
                )}
              </div>
              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setPermissionDrawerOpen(false)}>取消</button>
                <button type="button" className="px-btn" disabled={permissionSaving} onClick={() => void savePermissionSelection()}>
                  {permissionSaving ? '保存中' : '保存'}
                </button>
              </div>
            </article>
          </div>
        ) : null}

        {membersDrawerOpen ? (
          <div
            role="dialog"
            aria-modal="true"
            style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }}
            onClick={() => setMembersDrawerOpen(false)}
          >
            <article className="px-admin-card" style={{ width: 'min(920px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <h3 className="px-admin-card-title">{activeRole?.name || '角色成员'}</h3>
              <div style={{ marginTop: 10 }}>
                <input
                  className="px-field"
                  value={membersSearch}
                  placeholder="搜索成员姓名、用户名、邮箱"
                  onChange={(e) => setMembersSearch(e.target.value)}
                />
              </div>
              <div style={{ marginTop: 10, maxHeight: 420, overflow: 'auto', border: '1px solid #e5e7eb', borderRadius: 10 }}>
                {membersLoading ? (
                  <div className="px-admin-card-text" style={{ padding: 12 }}>加载中...</div>
                ) : filteredMemberCandidates.length === 0 ? (
                  <div className="px-admin-card-text" style={{ padding: 12 }}>暂无成员数据</div>
                ) : (
                  filteredMemberCandidates.map((item) => {
                    const memberId = resolveMemberId(item)
                    const title = item.display_name || item.username || `Member#${memberId}`
                    return (
                      <label key={memberId} style={{ display: 'flex', gap: 10, alignItems: 'flex-start', padding: '10px 12px', borderBottom: '1px solid #f1f5f9', cursor: 'pointer' }}>
                        <input type="checkbox" checked={memberSelection.includes(memberId)} onChange={() => toggleMember(memberId)} />
                        <div>
                          <div style={{ fontWeight: 600, fontSize: 13 }}>{title}</div>
                          <div className="px-admin-card-text">{item.email || item.username || '-'}</div>
                        </div>
                      </label>
                    )
                  })
                )}
              </div>
              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setMembersDrawerOpen(false)}>取消</button>
                <button type="button" className="px-btn" disabled={membersSaving} onClick={() => void saveMembersSelection()}>
                  {membersSaving ? '保存中' : '保存'}
                </button>
              </div>
            </article>
          </div>
        ) : null}
      </section>
    </main>
  )
}
