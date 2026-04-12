'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import {
  createIamDepartment,
  createIamTenant,
  listIamDepartments,
  listIamMembers,
  listIamPermissions,
  listIamRoles,
  listIamTenants,
  updateIamDepartment,
  type IamDepartment,
  type IamMember,
  type IamPermission,
  type IamRole,
  type IamTenant,
} from '@/lib/api/iam'

type MainTab = 'departments' | 'users' | 'permissions'
type PermissionTab = 'permission' | 'role'
type DepartmentNode = IamDepartment & { children: DepartmentNode[] }

function buildTree(list: IamDepartment[]): DepartmentNode[] {
  const map = new Map<number, DepartmentNode>()
  list.forEach((item) => map.set(item.id, { ...item, children: [] }))
  const roots: DepartmentNode[] = []
  list
    .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.id - b.id)
    .forEach((item) => {
      const node = map.get(item.id)
      if (!node) return
      const pid = item.parent_id ?? null
      if (pid && map.has(pid)) map.get(pid)!.children.push(node)
      else roots.push(node)
    })
  return roots
}

function flattenTree(nodes: DepartmentNode[], out: IamDepartment[] = []): IamDepartment[] {
  nodes.forEach((node) => {
    out.push(node)
    flattenTree(node.children, out)
  })
  return out
}

function filterTree(nodes: DepartmentNode[], q: string): DepartmentNode[] {
  if (!q) return nodes
  const keyword = q.toLowerCase()
  const dfs = (node: DepartmentNode): DepartmentNode | null => {
    const selfHit = String(node.name || '').toLowerCase().includes(keyword) || String(node.code || '').toLowerCase().includes(keyword)
    const children = node.children.map(dfs).filter((v): v is DepartmentNode => Boolean(v))
    if (selfHit || children.length) return { ...node, children }
    return null
  }
  return nodes.map(dfs).filter((v): v is DepartmentNode => Boolean(v))
}

function memberName(item: IamMember): string {
  return item.display_name?.trim() || item.username?.trim() || '-'
}

