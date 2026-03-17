'use client'

import { useEffect, useMemo, useState } from 'react'
import { listCapabilitiesCatalog } from '@/lib/api/capabilities'
import { ApiError } from '@/lib/api/normalizeApiError'
import { useLocalePreference } from '@/lib/ui/preferences'

type CatalogEntry = {
  id?: string
  capability_id?: string
  capabilityId?: string
  name?: string
  descriptor?: string
  description?: string
  module?: string
  version?: string
  updatedAt?: string
  updated_at?: string
}

function pickCapabilityID(item: CatalogEntry): string {
  return String(item.capability_id || item.capabilityId || item.id || item.name || '').trim()
}

function pickDescription(item: CatalogEntry): string {
  return String(item.description || item.descriptor || item.module || '').trim()
}

function pickVersion(item: CatalogEntry): string {
  return String(item.version || item.updatedAt || item.updated_at || '-').trim() || '-'
}

export default function CapabilitiesLifecyclePage() {
  const locale = useLocalePreference()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [catalog, setCatalog] = useState<CatalogEntry[]>([])

  const catalogCommand = locale === 'en'
    ? 'cd skeleton && px-plugin catalog generate --manifest ./plugin.yaml'
    : 'cd skeleton && px-plugin catalog generate --manifest ./plugin.yaml'

  const rows = useMemo(() => {
    return (catalog || [])
      .map((item) => ({
        capabilityId: pickCapabilityID(item),
        description: pickDescription(item),
        version: pickVersion(item),
      }))
      .filter((item) => item.capabilityId)
      .sort((a, b) => a.capabilityId.localeCompare(b.capabilityId))
  }, [catalog])

  const loadCatalog = async () => {
    setLoading(true)
    setError('')
    try {
      const entries = await listCapabilitiesCatalog()
      setCatalog((entries || []) as CatalogEntry[])
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError(locale === 'en' ? 'Failed to load lifecycle catalog.' : '加载生命周期能力目录失败。')
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadCatalog()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locale])

  return (
    <main className="px-admin-page" data-testid="capability-lifecycle-page">
      <section className="px-admin-shell">
        <section className="px-cap-hero">
          <div className="px-cap-hero-head">
            <div>
              <h1 className="px-cap-page-title">{locale === 'en' ? 'Capability Lifecycle Plan' : '能力生命周期计划'}</h1>
              <p className="px-cap-page-desc">
                {locale === 'en'
                  ? 'Describe rollout/deprecation differences, gray windows and notify matrix.'
                  : '生成能力升级/下线的差异说明、灰度窗口和通知矩阵，确保订阅方提前感知并可随时回滚。'}
              </p>
            </div>
            <div className="px-admin-toolbar">
              <button type="button" className="px-btn-ghost" onClick={() => void loadCatalog()}>
                {locale === 'en' ? 'Refresh' : '刷新'}
              </button>
            </div>
          </div>

          <div className="px-cap-sync-hint">
            <p className="px-cap-sync-question">{locale === 'en' ? 'Need to refresh catalog?' : '能力目录未刷新?'}</p>
            <p className="px-cap-sync-text">
              {locale === 'en'
                ? 'Catalog defaults to plugin.yaml capabilities.imports/provides. Generate catalog.json only for publish/checksum validation.'
                : '能力目录默认从 plugin.yaml 的 capabilities.imports/provides 解析，无需额外生成 catalog.json。若你要生成 capabilities/catalog.json 用于发布/校验，请使用 catalog 工具。'}
            </p>
            <code className="px-code" style={{ marginTop: 8, display: 'inline-block' }}>{catalogCommand}</code>
          </div>
        </section>

        <article className="px-admin-card">
          <h2 className="px-admin-card-title">{locale === 'en' ? 'Manageable Capabilities' : '可管理的能力'}</h2>
          <p className="px-admin-card-text">
            {locale === 'en' ? 'Sync capability catalog and pick items to plan lifecycle.' : '同步插件能力目录，挑选需要规划生命周期的能力。'}
          </p>

          {error ? (
            <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>
              {error}
            </p>
          ) : null}

          {loading ? (
            <p className="px-admin-card-text" style={{ marginTop: 12 }}>{locale === 'en' ? 'Loading...' : '加载中...'}</p>
          ) : null}

          {!loading && rows.length === 0 ? (
            <p className="px-admin-card-text" style={{ marginTop: 12 }}>{locale === 'en' ? 'No lifecycle candidates yet.' : '暂无可管理能力。'}</p>
          ) : null}

          {!loading && rows.length > 0 ? (
            <div className="px-table-wrap" style={{ marginTop: 12 }}>
              <table className="px-table" data-testid="capability-lifecycle-table">
                <thead>
                  <tr>
                    <th>{locale === 'en' ? 'Capability' : '能力'}</th>
                    <th>{locale === 'en' ? 'Description' : '描述'}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.capabilityId}>
                      <td>
                        <div className="px-cap-id">{row.capabilityId}</div>
                        <small>版本 {row.version}</small>
                      </td>
                      <td>{row.description || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </article>
      </section>
    </main>
  )
}
