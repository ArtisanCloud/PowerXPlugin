'use client'

import Link from 'next/link'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { createIamTenant, listIamTenants, type IamTenant, updateIamTenant } from '@/lib/api/iam'
import { tAdmin } from '@/lib/i18n/admin'
import { type AdminLocale, useLocalePreference } from '@/lib/ui/preferences'

type MetricItem = {
  key: string
  label: string
  value: string
}

type SettingCategory = {
  key: string
  title: string
  description: string
  icon: string
  iconBg: string
  path: string
}

function toLabel(key: string): string {
  return String(key || '').trim().toLowerCase()
}

function metricLabel(locale: AdminLocale, key: string): string {
  const normalized = toLabel(key)
  if (['tenants', 'tenant_count'].includes(normalized)) return tAdmin(locale, 'iam.overview.metrics.tenants')
  if (['members', 'member_count'].includes(normalized)) return tAdmin(locale, 'iam.overview.metrics.members')
  if (['roles', 'role_count'].includes(normalized)) return tAdmin(locale, 'iam.overview.metrics.roles')
  if (['departments', 'department_count'].includes(normalized)) return tAdmin(locale, 'iam.overview.metrics.departments')
  if (['delegated', 'delegated_iam'].includes(normalized)) return tAdmin(locale, 'iam.overview.metrics.delegated')
  return key
}

function normalizeMetrics(locale: AdminLocale, payload: Record<string, unknown> | null): MetricItem[] {
  if (!payload || typeof payload !== 'object') return []
  const entries = Object.entries(payload)
  const items: MetricItem[] = []
  for (const [key, value] of entries) {
    if (['list', 'items', 'data'].includes(key)) continue
    if (typeof value === 'number') {
      items.push({ key, label: metricLabel(locale, key), value: String(value) })
      continue
    }
    if (typeof value === 'boolean') {
      items.push({ key, label: metricLabel(locale, key), value: value ? tAdmin(locale, 'common.yes') : tAdmin(locale, 'common.no') })
      continue
    }
    if (typeof value === 'string' && value.trim()) {
      items.push({ key, label: metricLabel(locale, key), value })
    }
  }
  return items
}

function statusLabel(status: string): string {
  return String(status || '').toLowerCase() === 'suspended' ? 'suspended' : 'active'
}

function statusText(locale: AdminLocale, status: string): string {
  return statusLabel(status) === 'suspended'
    ? tAdmin(locale, 'iam.overview.status.suspended')
    : tAdmin(locale, 'iam.overview.status.active')
}

function statusBadgeStyle(status: string): { background: string; color: string } {
  if (String(status || '').toLowerCase() === 'suspended') {
    return { background: '#fef3c7', color: '#92400e' }
  }
  return { background: '#dcfce7', color: '#166534' }
}

