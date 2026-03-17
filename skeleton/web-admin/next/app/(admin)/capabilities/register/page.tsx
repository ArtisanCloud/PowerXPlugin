'use client'

import { useEffect, useMemo, useState } from 'react'
import { getCapabilityExposure, getCapabilityExposureTemplate, invokeCapability, listCapabilitiesCatalog, upsertCapabilityExposure } from '@/lib/api/capabilities'
import { ApiError } from '@/lib/api/normalizeApiError'
import { tAdmin } from '@/lib/i18n/admin'
import { useLocalePreference } from '@/lib/ui/preferences'

type CatalogEntry = {
  capability_id?: string
  capabilityId?: string
  id?: string
  name?: string
  module?: string
  kind?: string
  type?: string
  tags?: string[]
  checksum?: string
  version_checksum?: string
  sync_status?: string
  exposure_status?: string
  updatedAt?: string
  updated_at?: string
  execution?: {
    mode?: string
  }
}

type CatalogGroup = {
  module: string
  title: string
  list: CatalogEntry[]
}

type ExposureChannel = {
  type?: string
  name?: string
  enabled?: boolean
  scopes?: string[]
  method?: string
  path?: string
  target?: string
  description?: string
}

type ExposurePackage = {
  capability_id?: string
  docs_version?: string
  sdk_version?: string
  channels?: ExposureChannel[]
  sync_status?: string
  updated_at?: string
}

type ExposureTemplate = {
  channel_types?: string[]
  default_rate?: {
    requests_per_minute?: number
    burst?: number
    concurrency?: number
  }
}

type ExposureDraftChannel = {
  type: string
  name: string
  enabled: boolean
  scopes: string[]
  scopesText: string
  method?: string
  path?: string
  target?: string
  description?: string
}

type ExposureDraft = {
  capability_id: string
  docs_version: string
  sdk_version: string
  channels: ExposureDraftChannel[]
  sync_status: string
  updated_at: string
}

type InvokeStatus = 'idle' | 'running' | 'succeeded' | 'failed'

function normalizeModule(entry: CatalogEntry): string {
  const value = String(entry.module || '').trim()
  if (!value) return 'misc'
  return value
}

function pickCapabilityID(item: CatalogEntry): string {
  return String(item.capability_id || item.capabilityId || item.id || item.name || '-')
}

function formatKind(locale: 'zh-CN' | 'en', item: CatalogEntry): string {
  const kind = String(item.kind || item.type || '').toLowerCase()
  if (kind.includes('workflow')) return tAdmin(locale, 'cap.register.kind.workflow')
  if (kind.includes('capability') || kind.includes('atomic')) return tAdmin(locale, 'cap.register.kind.capability')
  return tAdmin(locale, 'cap.register.kind.other')
}

function formatExposure(locale: 'zh-CN' | 'en', item: CatalogEntry): string {
  const value = String(item.exposure_status || item.sync_status || '').trim()
  if (!value) return tAdmin(locale, 'cap.register.exposure.unset')
  return value
}

function parseExposurePayload(payload: Record<string, unknown>): ExposurePackage | null {
  const maybePackage = (payload.package || payload) as ExposurePackage | null | undefined
  if (!maybePackage || typeof maybePackage !== 'object') return null
  return maybePackage
}

function parseExposureTemplate(payload: Record<string, unknown>): ExposureTemplate | null {
  if (!payload || typeof payload !== 'object') return null
  return payload as ExposureTemplate
}

function defaultChannelNames(locale: 'zh-CN' | 'en') {
  return locale === 'en'
    ? {
        rest: 'REST',
        graphql: 'GraphQL',
        grpc: 'gRPC',
        webhook: 'Webhook',
        workflow: 'Workflow',
        agent: 'Agent',
        agent_sse: 'Agent SSE',
        sdk: 'SDK',
      }
    : {
        rest: 'REST',
        graphql: 'GraphQL',
        grpc: 'gRPC',
        webhook: 'Webhook',
        workflow: 'Workflow',
        agent: 'Agent',
        agent_sse: 'Agent SSE',
        sdk: 'SDK',
      }
}