export default function IamMembersPage() {
  const [mainTab, setMainTab] = useState<MainTab>('departments')
  const [permissionTab, setPermissionTab] = useState<PermissionTab>('permission')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [tenants, setTenants] = useState<IamTenant[]>([])
  const [tenantUuid, setTenantUuid] = useState('')
  const [departments, setDepartments] = useState<IamDepartment[]>([])
  const [members, setMembers] = useState<IamMember[]>([])
  const [roles, setRoles] = useState<IamRole[]>([])
  const [permissions, setPermissions] = useState<IamPermission[]>([])

  const [activeDeptId, setActiveDeptId] = useState<number | null>(null)
  const [expanded, setExpanded] = useState<Set<number>>(new Set())

  const [deptSearch, setDeptSearch] = useState('')
  const [tenantSearch, setTenantSearch] = useState('')
  const [tenantStatusFilter, setTenantStatusFilter] = useState('')
  const [tenantPlanFilter, setTenantPlanFilter] = useState('')
  const [tenantPage, setTenantPage] = useState(1)
  const [tenantPageSize, setTenantPageSize] = useState(5)
  const [permissionSearch, setPermissionSearch] = useState('')
  const [permissionCategory, setPermissionCategory] = useState('')
  const [permissionDetailOpen, setPermissionDetailOpen] = useState(false)
  const [activePermission, setActivePermission] = useState<IamPermission | null>(null)
  const [permissionName, setPermissionName] = useState('')
  const [permissionCode, setPermissionCode] = useState('')
  const [permissionCategoryValue, setPermissionCategoryValue] = useState('')
  const [permissionStatus, setPermissionStatus] = useState('active')
  const [permissionRisk, setPermissionRisk] = useState('')
  const [permissionScope, setPermissionScope] = useState('')
  const [permissionWeight, setPermissionWeight] = useState('100')
  const [permissionTags, setPermissionTags] = useState('')
  const [permissionDescription, setPermissionDescription] = useState('')
  const [roleSearch, setRoleSearch] = useState('')

  const [createDeptOpen, setCreateDeptOpen] = useState(false)
  const [creatingDept, setCreatingDept] = useState(false)
  const [deptName, setDeptName] = useState('')
  const [deptKey, setDeptKey] = useState('')
  const [deptParent, setDeptParent] = useState('')
  const [deptSort, setDeptSort] = useState('')
  const [deptLeader, setDeptLeader] = useState('')
  const [deptStatus, setDeptStatus] = useState('1')
  const [deptMeta, setDeptMeta] = useState('')

  const [creatingTenant, setCreatingTenant] = useState(false)
  const [createTenantOpen, setCreateTenantOpen] = useState(false)
  const [newTenantKey, setNewTenantKey] = useState('')
  const [newTenantName, setNewTenantName] = useState('')
  const [newTenantPlan, setNewTenantPlan] = useState('free')
  const [newTenantStatus, setNewTenantStatus] = useState('active')
  const [movingDeptId, setMovingDeptId] = useState<number | null>(null)

  const [editDeptOpen, setEditDeptOpen] = useState(false)
  const [editingDeptId, setEditingDeptId] = useState<number | null>(null)
  const [editingDeptName, setEditingDeptName] = useState('')
  const [editingDeptKey, setEditingDeptKey] = useState('')
  const [editingDeptParent, setEditingDeptParent] = useState('')
  const [editingDeptSort, setEditingDeptSort] = useState('')
  const [editingDeptLeader, setEditingDeptLeader] = useState('')
  const [editingDeptStatus, setEditingDeptStatus] = useState('1')
  const [editingDeptMeta, setEditingDeptMeta] = useState('')
  const [editingDeptSaving, setEditingDeptSaving] = useState(false)

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const tenantPayload = await listIamTenants()
      const nextTenants = tenantPayload.list || []
      setTenants(nextTenants)
      const tid = nextTenants[0]?.uuid || nextTenants[0]?.key || ''
      setTenantUuid(tid)

      const [departmentPayload, memberPayload, rolePayload, permissionPayload] = await Promise.all([
        tid ? listIamDepartments(tid) : Promise.resolve({ list: [], total: 0 }),
        listIamMembers(),
        listIamRoles(),
        listIamPermissions(),
      ])

      setDepartments(departmentPayload.list || [])
      setMembers(memberPayload.list || [])
      setRoles(rolePayload.list || [])
      setPermissions(permissionPayload.list || [])

      const firstRoot = (departmentPayload.list || []).find((item) => !item.parent_id)
      const nextActive = firstRoot?.id ?? departmentPayload.list?.[0]?.id ?? null
      setActiveDeptId(nextActive)
      if (nextActive) {
        setExpanded((prev) => new Set<number>([...Array.from(prev), nextActive]))
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载成员管理失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  const tree = useMemo(() => buildTree(departments), [departments])
  const filteredTree = useMemo(() => filterTree(tree, deptSearch.trim()), [deptSearch, tree])
  const flatDepartments = useMemo(() => flattenTree(filteredTree), [filteredTree])
  const activeDepartment = useMemo(() => departments.find((d) => d.id === activeDeptId) || null, [activeDeptId, departments])
  const activeChildren = useMemo(() => {
    if (!activeDeptId) return flatDepartments.filter((d) => !d.parent_id)
    return flatDepartments.filter((d) => (d.parent_id ?? null) === activeDeptId)
  }, [activeDeptId, flatDepartments])

  const filteredTenants = useMemo(() => {
    return tenants.filter((item) => {
      const q = tenantSearch.trim().toLowerCase()
      const hitQ = !q || String(item.name || '').toLowerCase().includes(q) || String(item.key || '').toLowerCase().includes(q)
      const hitStatus = !tenantStatusFilter || item.status === tenantStatusFilter
      const hitPlan = !tenantPlanFilter || item.plan === tenantPlanFilter
      return hitQ && hitStatus && hitPlan
    })
  }, [tenantPlanFilter, tenantSearch, tenantStatusFilter, tenants])
  const tenantTotalPages = useMemo(() => Math.max(1, Math.ceil(filteredTenants.length / Math.max(1, tenantPageSize))), [filteredTenants.length, tenantPageSize])
  const pagedTenants = useMemo(() => {
    const start = (tenantPage - 1) * tenantPageSize
    return filteredTenants.slice(start, start + tenantPageSize)
  }, [filteredTenants, tenantPage, tenantPageSize])
  const tenantPageNumbers = useMemo(() => {
    if (tenantTotalPages <= 7) return Array.from({ length: tenantTotalPages }, (_, i) => i + 1)
    if (tenantPage <= 4) return [1, 2, 3, 4, 5, tenantTotalPages]
    if (tenantPage >= tenantTotalPages - 3) return [1, tenantTotalPages - 4, tenantTotalPages - 3, tenantTotalPages - 2, tenantTotalPages - 1, tenantTotalPages]
    return [1, tenantPage - 1, tenantPage, tenantPage + 1, tenantTotalPages]
  }, [tenantPage, tenantTotalPages])
  const tenantRange = useMemo(() => {
    if (!filteredTenants.length) return { start: 0, end: 0 }
    const start = (tenantPage - 1) * tenantPageSize + 1
    const end = Math.min(tenantPage * tenantPageSize, filteredTenants.length)
    return { start, end }
  }, [filteredTenants.length, tenantPage, tenantPageSize])

  const permissionCategories = useMemo(() => {
    const set = new Set<string>()
    permissions.forEach((item) => {
      const head = String(item.resource || '').split('.')[0] || '-'
      set.add(head)
    })
    return Array.from(set)
  }, [permissions])

  const filteredPermissions = useMemo(() => {
    const q = permissionSearch.trim().toLowerCase()
    return permissions.filter((item) => {
      const category = String(item.resource || '').split('.')[0] || '-'
      const hitCategory = !permissionCategory || permissionCategory === category
      const text = `${item.resource || ''} ${item.action || ''} ${item.description || ''}`.toLowerCase()
      const hitQ = !q || text.includes(q)
      return hitCategory && hitQ
    })
  }, [permissionCategory, permissionSearch, permissions])

  const filteredRoles = useMemo(() => {
    const q = roleSearch.trim().toLowerCase()
    if (!q) return roles
    return roles.filter((item) => String(item.code || '').toLowerCase().includes(q) || String(item.name || '').toLowerCase().includes(q))
  }, [roleSearch, roles])

  const toggleExpand = (id: number) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  useEffect(() => {
    setTenantPage(1)
  }, [tenantSearch, tenantStatusFilter, tenantPlanFilter, tenantPageSize])

  useEffect(() => {
    if (tenantPage > tenantTotalPages) {
      setTenantPage(tenantTotalPages)
    }
  }, [tenantPage, tenantTotalPages])

  const createDepartment = async () => {
    if (!tenantUuid || !deptName.trim()) return
    setCreatingDept(true)
    try {
      await createIamDepartment({
        tenant_uuid: tenantUuid,
        name: deptName.trim(),
        parent_id: deptParent ? Number(deptParent) : undefined,
        code: deptKey.trim() || undefined,
        sort_order: deptSort.trim() ? Number(deptSort) : undefined,
        description: deptMeta.trim() || undefined,
      })
      setCreateDeptOpen(false)
      setDeptName('')
      setDeptKey('')
      setDeptParent('')
      setDeptSort('')
      setDeptLeader('')
      setDeptStatus('1')
      setDeptMeta('')
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '新增部门失败')
    } finally {
      setCreatingDept(false)
    }
  }

  const openEditDepartmentModal = (dept: IamDepartment) => {
    setEditingDeptId(dept.id)
    setEditingDeptName(dept.name || '')
    setEditingDeptKey(dept.code || '')
    setEditingDeptParent(dept.parent_id ? String(dept.parent_id) : '')
    setEditingDeptSort(String(dept.sort_order ?? 0))
    setEditingDeptLeader('')
    setEditingDeptStatus('1')
    setEditingDeptMeta('')
    setEditDeptOpen(true)
  }

  const saveEditedDepartment = async () => {
    if (!editingDeptId || !editingDeptName.trim()) return
    setEditingDeptSaving(true)
    try {
      await updateIamDepartment(editingDeptId, {
        name: editingDeptName.trim(),
        parent_id: editingDeptParent ? Number(editingDeptParent) : 0,
        sort_order: editingDeptSort.trim() ? Number(editingDeptSort) : 0,
        description: editingDeptMeta.trim() || undefined,
      })
      setEditDeptOpen(false)
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '编辑部门失败')
    } finally {
      setEditingDeptSaving(false)
    }
  }

  const reorderDepartment = async (dept: IamDepartment, direction: 'up' | 'down') => {
    const siblings = departments
      .filter((item) => (item.parent_id ?? null) === (dept.parent_id ?? null))
      .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.id - b.id)
    const index = siblings.findIndex((item) => item.id === dept.id)
    if (index < 0) return
    const swapIndex = direction === 'up' ? index - 1 : index + 1
    if (swapIndex < 0 || swapIndex >= siblings.length) return
    const target = siblings[swapIndex]

    setMovingDeptId(dept.id)
    try {
      const reordered = [...siblings]
      const [removed] = reordered.splice(index, 1)
      reordered.splice(swapIndex, 0, removed)
      await Promise.all(
        reordered.map((item, idx) =>
          updateIamDepartment(item.id, { sort_order: idx + 1 }),
        ),
      )
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '部门排序调整失败')
    } finally {
      setMovingDeptId(null)
    }
  }

  const openCreateDepartmentModal = () => {
    const fallbackParentId =
      activeDeptId ??
      departments.find((item) => !item.parent_id)?.id ??
      null
    setDeptParent(fallbackParentId ? String(fallbackParentId) : '')
    setCreateDeptOpen(true)
  }

  const createTenant = async () => {
    const key = newTenantKey.trim().toLowerCase()
    const name = newTenantName.trim()
    if (!key || !name) return
    setCreatingTenant(true)
    try {
      await createIamTenant({
        key,
        name,
        status: newTenantStatus,
        plan: newTenantPlan,
      })
      setCreateTenantOpen(false)
      setNewTenantKey('')
      setNewTenantName('')
      setNewTenantPlan('free')
      setNewTenantStatus('active')
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '新增租户失败')
    } finally {
      setCreatingTenant(false)
    }
  }

  const openPermissionDetail = (permission: IamPermission) => {
    const resource = String(permission.resource || '')
    const action = String(permission.action || '')
    setPermissionName(`${resource || '*'}:${action || '*'}`)
    setPermissionCode(`${resource || '*'}${action ? `.${action}` : ''}`)
    setPermissionCategoryValue(resource.split('.')[0] || '*')
    setPermissionStatus('active')
    setPermissionRisk('')
    setPermissionScope('')
    setPermissionWeight('100')
    setPermissionTags('')
    setPermissionDescription(permission.description || '')
    setActivePermission(permission)
    setPermissionDetailOpen(true)
  }

  const savePermissionDetail = () => {
    setError('当前后端仅开放权限读取接口，暂不支持保存编辑')
  }

  const renderNode = (node: DepartmentNode, depth = 0): JSX.Element => {
    const hasChildren = node.children.length > 0
    const isExpanded = expanded.has(node.id)
    const isActive = node.id === activeDeptId

    return (
      <div key={node.id}>
        <button
          type="button"
          onClick={() => setActiveDeptId(node.id)}
          style={{
            width: '100%',
            height: 34,
            border: '1px solid transparent',
            borderRadius: 8,
            background: isActive ? '#dff4e8' : 'transparent',
            textAlign: 'left',
            color: '#374151',
            padding: `0 8px 0 ${8 + depth * 16}px`,
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            cursor: 'pointer',
          }}
        >
          {hasChildren ? (
            <span
              style={{ width: 14, color: '#6b7280' }}
              onClick={(event) => {
                event.stopPropagation()
                toggleExpand(node.id)
              }}
            >
              {isExpanded ? '▾' : '▸'}
            </span>
          ) : (
            <span style={{ width: 14 }} />
          )}
          <span>{node.name}</span>
        </button>
        {hasChildren && isExpanded ? (
          <div style={{ display: 'grid', gap: 2 }}>
            {node.children.map((item) => renderNode(item, depth + 1))}
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <main className="px-admin-page" data-testid="iam-members-page">
      <section className="px-admin-shell">
        <section className="px-cap-hero">
          <h1 className="px-cap-page-title">用户管理</h1>
          <p className="px-cap-page-desc">管理组织、成员与权限。</p>
        </section>

        <article className="px-admin-card">
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,minmax(0,1fr))', gap: 8, border: '1px solid #e5e7eb', borderRadius: 12, padding: 4, background: '#f8fafc' }}>
            {[
              { key: 'departments' as MainTab, label: '部门' },
              { key: 'users' as MainTab, label: '用户' },
              { key: 'permissions' as MainTab, label: '权限' },
            ].map((item) => (
              <button
                key={item.key}
                type="button"
                onClick={() => setMainTab(item.key)}
                style={{
                  height: 34,
                  borderRadius: 10,
                  border: '1px solid transparent',
                  background: mainTab === item.key ? '#ffffff' : 'transparent',
                  color: mainTab === item.key ? '#16a34a' : '#6b7280',
                  fontWeight: 700,
                  cursor: 'pointer',
                }}
              >
                {item.label}
              </button>
            ))}
          </div>

          {error ? <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>{error}</p> : null}

          {mainTab === 'departments' ? (
            <section style={{ marginTop: 16 }}>
              <div className="px-cap-hero-head">
                <div>
                  <h2 className="px-admin-card-title">部门管理</h2>
                  <p className="px-admin-card-text">维护组织层级与负责人。</p>
                </div>
                <button type="button" className="px-btn" onClick={openCreateDepartmentModal}>新增部门</button>
              </div>

              <input className="px-field" style={{ marginTop: 12, maxWidth: 320 }} value={deptSearch} placeholder="搜索部门..." onChange={(e) => setDeptSearch(e.target.value)} />

              <div style={{ marginTop: 12, display: 'grid', gridTemplateColumns: '1fr 2fr', gap: 12 }}>
                <div style={{ border: '1px solid #e5e7eb', borderRadius: 10, padding: 12 }}>
                  <div style={{ fontWeight: 700, marginBottom: 10 }}>部门管理</div>
                  {loading ? <p className="px-admin-card-text">加载中...</p> : filteredTree.length === 0 ? <p className="px-admin-card-text">暂无部门</p> : <div style={{ display: 'grid', gap: 2 }}>{filteredTree.map((node) => renderNode(node))}</div>}
                </div>
                <div style={{ border: '1px solid #e5e7eb', borderRadius: 10, padding: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 10 }}>
                    <div style={{ fontWeight: 700 }}>{(activeDepartment?.name || 'General') + ' - 部门管理'}</div>
                    <div className="px-admin-card-text">名称</div>
                  </div>
                  <div className="px-table-wrap">
                    <table className="px-table">
                      <thead>
                        <tr><th>名称</th><th>ID</th><th>上级部门</th><th>排序</th><th>负责人</th><th>操作</th></tr>
                      </thead>
                      <tbody>
                        {activeChildren.length === 0 ? (
                          <tr><td colSpan={6} className="px-admin-card-text">暂无数据</td></tr>
                        ) : activeChildren.map((item) => (
                          <tr key={item.id}>
                            <td>{item.name}</td>
                            <td>{item.id}</td>
                            <td>{departments.find((d) => d.id === item.parent_id)?.name || '-'}</td>
                            <td>{item.sort_order ?? 0}</td>
                            <td>{members.find((m) => String(m.id) === deptLeader)?.display_name || '-'}</td>
                            <td>
                              <div style={{ display: 'flex', gap: 10, fontSize: 13 }}>
                                <button
                                  type="button"
                                  style={{ color: '#22c55e', border: 0, background: 'transparent', cursor: 'pointer' }}
                                  disabled={movingDeptId === item.id}
                                  onClick={() => void reorderDepartment(item, 'up')}
                                >
                                  上移
                                </button>
                                <button
                                  type="button"
                                  style={{ color: '#22c55e', border: 0, background: 'transparent', cursor: 'pointer' }}
                                  disabled={movingDeptId === item.id}
                                  onClick={() => void reorderDepartment(item, 'down')}
                                >
                                  下移
                                </button>
                                <button
                                  type="button"
                                  style={{ color: '#22c55e', border: 0, background: 'transparent', cursor: 'pointer' }}
                                  onClick={() => openEditDepartmentModal(item)}
                                >
                                  编辑
                                </button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </section>
          ) : null}

          {mainTab === 'users' ? (
            <section style={{ marginTop: 16 }}>
              <div className="px-cap-hero-head">
                <div>
                  <h2 className="px-admin-card-title">租户目录 · 超级管理员视图</h2>
                  <p className="px-admin-card-text">选择租户以管理其用户和角色。</p>
                </div>
                <div className="px-admin-toolbar" style={{ marginTop: 0 }}>
                  <button type="button" className="px-btn" onClick={() => setCreateTenantOpen(true)}>
                    新增租户
                  </button>
                  <button type="button" className="px-btn-ghost" onClick={() => void loadAll()}>重新加载</button>
                </div>
              </div>

              <div style={{ marginTop: 12, display: 'grid', gridTemplateColumns: '1.6fr 1fr 1fr auto', gap: 10 }}>
                <input className="px-field" value={tenantSearch} placeholder="搜索租户..." onChange={(e) => setTenantSearch(e.target.value)} />
                <select className="px-select" value={tenantStatusFilter} onChange={(e) => setTenantStatusFilter(e.target.value)}>
                  <option value="">全部状态</option>
                  <option value="active">启用</option>
                  <option value="suspended">停用</option>
                </select>
                <select className="px-select" value={tenantPlanFilter} onChange={(e) => setTenantPlanFilter(e.target.value)}>
                  <option value="">全部套餐</option>
                  <option value="free">free</option>
                  <option value="standard">standard</option>
                  <option value="premium">premium</option>
                </select>
                <button
                  type="button"
                  className="px-btn-ghost"
                  onClick={() => {
                    setTenantSearch('')
                    setTenantStatusFilter('')
                    setTenantPlanFilter('')
                    setTenantPage(1)
                  }}
                >
                  重置
                </button>
              </div>

              <div style={{ marginTop: 12, display: 'grid', gap: 8 }}>
                {pagedTenants.map((item) => (
                  <div key={item.id} style={{ border: '1px solid #e5e7eb', borderRadius: 10, padding: '12px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 16 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                      <div style={{ width: 40, height: 40, borderRadius: 10, background: '#dbeafe', display: 'grid', placeItems: 'center', color: '#2563eb', fontSize: 18 }}>
                        🏢
                      </div>
                      <div>
                        <div style={{ fontSize: 18, fontWeight: 700, lineHeight: 1.25 }}>{item.name}</div>
                        <div className="px-admin-card-text" style={{ marginTop: 4 }}>未配置域名</div>
                      </div>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(72px,auto))', alignItems: 'center', gap: 18 }}>
                      <div style={{ textAlign: 'right', minWidth: 72 }}>
                        <div style={{ fontSize: 24, fontWeight: 700, lineHeight: 1 }}>{item.user_count ?? item.member_count ?? 0}</div>
                        <div className="px-admin-card-text" style={{ marginTop: 2 }}>用户数</div>
                      </div>
                      <div style={{ textAlign: 'right', minWidth: 72 }}>
                        <span className="px-cap-tag" style={{ background: '#e2e8f0', borderColor: '#cbd5e1', color: '#475569' }}>{item.status === 'active' ? '启用' : '停用'}</span>
                        <div className="px-admin-card-text" style={{ marginTop: 4 }}>{item.plan || '基础版'}</div>
                      </div>
                      <div style={{ textAlign: 'right', minWidth: 96 }}>
                        <div style={{ fontWeight: 600, color: '#64748b', whiteSpace: 'nowrap' }}>{item.created_at ? item.created_at.slice(0, 10).replace(/-/g, '/') : '-'}</div>
                        <div className="px-admin-card-text" style={{ marginTop: 2 }}>创建时间</div>
                      </div>
                    </div>
                  </div>
                ))}
                {filteredTenants.length === 0 ? <p className="px-admin-card-text">暂无租户</p> : null}
              </div>
              {filteredTenants.length > 0 ? (
                <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
                  <div className="px-admin-card-text">
                    显示 {tenantRange.start}-{tenantRange.end} / {filteredTenants.length}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <select className="px-select" value={String(tenantPageSize)} onChange={(e) => setTenantPageSize(Number(e.target.value) || 5)} style={{ width: 90 }}>
                      <option value="5">5 / 页</option>
                      <option value="10">10 / 页</option>
                      <option value="20">20 / 页</option>
                    </select>
                    <button type="button" className="px-btn-ghost" disabled={tenantPage <= 1} onClick={() => setTenantPage((p) => Math.max(1, p - 1))}>上一页</button>
                    {tenantPageNumbers.map((p) => (
                      <button
                        key={p}
                        type="button"
                        className={p === tenantPage ? 'px-btn' : 'px-btn-ghost'}
                        onClick={() => setTenantPage(p)}
                      >
                        {p}
                      </button>
                    ))}
                    <button type="button" className="px-btn-ghost" disabled={tenantPage >= tenantTotalPages} onClick={() => setTenantPage((p) => Math.min(tenantTotalPages, p + 1))}>下一页</button>
                  </div>
                </div>
              ) : null}
            </section>
          ) : null}

          {mainTab === 'permissions' ? (
            <section style={{ marginTop: 16 }}>
              <div style={{ display: 'flex', gap: 16, borderBottom: '1px solid #e5e7eb', paddingBottom: 8 }}>
                <button type="button" onClick={() => setPermissionTab('permission')} style={{ border: 0, background: 'transparent', color: permissionTab === 'permission' ? '#16a34a' : '#374151', fontWeight: 700, cursor: 'pointer' }}>权限管理</button>
                <button type="button" onClick={() => setPermissionTab('role')} style={{ border: 0, background: 'transparent', color: permissionTab === 'role' ? '#16a34a' : '#374151', fontWeight: 700, cursor: 'pointer' }}>角色配置</button>
              </div>

              {permissionTab === 'permission' ? (
                <div style={{ marginTop: 14 }}>
                  <div className="px-cap-hero-head">
                    <div>
                      <h2 className="px-admin-card-title">权限管理</h2>
                      <p className="px-admin-card-text">管理系统权限和访问控制</p>
                    </div>
                    <button type="button" className="px-btn">新增权限</button>
                  </div>
                  <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1.6fr 0.6fr', gap: 10 }}>
                    <input className="px-field" value={permissionSearch} placeholder="搜索名称、代码、标识或描述..." onChange={(e) => setPermissionSearch(e.target.value)} />
                    <select className="px-select" value={permissionCategory} onChange={(e) => setPermissionCategory(e.target.value)}>
                      <option value="">全部分类</option>
                      {permissionCategories.map((item) => <option key={item} value={item}>{item}</option>)}
                    </select>
                  </div>
                  <div className="px-table-wrap" style={{ marginTop: 10 }}>
                    <table className="px-table">
                      <thead><tr><th>标识</th><th>描述</th><th>状态</th><th>分类</th><th>创建时间</th><th>操作</th></tr></thead>
                      <tbody>
                        {filteredPermissions.length === 0 ? <tr><td colSpan={6} className="px-admin-card-text">暂无权限数据</td></tr> : filteredPermissions.map((item) => {
                          const category = String(item.resource || '').split('.')[0] || '-'
                          const code = `${item.resource}:${item.action}`
                          return (
                            <tr key={item.id}>
                              <td>
                                <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                                  <span className="px-cap-tag">{category}</span>
                                  <span className="px-cap-tag">{String(item.resource || '').split('.').slice(-1)[0] || '-'}</span>
                                  <span className="px-cap-tag">{item.action || '-'}</span>
                                </div>
                                <div className="px-admin-card-text" style={{ marginTop: 4 }}>{code}</div>
                              </td>
                              <td>{item.description || '-'}</td>
                              <td>生效</td>
                              <td>{category}</td>
                              <td>-</td>
                              <td>
                                <button
                                  type="button"
                                  className="px-btn-ghost"
                                  onClick={() => openPermissionDetail(item)}
                                >
                                  编辑
                                </button>
                              </td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              ) : (
                <div style={{ marginTop: 14 }}>
                  <div className="px-cap-hero-head">
                    <div>
                      <h2 className="px-admin-card-title">角色配置</h2>
                      <p className="px-admin-card-text">管理角色与权限主体映射。</p>
                    </div>
                    <input className="px-field" style={{ minWidth: 280 }} value={roleSearch} placeholder="搜索角色名或编码" onChange={(e) => setRoleSearch(e.target.value)} />
                  </div>
                  <div className="px-table-wrap" style={{ marginTop: 10 }}>
                    <table className="px-table">
                      <thead><tr><th>角色编码</th><th>角色名称</th><th>描述</th></tr></thead>
                      <tbody>
                        {filteredRoles.length === 0 ? <tr><td colSpan={3} className="px-admin-card-text">暂无角色数据</td></tr> : filteredRoles.map((item) => (
                          <tr key={item.id}><td>{item.code || '-'}</td><td>{item.name || '-'}</td><td>{item.description || '-'}</td></tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </section>
          ) : null}
        </article>

        {createDeptOpen ? (
          <div role="dialog" aria-modal="true" style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }} onClick={() => setCreateDeptOpen(false)}>
            <article className="px-admin-card" style={{ width: 'min(760px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <h3 className="px-admin-card-title">新增部门</h3>
              <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                <label><div className="px-admin-card-text">名称 *</div><input className="px-field" value={deptName} onChange={(e) => setDeptName(e.target.value)} /></label>
                <label><div className="px-admin-card-text">上级部门</div><select className="px-select" value={deptParent} onChange={(e) => setDeptParent(e.target.value)}><option value="">无上级</option>{departments.map((d) => <option key={d.id} value={String(d.id)}>{d.name}</option>)}</select></label>
                <label><div className="px-admin-card-text">唯一键 Key</div><input className="px-field" value={deptKey} onChange={(e) => setDeptKey(e.target.value)} placeholder="英文/短横线/下划线" /></label>
                <label><div className="px-admin-card-text">排序</div><input className="px-field" value={deptSort} onChange={(e) => setDeptSort(e.target.value)} placeholder="数字越小越靠前" /></label>
                <label><div className="px-admin-card-text">负责人</div><select className="px-select" value={deptLeader} onChange={(e) => setDeptLeader(e.target.value)}><option value="">无</option>{members.map((m) => <option key={m.id} value={String(m.id)}>{memberName(m)}</option>)}</select></label>
                <label><div className="px-admin-card-text">状态</div><div style={{ display: 'grid', gap: 6, marginTop: 8 }}><label><input type="radio" checked={deptStatus === '1'} onChange={() => setDeptStatus('1')} /> 启用</label><label><input type="radio" checked={deptStatus === '0'} onChange={() => setDeptStatus('0')} /> 停用</label></div></label>
                <label style={{ gridColumn: '1 / span 2' }}><div className="px-admin-card-text">扩展 Meta(JSON)</div><textarea value={deptMeta} onChange={(e) => setDeptMeta(e.target.value)} style={{ width: '100%', minHeight: 96, borderRadius: 10, border: '1px solid #cbd5e1', padding: 10, fontSize: 14 }} /></label>
              </div>
              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setCreateDeptOpen(false)}>取消</button>
                <button type="button" className="px-btn" disabled={creatingDept || !deptName.trim()} onClick={() => void createDepartment()}>{creatingDept ? '保存中' : '保存'}</button>
              </div>
            </article>
          </div>
        ) : null}

        {editDeptOpen ? (
          <div role="dialog" aria-modal="true" style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }} onClick={() => setEditDeptOpen(false)}>
            <article className="px-admin-card" style={{ width: 'min(760px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <h3 className="px-admin-card-title">编辑部门</h3>
              <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                <label><div className="px-admin-card-text">名称 *</div><input className="px-field" value={editingDeptName} onChange={(e) => setEditingDeptName(e.target.value)} /></label>
                <label>
                  <div className="px-admin-card-text">上级部门</div>
                  <select className="px-select" value={editingDeptParent} onChange={(e) => setEditingDeptParent(e.target.value)}>
                    <option value="">无上级</option>
                    {departments.filter((d) => d.id !== editingDeptId).map((d) => <option key={d.id} value={String(d.id)}>{d.name}</option>)}
                  </select>
                </label>
                <label><div className="px-admin-card-text">唯一键 Key</div><input className="px-field" value={editingDeptKey} onChange={(e) => setEditingDeptKey(e.target.value)} placeholder="英文/短横线/下划线" /></label>
                <label><div className="px-admin-card-text">排序</div><input className="px-field" value={editingDeptSort} onChange={(e) => setEditingDeptSort(e.target.value)} placeholder="数字越小越靠前" /></label>
                <label>
                  <div className="px-admin-card-text">负责人</div>
                  <select className="px-select" value={editingDeptLeader} onChange={(e) => setEditingDeptLeader(e.target.value)}>
                    <option value="">无</option>
                    {members.map((m) => <option key={m.id} value={String(m.id)}>{memberName(m)}</option>)}
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">状态</div>
                  <div style={{ display: 'grid', gap: 6, marginTop: 8 }}>
                    <label><input type="radio" checked={editingDeptStatus === '1'} onChange={() => setEditingDeptStatus('1')} /> 启用</label>
                    <label><input type="radio" checked={editingDeptStatus === '0'} onChange={() => setEditingDeptStatus('0')} /> 停用</label>
                  </div>
                </label>
                <label style={{ gridColumn: '1 / span 2' }}>
                  <div className="px-admin-card-text">扩展 Meta(JSON)</div>
                  <textarea value={editingDeptMeta} onChange={(e) => setEditingDeptMeta(e.target.value)} style={{ width: '100%', minHeight: 96, borderRadius: 10, border: '1px solid #cbd5e1', padding: 10, fontSize: 14 }} />
                </label>
              </div>
              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setEditDeptOpen(false)}>取消</button>
                <button type="button" className="px-btn" disabled={editingDeptSaving || !editingDeptName.trim()} onClick={() => void saveEditedDepartment()}>{editingDeptSaving ? '保存中' : '保存'}</button>
              </div>
            </article>
          </div>
        ) : null}

        {createTenantOpen ? (
          <div role="dialog" aria-modal="true" style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }} onClick={() => setCreateTenantOpen(false)}>
            <article className="px-admin-card" style={{ width: 'min(620px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <h3 className="px-admin-card-title">新增租户</h3>
              <p className="px-admin-card-text">填写后点击保存才会创建。</p>
              <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                <label>
                  <div className="px-admin-card-text">租户 Key *</div>
                  <input className="px-field" value={newTenantKey} placeholder="tenant-demo" onChange={(e) => setNewTenantKey(e.target.value)} />
                </label>
                <label>
                  <div className="px-admin-card-text">租户名称 *</div>
                  <input className="px-field" value={newTenantName} placeholder="tenant-demo" onChange={(e) => setNewTenantName(e.target.value)} />
                </label>
                <label>
                  <div className="px-admin-card-text">套餐</div>
                  <select className="px-select" value={newTenantPlan} onChange={(e) => setNewTenantPlan(e.target.value)}>
                    <option value="free">free</option>
                    <option value="standard">standard</option>
                    <option value="premium">premium</option>
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">状态</div>
                  <select className="px-select" value={newTenantStatus} onChange={(e) => setNewTenantStatus(e.target.value)}>
                    <option value="active">启用</option>
                    <option value="suspended">停用</option>
                  </select>
                </label>
              </div>
              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setCreateTenantOpen(false)}>取消</button>
                <button type="button" className="px-btn" disabled={creatingTenant || !newTenantKey.trim() || !newTenantName.trim()} onClick={() => void createTenant()}>
                  {creatingTenant ? '保存中' : '保存'}
                </button>
              </div>
            </article>
          </div>
        ) : null}

        {permissionDetailOpen && activePermission ? (
          <div
            role="dialog"
            aria-modal="true"
            style={{ position: 'fixed', inset: 0, background: 'rgba(2,6,23,.55)', display: 'grid', placeItems: 'center', zIndex: 1300, padding: 16 }}
            onClick={() => setPermissionDetailOpen(false)}
          >
            <article className="px-admin-card" style={{ width: 'min(760px,100%)' }} onClick={(e) => e.stopPropagation()}>
              <h3 className="px-admin-card-title">编辑权限</h3>
              <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                <label>
                  <div className="px-admin-card-text">权限名称</div>
                  <input className="px-field" value={permissionName} onChange={(e) => setPermissionName(e.target.value)} />
                </label>
                <label>
                  <div className="px-admin-card-text">权限代码</div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 8, alignItems: 'center' }}>
                    <input className="px-field" value={permissionCode} readOnly />
                    <button
                      type="button"
                      className="px-btn-ghost"
                      onClick={() => void navigator.clipboard?.writeText(permissionCode)}
                      title="复制权限代码"
                    >
                      复制
                    </button>
                  </div>
                  <div className="px-admin-card-text" style={{ marginTop: 6 }}>创建后不可修改</div>
                </label>
                <label>
                  <div className="px-admin-card-text">分类</div>
                  <select className="px-select" value={permissionCategoryValue} onChange={(e) => setPermissionCategoryValue(e.target.value)}>
                    <option value="*">*</option>
                    <option value="iam">iam</option>
                    <option value="system">system</option>
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">状态</div>
                  <select className="px-select" value={permissionStatus} onChange={(e) => setPermissionStatus(e.target.value)}>
                    <option value="active">生效</option>
                    <option value="disabled">停用</option>
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">风险等级</div>
                  <input className="px-field" value={permissionRisk} onChange={(e) => setPermissionRisk(e.target.value)} />
                </label>
                <label>
                  <div className="px-admin-card-text">作用域</div>
                  <input className="px-field" value={permissionScope} onChange={(e) => setPermissionScope(e.target.value)} />
                </label>
                <label>
                  <div className="px-admin-card-text">排序权重</div>
                  <input className="px-field" value={permissionWeight} onChange={(e) => setPermissionWeight(e.target.value)} />
                  <div className="px-admin-card-text" style={{ marginTop: 6 }}>数值越小优先级越高</div>
                </label>
                <label>
                  <div className="px-admin-card-text">标签</div>
                  <input className="px-field" value={permissionTags} onChange={(e) => setPermissionTags(e.target.value)} />
                </label>
                <label style={{ gridColumn: '1 / span 2' }}>
                  <div className="px-admin-card-text">描述</div>
                  <textarea
                    style={{ width: '100%', minHeight: 96, borderRadius: 10, border: '1px solid #cbd5e1', padding: 10, fontSize: 14 }}
                    value={permissionDescription}
                    onChange={(e) => setPermissionDescription(e.target.value)}
                  />
                </label>
              </div>
              <div className="px-admin-toolbar" style={{ justifyContent: 'flex-end', marginTop: 14 }}>
                <button type="button" className="px-btn-ghost" onClick={() => setPermissionDetailOpen(false)}>取消</button>
                <button type="button" className="px-btn" onClick={savePermissionDetail}>
                  保存更改
                </button>
              </div>
            </article>
          </div>
        ) : null}
      </section>
    </main>
  )
}