export default function IamOverviewPage() {
  const locale = useLocalePreference()
  const [tenants, setTenants] = useState<IamTenant[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [planModalOpen, setPlanModalOpen] = useState(false)
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [planSaving, setPlanSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [updatingTenantId, setUpdatingTenantId] = useState<number | null>(null)

  const [selectedTenant, setSelectedTenant] = useState<IamTenant | null>(null)
  const [planForm, setPlanForm] = useState({ plan: 'free', name: '' })
  const [createForm, setCreateForm] = useState({ key: '', name: '', plan: 'free', status: 'active' })

  const metrics = useMemo(() => {
    const memberCount = tenants.reduce((acc, item) => {
      const member = Number(item.member_count ?? item.user_count ?? 0)
      return Number.isFinite(member) ? acc + member : acc
    }, 0)
    return normalizeMetrics(locale, {
      tenant_count: tenants.length,
      member_count: memberCount,
    })
  }, [locale, tenants])
  const settingCategories = useMemo<SettingCategory[]>(() => ([
    {
      key: 'members',
      title: tAdmin(locale, 'iam.overview.cards.members.title'),
      description: tAdmin(locale, 'iam.overview.cards.members.desc'),
      icon: '👥',
      iconBg: '#dbeafe',
      path: '/admin/iam/members',
    },
    {
      key: 'roles',
      title: tAdmin(locale, 'iam.overview.cards.roles.title'),
      description: tAdmin(locale, 'iam.overview.cards.roles.desc'),
      icon: '🛡️',
      iconBg: '#dcfce7',
      path: '/admin/iam/roles',
    },
    {
      key: 'settings',
      title: tAdmin(locale, 'iam.overview.cards.settings.title'),
      description: tAdmin(locale, 'iam.overview.cards.settings.desc'),
      icon: '⚙️',
      iconBg: '#f3e8ff',
      path: '/admin/iam/settings',
    },
  ]), [locale])

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const tenantList = await listIamTenants()
      setTenants(tenantList.list || [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : tAdmin(locale, 'iam.overview.error.load'))
    } finally {
      setLoading(false)
    }
  }, [locale])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  const handleChangeTenantStatus = async (tenant: IamTenant, nextStatus: string) => {
    try {
      setUpdatingTenantId(tenant.id)
      await updateIamTenant(tenant.id, { status: nextStatus })
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : tAdmin(locale, 'iam.overview.error.updateStatus'))
    } finally {
      setUpdatingTenantId(null)
    }
  }

  const openPlanModal = (tenant: IamTenant) => {
    setSelectedTenant(tenant)
    setPlanForm({
      plan: tenant.plan || 'free',
      name: tenant.name || '',
    })
    setPlanModalOpen(true)
  }

  const submitPlan = async () => {
    if (!selectedTenant) return
    setPlanSaving(true)
    try {
      await updateIamTenant(selectedTenant.id, {
        plan: planForm.plan,
        name: planForm.name,
      })
      setPlanModalOpen(false)
      setSelectedTenant(null)
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : tAdmin(locale, 'iam.overview.error.updatePlan'))
    } finally {
      setPlanSaving(false)
    }
  }

  const submitCreate = async () => {
    const key = createForm.key.trim()
    const name = createForm.name.trim()
    if (!key || !name) {
      setError(tAdmin(locale, 'iam.overview.error.required'))
      return
    }
    setCreating(true)
    try {
      await createIamTenant({
        key,
        name,
        plan: createForm.plan,
        status: createForm.status,
      })
      setCreateModalOpen(false)
      setCreateForm({ key: '', name: '', plan: 'free', status: 'active' })
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : tAdmin(locale, 'iam.overview.error.createTenant'))
    } finally {
      setCreating(false)
    }
  }

  const metricValueMap = useMemo(() => {
    const map = new Map<string, string>()
    metrics.forEach((item) => {
      map.set(toLabel(item.key), item.value)
    })
    return map
  }, [metrics])

  const topMetrics = [
    { key: 'members', label: tAdmin(locale, 'iam.overview.metrics.members'), value: metricValueMap.get('members') ?? metricValueMap.get('member_count') ?? '-' },
    { key: 'roles', label: tAdmin(locale, 'iam.overview.metrics.roles'), value: metricValueMap.get('roles') ?? metricValueMap.get('role_count') ?? '-' },
    { key: 'tenants', label: tAdmin(locale, 'iam.overview.metrics.tenants'), value: metricValueMap.get('tenants') ?? metricValueMap.get('tenant_count') ?? '-' },
    { key: 'delegated', label: tAdmin(locale, 'iam.overview.metrics.delegated'), value: metricValueMap.get('delegated') ?? metricValueMap.get('delegated_iam') ?? '-' },
  ]

  return (
    <main className="px-admin-page" data-testid="iam-overview-page">
      <section className="px-admin-shell">
        <section className="px-cap-hero">
          <div className="px-cap-hero-head">
            <div>
              <p className="px-admin-card-text" style={{ textTransform: 'uppercase' }}>{tAdmin(locale, 'iam.overview.orgAccess')}</p>
              <h1 className="px-cap-page-title">{tAdmin(locale, 'iam.overview.title')}</h1>
              <p className="px-cap-page-desc">{tAdmin(locale, 'iam.overview.caption')}</p>
            </div>
            <div className="px-admin-toolbar">
              <button type="button" className="px-btn" onClick={() => setCreateModalOpen(true)}>{tAdmin(locale, 'iam.overview.createTenant')}</button>
              <button
                type="button"
                className="px-btn-ghost"
                disabled={!tenants.length}
                onClick={() => tenants[0] && openPlanModal(tenants[0])}
              >
                {tAdmin(locale, 'iam.overview.adjustPlan')}
              </button>
              <button type="button" className="px-btn-ghost" onClick={() => void loadAll()}>
                {tAdmin(locale, 'iam.overview.refresh')}
              </button>
            </div>
          </div>
        </section>

        {error ? (
          <p role="alert" className="px-alert px-alert-danger" style={{ marginBottom: 12 }}>
            {error}
          </p>
        ) : null}

        <div className="px-iam-metrics-grid">
          {topMetrics.map((item) => (
            <article key={item.key} className="px-admin-card" style={{ margin: 0 }}>
              <p className="px-admin-card-text">{item.label}</p>
              <p style={{ fontSize: 28, fontWeight: 700, marginTop: 8 }}>{item.value}</p>
            </article>
          ))}
        </div>

        <div className="px-iam-categories-grid" style={{ marginTop: 14 }}>
          {settingCategories.map((category) => (
          <Link
            key={category.key}
            href={category.path}
              className="px-admin-card"
              style={{ margin: 0, textDecoration: 'none', color: 'inherit', transition: 'box-shadow .2s ease, transform .2s ease' }}
            >
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
                <div style={{ width: 40, height: 40, borderRadius: 10, background: category.iconBg, display: 'grid', placeItems: 'center', fontSize: 18 }}>
                  <span aria-hidden>{category.icon}</span>
                </div>
                <div>
                  <h3 className="px-admin-card-title">{category.title}</h3>
                  <p className="px-admin-card-text">{category.description}</p>
                </div>
              </div>
            </Link>
          ))}
        </div>

        <article className="px-admin-card" style={{ marginTop: 14 }}>
          <h2 className="px-admin-card-title">{tAdmin(locale, 'iam.overview.table.title')}</h2>
          <p className="px-admin-card-text">{tAdmin(locale, 'iam.overview.table.caption')}</p>

          {loading ? (
            <p className="px-admin-card-text" style={{ marginTop: 12 }}>{tAdmin(locale, 'iam.overview.loading')}</p>
          ) : (
            <div className="px-table-wrap" style={{ marginTop: 12 }}>
              <table className="px-table" data-testid="iam-overview-tenants-table">
                <thead>
                  <tr>
                    <th>{tAdmin(locale, 'iam.overview.table.col.tenant')}</th>
                    <th>{tAdmin(locale, 'iam.overview.table.col.key')}</th>
                    <th>{tAdmin(locale, 'iam.overview.table.col.status')}</th>
                    <th>{tAdmin(locale, 'iam.overview.table.col.plan')}</th>
                    <th>{tAdmin(locale, 'iam.overview.table.col.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {tenants.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-admin-card-text">{tAdmin(locale, 'iam.overview.table.empty')}</td>
                    </tr>
                  ) : tenants.map((tenant) => (
                    <tr key={tenant.id}>
                      <td>{tenant.name}</td>
                      <td>{tenant.key}</td>
                      <td>
                        <span className="px-cap-tag" style={statusBadgeStyle(tenant.status)}>
                          {statusText(locale, tenant.status)}
                        </span>
                      </td>
                      <td>{tenant.plan || 'free'}</td>
                      <td>
                        <div className="px-row-actions">
                          <select
                            className="px-select"
                            style={{ maxWidth: 140 }}
                            value={tenant.status || 'active'}
                            disabled={updatingTenantId === tenant.id}
                            onChange={(event) => void handleChangeTenantStatus(tenant, event.target.value)}
                          >
                            <option value="active">{tAdmin(locale, 'iam.overview.status.active')}</option>
                            <option value="suspended">{tAdmin(locale, 'iam.overview.status.suspended')}</option>
                          </select>
                          <button type="button" className="px-btn-ghost" onClick={() => openPlanModal(tenant)}>
                            {updatingTenantId === tenant.id ? tAdmin(locale, 'common.updating') : tAdmin(locale, 'iam.overview.adjustPlan')}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </article>
      </section>

      {planModalOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(2, 6, 23, 0.55)',
            display: 'grid',
            placeItems: 'center',
            zIndex: 1300,
            padding: 16,
          }}
          onClick={() => setPlanModalOpen(false)}
        >
          <article className="px-admin-card" style={{ width: 'min(560px, 100%)' }} onClick={(e) => e.stopPropagation()}>
            <h3 className="px-admin-card-title">{tAdmin(locale, 'iam.overview.modal.plan.title')}</h3>
            <p className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.plan.caption')}</p>

            {selectedTenant ? (
              <p className="px-admin-card-text" style={{ marginTop: 8 }}>{selectedTenant.name || selectedTenant.key}</p>
            ) : null}

            <div style={{ marginTop: 12, display: 'grid', gap: 10 }}>
              <label>
                <div className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.plan.plan')}</div>
                <select className="px-select" value={planForm.plan} onChange={(e) => setPlanForm((prev) => ({ ...prev, plan: e.target.value }))}>
                  <option value="free">Free</option>
                  <option value="standard">Standard</option>
                  <option value="premium">Premium</option>
                </select>
              </label>
              <label>
                <div className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.plan.name')}</div>
                <input className="px-field" value={planForm.name} onChange={(e) => setPlanForm((prev) => ({ ...prev, name: e.target.value }))} />
              </label>
            </div>

            <div className="px-admin-toolbar" style={{ marginTop: 14, justifyContent: 'flex-end' }}>
              <button type="button" className="px-btn-ghost" onClick={() => setPlanModalOpen(false)}>{tAdmin(locale, 'common.cancel')}</button>
              <button type="button" className="px-btn" disabled={planSaving} onClick={() => void submitPlan()}>{planSaving ? tAdmin(locale, 'common.saving') : tAdmin(locale, 'common.save')}</button>
            </div>
          </article>
        </div>
      ) : null}

      {createModalOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(2, 6, 23, 0.55)',
            display: 'grid',
            placeItems: 'center',
            zIndex: 1300,
            padding: 16,
          }}
          onClick={() => setCreateModalOpen(false)}
        >
          <article className="px-admin-card" style={{ width: 'min(680px, 100%)' }} onClick={(e) => e.stopPropagation()}>
            <h3 className="px-admin-card-title">{tAdmin(locale, 'iam.overview.modal.create.title')}</h3>
            <p className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.create.caption')}</p>

            <div style={{ marginTop: 12, display: 'grid', gap: 10 }}>
              <label>
                <div className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.create.key')}</div>
                <input className="px-input" value={createForm.key} placeholder="tenant-key" onChange={(e) => setCreateForm((prev) => ({ ...prev, key: e.target.value }))} />
              </label>
              <label>
                <div className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.create.name')}</div>
                <input className="px-input" value={createForm.name} onChange={(e) => setCreateForm((prev) => ({ ...prev, name: e.target.value }))} />
              </label>
              <div className="px-iam-modal-grid">
                <label>
                  <div className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.create.plan')}</div>
                  <select className="px-select" value={createForm.plan} onChange={(e) => setCreateForm((prev) => ({ ...prev, plan: e.target.value }))}>
                    <option value="free">Free</option>
                    <option value="standard">Standard</option>
                    <option value="premium">Premium</option>
                  </select>
                </label>
                <label>
                  <div className="px-admin-card-text">{tAdmin(locale, 'iam.overview.modal.create.status')}</div>
                  <select className="px-select" value={createForm.status} onChange={(e) => setCreateForm((prev) => ({ ...prev, status: e.target.value }))}>
                    <option value="active">{tAdmin(locale, 'iam.overview.status.active')}</option>
                    <option value="suspended">{tAdmin(locale, 'iam.overview.status.suspended')}</option>
                  </select>
                </label>
              </div>
            </div>

            <div className="px-admin-toolbar" style={{ marginTop: 14, justifyContent: 'flex-end' }}>
              <button type="button" className="px-btn-ghost" disabled={creating} onClick={() => setCreateModalOpen(false)}>{tAdmin(locale, 'common.cancel')}</button>
              <button type="button" className="px-btn" disabled={creating} onClick={() => void submitCreate()}>{creating ? tAdmin(locale, 'common.creating') : tAdmin(locale, 'common.save')}</button>
            </div>
          </article>
        </div>
      ) : null}
    </main>
  )
}