function buildDefaultTestPayload(capabilityID: string): string {
  return JSON.stringify(
    {
      endpoint: 'powerx.knowledge.v1.KnowledgeSpaceAdminService',
      rpc: 'CreateKnowledgeSpace',
      metadata: {},
      body: {
        capability_id: capabilityID,
      },
    },
    null,
    2,
  )
}

function mergeExposureChannels(
  locale: 'zh-CN' | 'en',
  template: ExposureTemplate | null,
  exposure: ExposurePackage | null,
): ExposureDraftChannel[] {
  const names = defaultChannelNames(locale)
  const existing = exposure?.channels || []
  const existingMap = new Map(existing.map((channel) => [String(channel.type || '').trim(), channel] as const))
  const templateTypes = template?.channel_types?.length
    ? template.channel_types
    : ['rest', 'graphql', 'grpc', 'webhook', 'workflow', 'agent', 'agent_sse', 'sdk']

  const merged = templateTypes.map((type) => {
    const key = String(type || '').trim()
    const hit = existingMap.get(key)
    const scopes = hit?.scopes || []
    return {
      type: key,
      name: hit?.name || names[key as keyof ReturnType<typeof defaultChannelNames>] || key || 'channel',
      enabled: !!hit?.enabled,
      method: hit?.method,
      path: hit?.path,
      target: hit?.target,
      description: hit?.description,
      scopes,
      scopesText: scopes.join(', '),
    }
  })

  for (const channel of existing) {
    const key = String(channel.type || '').trim()
    if (!key) continue
    if (merged.some((item) => item.type === key)) continue
    const scopes = channel.scopes || []
    merged.push({
      type: key,
      name: channel.name || key,
      enabled: !!channel.enabled,
      method: channel.method,
      path: channel.path,
      target: channel.target,
      description: channel.description,
      scopes,
      scopesText: scopes.join(', '),
    })
  }
  return merged
}

export default function CapabilitiesRegisterPage() {
  const locale = useLocalePreference()
  const [catalog, setCatalog] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedModules, setExpandedModules] = useState<Record<string, boolean>>({})
  const [actionLoadingId, setActionLoadingId] = useState('')

  const [exposureOpen, setExposureOpen] = useState(false)
  const [exposureData, setExposureData] = useState<ExposurePackage | null>(null)
  const [exposureTemplate, setExposureTemplate] = useState<ExposureTemplate | null>(null)
  const [exposureDraft, setExposureDraft] = useState<ExposureDraft | null>(null)
  const [exposureLoading, setExposureLoading] = useState(false)
  const [exposureSaving, setExposureSaving] = useState(false)
  const [exposureNotice, setExposureNotice] = useState('')

  const [testOpen, setTestOpen] = useState(false)
  const [invokeStatus, setInvokeStatus] = useState<InvokeStatus>('idle')
  const [invokeTrace, setInvokeTrace] = useState('—')
  const [invokeError, setInvokeError] = useState('')
  const [invokeResult, setInvokeResult] = useState('')
  const [testForm, setTestForm] = useState({
    capabilityID: '',
    action: 'Space',
    protocol: 'grpc',
    targetMode: 'local',
    apiBase: 'http://127.0.0.1:8078',
    tenantUuid: '00000000-0000-0000-0000-000000000001',
    mockModule: 'media / workflow',
    payload: '{}',
  })

  const groupedCatalog = useMemo<CatalogGroup[]>(() => {
    const map = new Map<string, CatalogEntry[]>()
    for (const item of catalog) {
      const moduleName = normalizeModule(item)
      const list = map.get(moduleName) || []
      list.push(item)
      map.set(moduleName, list)
    }
    return [...map.entries()]
      .map(([module, list]) => ({
        module,
        title: module.split('.').pop() || module,
        list,
      }))
      .sort((a, b) => a.module.localeCompare(b.module))
  }, [catalog])

  useEffect(() => {
    const next: Record<string, boolean> = {}
    for (const item of groupedCatalog) {
      next[item.module] = expandedModules[item.module] ?? true
    }
    setExpandedModules(next)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groupedCatalog.length])

  const loadAll = async () => {
    setLoading(true)
    setError('')
    try {
      const entries = await listCapabilitiesCatalog()
      setCatalog((entries || []) as CatalogEntry[])
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError(tAdmin(locale, 'cap.register.error.load'))
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locale])

  const toggleModule = (module: string) => {
    setExpandedModules((prev) => ({
      ...prev,
      [module]: !prev[module],
    }))
  }

  const handleConfigure = async (item: CatalogEntry) => {
    const capabilityID = pickCapabilityID(item)
    if (!capabilityID || capabilityID === '-') return

    setActionLoadingId(capabilityID)
    setExposureNotice('')
    try {
      const [templatePayload, exposurePayload] = await Promise.all([
        getCapabilityExposureTemplate(),
        getCapabilityExposure(capabilityID),
      ])
      const template = parseExposureTemplate(templatePayload)
      const exposure = parseExposurePayload(exposurePayload)
      setExposureTemplate(template)
      setExposureData(exposure)
      setExposureDraft({
        capability_id: capabilityID,
        docs_version: exposure?.docs_version || '1.0.0',
        sdk_version: exposure?.sdk_version || '1.0.0',
        channels: mergeExposureChannels(locale, template, exposure),
        sync_status: String(exposure?.sync_status || 'unconfigured'),
        updated_at: String(exposure?.updated_at || ''),
      })
      setExposureOpen(true)
    } catch (err) {
      const message = err instanceof ApiError ? err.message : tAdmin(locale, 'cap.register.error.load')
      setError(message)
    } finally {
      setActionLoadingId('')
    }
  }

  const handleExposureReload = async () => {
    const capabilityID = exposureDraft?.capability_id || exposureData?.capability_id || ''
    if (!capabilityID) return
    setExposureLoading(true)
    setExposureNotice('')
    try {
      const payload = await getCapabilityExposure(capabilityID)
      const exposure = parseExposurePayload(payload)
      setExposureData(exposure)
      setExposureDraft((prev) => ({
        capability_id: capabilityID,
        docs_version: exposure?.docs_version || prev?.docs_version || '1.0.0',
        sdk_version: exposure?.sdk_version || prev?.sdk_version || '1.0.0',
        channels: mergeExposureChannels(locale, exposureTemplate, exposure),
        sync_status: String(exposure?.sync_status || 'unconfigured'),
        updated_at: String(exposure?.updated_at || ''),
      }))
      setExposureNotice(locale === 'en' ? 'Configuration loaded.' : '配置已加载。')
    } catch (err) {
      setExposureNotice(err instanceof ApiError ? err.message : (locale === 'en' ? 'Load failed' : '加载失败'))
    } finally {
      setExposureLoading(false)
    }
  }

  const handleExposureResetChannels = () => {
    setExposureDraft((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        channels: prev.channels.map((channel) => ({
          ...channel,
          enabled: false,
          scopes: [],
          scopesText: '',
        })),
      }
    })
  }

  const handleExposureSave = async () => {
    if (!exposureDraft?.capability_id) return
    setExposureSaving(true)
    setExposureNotice('')
    try {
      const rate = exposureTemplate?.default_rate || {}
      const payload = {
        capability_id: exposureDraft.capability_id,
        docs_version: exposureDraft.docs_version || '1.0.0',
        sdk_version: exposureDraft.sdk_version || '1.0.0',
        channels: exposureDraft.channels.map((channel) => ({
          type: channel.type,
          name: channel.name,
          enabled: channel.enabled,
          method: channel.method,
          path: channel.path,
          target: channel.target,
          description: channel.description,
          scopes: channel.scopes,
        })),
        auth: {
          strategy: 'powerx_session',
          audience: '',
          scopes: [],
        },
        rate_limit: {
          requests_per_minute: Number(rate.requests_per_minute || 600),
          burst: Number(rate.burst || 120),
          concurrency: Number(rate.concurrency || 10),
        },
        tenants: [],
      }
      const saved = await upsertCapabilityExposure(exposureDraft.capability_id, payload)
      const savedExposure = parseExposurePayload(saved)
      setExposureData(savedExposure)
      setExposureDraft((prev) => ({
        capability_id: exposureDraft.capability_id,
        docs_version: savedExposure?.docs_version || prev?.docs_version || '1.0.0',
        sdk_version: savedExposure?.sdk_version || prev?.sdk_version || '1.0.0',
        channels: mergeExposureChannels(locale, exposureTemplate, savedExposure),
        sync_status: String(savedExposure?.sync_status || prev?.sync_status || 'pending'),
        updated_at: String(savedExposure?.updated_at || ''),
      }))
      setExposureNotice(locale === 'en' ? 'Saved.' : '已保存。')
    } catch (err) {
      setExposureNotice(err instanceof ApiError ? err.message : (locale === 'en' ? 'Save failed' : '保存失败'))
    } finally {
      setExposureSaving(false)
    }
  }

  const handleDebug = (item: CatalogEntry) => {
    const capabilityID = pickCapabilityID(item)
    if (!capabilityID || capabilityID === '-') return
    setInvokeStatus('idle')
    setInvokeTrace('—')
    setInvokeError('')
    setInvokeResult('')
    setTestForm((prev) => ({
      ...prev,
      capabilityID,
      action: capabilityID.split('.').pop() || 'Action',
      protocol: 'grpc',
      payload: buildDefaultTestPayload(capabilityID),
    }))
    setTestOpen(true)
  }

  const handleInvoke = async (kind: 'success' | 'fail' | 'mock' = 'success') => {
    setInvokeStatus('running')
    setInvokeError('')
    setInvokeResult('')

    let payloadJSON: Record<string, unknown> = {}
    try {
      payloadJSON = JSON.parse(testForm.payload || '{}') as Record<string, unknown>
    } catch {
      setInvokeStatus('failed')
      setInvokeError(locale === 'en' ? 'Payload must be valid JSON' : 'Payload 必须是有效 JSON')
      return
    }

    try {
      let normalizedPayload: Record<string, unknown> = payloadJSON
      if (testForm.protocol === 'grpc') {
        const nestedBody = (payloadJSON.body && typeof payloadJSON.body === 'object')
          ? (payloadJSON.body as Record<string, unknown>)
          : {}
        const rpc = String(payloadJSON.rpc || nestedBody.rpc || '').trim()
        const endpoint = String(payloadJSON.endpoint || nestedBody.service || '').trim()
        const grpcBody = (nestedBody.input && typeof nestedBody.input === 'object')
          ? (nestedBody.input as Record<string, unknown>)
          : nestedBody
        normalizedPayload = {
          endpoint,
          rpc: rpc || 'Method',
          metadata: (payloadJSON.metadata && typeof payloadJSON.metadata === 'object')
            ? payloadJSON.metadata
            : (nestedBody.metadata && typeof nestedBody.metadata === 'object' ? nestedBody.metadata : {}),
          body: grpcBody,
        }
      }

      const isGrpcPayload = String(normalizedPayload.endpoint || '').trim() !== ''
        && String(normalizedPayload.rpc || '').trim() !== ''
      const protocolForInvoke = isGrpcPayload ? 'grpc' : (testForm.protocol || undefined)

      const requestBody: Record<string, unknown> = {
        capabilityId: testForm.capabilityID || undefined,
        action: testForm.action || undefined,
        preferredProtocol: protocolForInvoke,
        payload: normalizedPayload,
        metadata: {
          mode: testForm.targetMode || undefined,
          apiBase: testForm.apiBase || undefined,
          tenantUuid: testForm.tenantUuid || undefined,
          kind,
        },
      }
      const requestHeaders: Record<string, string> = {}
      if (testForm.mockModule.trim()) {
        requestHeaders['X-PX-Use-Mock'] = testForm.mockModule.trim()
      }
      const response = await invokeCapability(requestBody, requestHeaders)
      setInvokeStatus('succeeded')
      setInvokeTrace(String(response.trace_id || response.traceId || `trace-${Date.now()}`))
      setInvokeResult(JSON.stringify(response, null, 2))
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'invoke failed'
      setInvokeStatus('failed')
      setInvokeTrace(`trace-${Date.now()}`)
      setInvokeError(message)
    }
  }

  const exposureChannels = exposureDraft?.channels || []

  return (
    <main className="px-admin-page" data-testid="capability-register-page">
      <section className="px-admin-shell">
        <section className="px-cap-hero">
          <div className="px-cap-hero-head">
            <div>
              <h1 className="px-cap-page-title">{tAdmin(locale, 'cap.register.tableTitle')}</h1>
              <p className="px-cap-page-desc">{tAdmin(locale, 'cap.register.desc')}</p>
            </div>
            <div className="px-admin-toolbar">
              <button type="button" className="px-btn-ghost" onClick={() => void loadAll()}>
                {tAdmin(locale, 'cap.register.refresh')}
              </button>
              <button type="button" className="px-btn">
                {tAdmin(locale, 'cap.register.create')}
              </button>
            </div>
          </div>
          <div className="px-cap-sync-hint">
            <p className="px-cap-sync-question">{locale === 'en' ? 'Need to refresh catalog?' : '能力目录未刷新?'}</p>
            <p className="px-cap-sync-text">{locale === 'en'
              ? 'Catalog defaults to plugin.yaml capabilities.imports/provides. Generate catalog.json only when checksum validation requires it.'
              : '能力目录默认从 plugin.yaml 的 capabilities.imports/provides 解析，若要生成 capabilities/catalog.json 用于发布校验，请使用 PowerXPlugin 仓库内的 catalog 工具。'}</p>
            <button type="button" className="px-cap-sync-toggle" aria-label="catalog-toggle" />
          </div>
        </section>

        <article className="px-admin-card">
          <h2 className="px-admin-card-title">{locale === 'en' ? 'Registered Capabilities' : '已注册能力'}</h2>
          <p className="px-admin-card-text">{tAdmin(locale, 'cap.register.tableHint')}</p>

          {error ? (
            <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>
              {error}
            </p>
          ) : null}

          {loading ? (
            <p className="px-admin-card-text" style={{ marginTop: 12 }}>{tAdmin(locale, 'templates.crud.loading')}</p>
          ) : null}

          {!loading && groupedCatalog.length === 0 ? (
            <p className="px-admin-card-text" style={{ marginTop: 12 }}>{tAdmin(locale, 'cap.register.empty')}</p>
          ) : null}

          {!loading && groupedCatalog.length > 0 ? (
            <div className="px-cap-group-list">
              {groupedCatalog.map((group) => {
                const open = expandedModules[group.module] ?? true
                return (
                  <section key={group.module} className="px-cap-group-card">
                    <button
                      type="button"
                      className="px-cap-group-head"
                      onClick={() => toggleModule(group.module)}
                    >
                      <div>
                        <h3>{group.title}</h3>
                        <p>{group.module} · {group.list.length} {tAdmin(locale, 'cap.register.count')}</p>
                      </div>
                      <div className="px-cap-group-meta">
                        <span>{tAdmin(locale, 'cap.register.atomicCount')} · {group.list.length}</span>
                        <span className={`px-cap-chevron ${open ? 'is-open' : ''}`}>⌄</span>
                      </div>
                    </button>

                    {open ? (
                      <div className="px-table-wrap">
                        <table className="px-table" data-testid="capability-register-catalog-table">
                          <thead>
                            <tr>
                              <th>{tAdmin(locale, 'cap.register.capabilityName')}</th>
                              <th>{tAdmin(locale, 'cap.register.kind')}</th>
                              <th>{tAdmin(locale, 'cap.register.executionMode')}</th>
                              <th>{tAdmin(locale, 'cap.register.exposure')}</th>
                              <th>{tAdmin(locale, 'cap.register.tags')}</th>
                              <th>{tAdmin(locale, 'cap.register.checksum')}</th>
                              <th>{tAdmin(locale, 'templates.crud.col.actions')}</th>
                            </tr>
                          </thead>
                          <tbody>
                            {group.list.map((item, index) => (
                              <tr key={`${group.module}-${pickCapabilityID(item)}-${index}`}>
                                <td>
                                  <div className="px-cap-id">{pickCapabilityID(item)}</div>
                                  <small>版本 {(item.updatedAt || item.updated_at || '-')}</small>
                                </td>
                                <td>{formatKind(locale, item)}</td>
                                <td>{item.execution?.mode ? String(item.execution.mode).toUpperCase() : 'SYNC'}</td>
                                <td>{formatExposure(locale, item)}</td>
                                <td>
                                  <div className="px-cap-tags">
                                    {Array.isArray(item.tags) && item.tags.length ? item.tags.map((tag) => (
                                      <span key={`${pickCapabilityID(item)}-${tag}`} className="px-cap-tag">{tag}</span>
                                    )) : '-'}
                                  </div>
                                </td>
                                <td className="px-cap-checksum">{String(item.checksum || item.version_checksum || '-')}</td>
                                <td>
                                  <div className="px-row-actions">
                                    <button
                                      type="button"
                                      className="px-btn"
                                      disabled={actionLoadingId === pickCapabilityID(item)}
                                      onClick={() => void handleConfigure(item)}
                                    >
                                      {actionLoadingId === pickCapabilityID(item) ? tAdmin(locale, 'templates.crud.loading') : tAdmin(locale, 'cap.register.configure')}
                                    </button>
                                    <button type="button" className="px-btn-ghost" onClick={() => handleDebug(item)}>
                                      {tAdmin(locale, 'cap.register.test')}
                                    </button>
                                  </div>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : null}
                  </section>
                )
              })}
            </div>
          ) : null}
        </article>
      </section>

      {exposureOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(2, 6, 23, 0.55)',
            display: 'grid',
            placeItems: 'center',
            zIndex: 1200,
            padding: 16,
          }}
          onClick={() => setExposureOpen(false)}
        >
          <article
            className="px-admin-card"
            style={{ width: 'min(1120px, 100%)', maxHeight: '88vh', overflow: 'auto' }}
            onClick={(event) => event.stopPropagation()}
          >
            <div className="px-admin-toolbar" style={{ justifyContent: 'space-between', marginTop: 0 }}>
              <div>
                <h3 className="px-admin-card-title" style={{ margin: 0 }}>{locale === 'en' ? 'Capability Exposure Configuration' : '能力暴露通道配置'}</h3>
                <p className="px-admin-card-text" style={{ marginTop: 6 }}>
                  {locale === 'en' ? 'Configure exposure channels, policy and tenant quota.' : '为能力维护通道、鉴权策略、限流与租户授权。'}
                </p>
              </div>
              <button type="button" className="px-btn-ghost" onClick={() => setExposureOpen(false)}>
                {tAdmin(locale, 'templates.crud.modal.cancel')}
              </button>
            </div>

            <div style={{ marginTop: 16, display: 'grid', gap: 14 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span className="px-admin-card-text">能力 ID:</span>
                <span className="px-cap-tag">{String(exposureDraft?.capability_id || exposureData?.capability_id || '-')}</span>
                <span className="px-admin-card-text">{exposureDraft?.sync_status || '未配置'}</span>
                <span className="px-admin-card-text">最近更新: {exposureDraft?.updated_at || '—'}</span>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 160px 160px', gap: 12 }}>
                <label>
                  <div className="px-admin-card-text">能力 ID *</div>
                  <input
                    className="px-input"
                    value={String(exposureDraft?.capability_id || exposureData?.capability_id || '-')}
                    onChange={(e) => setExposureDraft((prev) => (prev ? { ...prev, capability_id: e.target.value } : prev))}
                  />
                </label>
                <label>
                  <div className="px-admin-card-text">文档版本</div>
                  <input
                    className="px-input"
                    value={String(exposureDraft?.docs_version || '1.0.0')}
                    onChange={(e) => setExposureDraft((prev) => (prev ? { ...prev, docs_version: e.target.value } : prev))}
                  />
                </label>
                <label>
                  <div className="px-admin-card-text">SDK 版本</div>
                  <input
                    className="px-input"
                    value={String(exposureDraft?.sdk_version || '1.0.0')}
                    onChange={(e) => setExposureDraft((prev) => (prev ? { ...prev, sdk_version: e.target.value } : prev))}
                  />
                </label>
              </div>

              <div className="px-admin-toolbar" style={{ marginTop: 0 }}>
                <button type="button" className="px-btn-ghost" disabled={exposureLoading} onClick={() => void handleExposureReload()}>
                  {exposureLoading ? (locale === 'en' ? 'Loading...' : '加载中...') : (locale === 'en' ? 'Load Config' : '加载配置')}
                </button>
                <button type="button" className="px-btn-ghost" onClick={handleExposureResetChannels}>
                  {locale === 'en' ? 'Reset Channels' : '重置通道'}
                </button>
                <button type="button" className="px-btn" disabled={exposureSaving} onClick={() => void handleExposureSave()}>
                  {exposureSaving ? (locale === 'en' ? 'Saving...' : '保存中...') : (locale === 'en' ? 'Save' : '保存')}
                </button>
              </div>

              {exposureNotice ? <p className="px-admin-card-text">{exposureNotice}</p> : null}

              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <strong>{locale === 'en' ? 'Channel Configuration' : '通道配置'}</strong>
                <span className="px-admin-card-text">
                  {exposureChannels.filter((item) => !!item.enabled).length}/{exposureChannels.length} {locale === 'en' ? 'enabled' : '通道已启用'}
                </span>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 12 }}>
                {exposureChannels.map((channel, index) => (
                  <div key={`${channel.type || 'channel'}-${index}`} className="px-admin-card" style={{ margin: 0, padding: 14 }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                      <strong>{channel.name || channel.type || `channel-${index + 1}`}</strong>
                        <span
                          style={{
                            display: 'inline-block',
                            width: 34,
                            height: 20,
                            borderRadius: 999,
                            background: channel.enabled ? '#22c55e' : '#e2e8f0',
                            position: 'relative',
                            cursor: 'pointer',
                          }}
                          aria-label={channel.enabled ? 'enabled' : 'disabled'}
                          onClick={() => {
                            setExposureDraft((prev) => {
                              if (!prev) return prev
                              return {
                                ...prev,
                                channels: prev.channels.map((item) => item.type === channel.type ? { ...item, enabled: !item.enabled } : item),
                              }
                            })
                          }}
                        >
                        <span
                          style={{
                            position: 'absolute',
                            top: 2,
                            left: channel.enabled ? 16 : 2,
                            width: 16,
                            height: 16,
                            borderRadius: '50%',
                            background: '#fff',
                          }}
                        />
                      </span>
                    </div>
                    <div style={{ marginTop: 8, display: 'grid', gap: 6 }}>
                      {channel.method || channel.path ? <small>{channel.method || '-'} {channel.path || '-'}</small> : null}
                      {channel.target ? <small>{locale === 'en' ? 'Target' : '目标'}: {channel.target}</small> : null}
                      <small>{channel.description || (locale === 'en' ? 'Enabled channels appear in API/Workflow/Agent list.' : '启用后将出现在查主的 API/Workflow/Agent 列表中')}</small>
                      <label>
                        <small>{locale === 'en' ? 'Scopes' : '作用域'}</small>
                        <input
                          className="px-input"
                          style={{ marginTop: 6 }}
                          value={channel.scopesText}
                          placeholder="scope.one, scope.two"
                          onChange={(e) => {
                            const value = e.target.value
                            setExposureDraft((prev) => {
                              if (!prev) return prev
                              return {
                                ...prev,
                                channels: prev.channels.map((item) => item.type === channel.type
                                  ? {
                                      ...item,
                                      scopesText: value,
                                      scopes: value
                                        .split(',')
                                        .map((part) => part.trim())
                                        .filter(Boolean),
                                    }
                                  : item),
                              }
                            })
                          }}
                        />
                      </label>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </article>
        </div>
      ) : null}

      {testOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(2, 6, 23, 0.55)',
            display: 'grid',
            placeItems: 'center',
            zIndex: 1210,
            padding: 16,
          }}
          onClick={() => setTestOpen(false)}
        >
          <article
            className="px-admin-card"
            style={{ width: 'min(1240px, 100%)', maxHeight: '88vh', overflow: 'auto' }}
            onClick={(event) => event.stopPropagation()}
          >
            <div className="px-admin-toolbar" style={{ justifyContent: 'space-between', marginTop: 0 }}>
              <div>
                <h3 className="px-admin-card-title" style={{ margin: 0 }}>{locale === 'en' ? 'Capability Debug Panel' : '能力调试面板'}</h3>
                <p className="px-admin-card-text" style={{ marginTop: 6 }}>
                  {locale === 'en' ? 'Invoke local endpoint directly to verify protocol configuration.' : '无需依赖网关，直接调用插件本地接口验证配置。'}
                </p>
              </div>
              <button type="button" className="px-btn-ghost" onClick={() => setTestOpen(false)}>
                {tAdmin(locale, 'templates.crud.modal.cancel')}
              </button>
            </div>

            <div style={{ marginTop: 16, display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: 14 }}>
              <section className="px-admin-card" style={{ margin: 0 }}>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                  <label>
                    <div className="px-admin-card-text">能力 ID</div>
                    <input className="px-input" value={testForm.capabilityID} readOnly />
                  </label>
                  <label>
                    <div className="px-admin-card-text">动作</div>
                    <input className="px-input" value={testForm.action} onChange={(e) => setTestForm((prev) => ({ ...prev, action: e.target.value }))} />
                  </label>
                </div>

                <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1fr 1fr 1.2fr', gap: 10 }}>
                  <label>
                    <div className="px-admin-card-text">协议</div>
                    <select className="px-input" value={testForm.protocol} onChange={(e) => setTestForm((prev) => ({ ...prev, protocol: e.target.value }))}>
                      <option value="grpc">grpc</option>
                      <option value="rest">rest</option>
                      <option value="workflow">workflow</option>
                    </select>
                  </label>
                  <label>
                    <div className="px-admin-card-text">调试目标</div>
                    <select className="px-input" value={testForm.targetMode} onChange={(e) => setTestForm((prev) => ({ ...prev, targetMode: e.target.value }))}>
                      <option value="local">local</option>
                      <option value="gateway">gateway</option>
                    </select>
                  </label>
                  <label>
                    <div className="px-admin-card-text">API Base 地址</div>
                    <input className="px-input" value={testForm.apiBase} onChange={(e) => setTestForm((prev) => ({ ...prev, apiBase: e.target.value }))} />
                  </label>
                </div>

                <label style={{ display: 'block', marginTop: 10 }}>
                  <div className="px-admin-card-text">调试 Payload</div>
                  <textarea
                    className="px-code"
                    style={{ width: '100%', minHeight: 310, marginTop: 6 }}
                    value={testForm.payload}
                    onChange={(e) => setTestForm((prev) => ({ ...prev, payload: e.target.value }))}
                  />
                </label>

                <div style={{ marginTop: 10, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                  <label>
                    <div className="px-admin-card-text">Tenant UUID</div>
                    <input className="px-input" value={testForm.tenantUuid} onChange={(e) => setTestForm((prev) => ({ ...prev, tenantUuid: e.target.value }))} />
                  </label>
                  <label>
                    <div className="px-admin-card-text">Mock 模块</div>
                    <input className="px-input" value={testForm.mockModule} onChange={(e) => setTestForm((prev) => ({ ...prev, mockModule: e.target.value }))} />
                  </label>
                </div>

                <div className="px-admin-toolbar" style={{ marginTop: 12 }}>
                  <button type="button" className="px-btn" disabled={invokeStatus === 'running'} onClick={() => void handleInvoke('success')}>
                    {invokeStatus === 'running' ? (locale === 'en' ? 'Invoking...' : '调用中...') : (locale === 'en' ? 'Invoke' : '调用')}
                  </button>
                  <button type="button" className="px-btn-ghost" disabled={invokeStatus === 'running'} onClick={() => void handleInvoke('mock')}>
                    {locale === 'en' ? 'Mock Invoke' : '模拟调用'}
                  </button>
                  <button type="button" className="px-btn-ghost" disabled={invokeStatus === 'running'} onClick={() => void handleInvoke('fail')}>
                    {locale === 'en' ? 'Fail Case' : '失败用例'}
                  </button>
                </div>
              </section>

              <section className="px-admin-card" style={{ margin: 0 }}>
                <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap' }}>
                  <span>状态: <strong>{invokeStatus === 'idle' ? '未调用' : invokeStatus}</strong></span>
                  <span>Trace ID: <strong>{invokeTrace || '—'}</strong></span>
                </div>
                <div style={{ marginTop: 10 }}>
                  {invokeError ? <p className="px-alert px-alert-danger">{invokeError}</p> : null}
                  <pre className="px-code" style={{ minHeight: 420 }}>{invokeResult || '暂无记录'}</pre>
                </div>
              </section>
            </div>
          </article>
        </div>
      ) : null}
    </main>
  )
}
