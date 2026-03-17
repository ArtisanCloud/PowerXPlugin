'use client'

import { useEffect, useMemo, useState } from 'react'
import { invokeCapability, listCapabilitiesCatalog } from '@/lib/api/capabilities'
import { ApiError } from '@/lib/api/normalizeApiError'
import { useSearchParams } from 'next/navigation'

type CatalogEntry = {
  id?: string
  capability_id?: string
  capabilityId?: string
  module?: string
  kind?: string
  type?: string
  protocols?: unknown
}

type ModuleOption = {
  value: string
  label: string
  count: number
}

type CapabilityOption = {
  value: string
  label: string
  module: string
  kind: string
}

type InvokeHistory = {
  id: string
  capabilityId: string
  protocol: string
  status: string
  traceId: string
  success: boolean
  timestamp: string
}

function pickCapabilityId(item: CatalogEntry): string {
  return String(item.capability_id || item.capabilityId || item.id || '').trim()
}

function pickModule(item: CatalogEntry): string {
  const fromField = String(item.module || '').trim()
  if (fromField) return fromField
  const cap = pickCapabilityId(item)
  if (!cap) return 'misc'
  const parts = cap.split('.')
  return parts.length > 2 ? parts.slice(0, parts.length - 1).join('.') : cap
}

function nowLabel() {
  return new Date().toLocaleString('zh-CN', { hour12: false })
}

function normalizeKind(item: CatalogEntry): string {
  const raw = String(item.kind || '').trim().toLowerCase()
  if (!raw) return 'Capability'
  if (raw.includes('workflow')) return 'Workflow'
  if (raw.includes('atomic') || raw.includes('capability')) return 'Capability'
  return raw.charAt(0).toUpperCase() + raw.slice(1)
}

function normalizeProtocol(value: string): string {
  const raw = String(value || '').trim().toLowerCase()
  if (!raw) return ''
  if (raw === 'http') return 'rest'
  if (raw === 'workflow_step') return 'workflow'
  return raw
}

function extractSupportedProtocols(item?: CatalogEntry | null): string[] {
  if (!item) return ['rest', 'grpc']
  const found = new Set<string>()
  const protocols = item.protocols
  if (Array.isArray(protocols)) {
    for (const entry of protocols) {
      if (entry && typeof entry === 'object') {
        const row = entry as Record<string, unknown>
        const candidate = normalizeProtocol(String(row.protocol || row.channel || row.type || ''))
        if (candidate) found.add(candidate)
      }
    }
  } else if (protocols && typeof protocols === 'object') {
    for (const key of Object.keys(protocols as Record<string, unknown>)) {
      const candidate = normalizeProtocol(key)
      if (candidate) found.add(candidate)
    }
  }
  const kind = normalizeKind(item).toLowerCase()
  const capabilityId = pickCapabilityId(item).toLowerCase()
  const isWorkflow = kind.includes('workflow') || capabilityId.includes('.workflow.')
  const allowed = isWorkflow ? ['workflow', 'rest', 'grpc'] : ['rest', 'grpc']
  const filtered = [...found].filter((item) => allowed.includes(item))
  if (filtered.length) {
    const order = ['grpc', 'rest', 'workflow']
    return filtered.sort((a, b) => order.indexOf(a) - order.indexOf(b))
  }
  if (isWorkflow) return ['workflow']
  return ['grpc', 'rest']
}

function protocolLabel(value: string): string {
  const p = normalizeProtocol(value)
  if (p === 'grpc') return 'gRPC'
  if (p === 'rest') return 'REST'
  if (p === 'workflow') return 'Workflow'
  return p
}

function deriveDefaultAction(capability: string, protocol: string): string {
  const p = normalizeProtocol(protocol)
  const parts = String(capability || '').split('.').filter(Boolean)
  const tail = (parts[parts.length - 1] || '').toLowerCase()
  if (p === 'rest') {
    if (['read', 'list', 'query', 'search', 'manage'].includes(tail)) return 'List'
    if (['create', 'add'].includes(tail)) return 'Create'
    if (['update', 'edit', 'patch'].includes(tail)) return 'Update'
    if (['delete', 'remove'].includes(tail)) return 'Delete'
    return tail ? `${tail.charAt(0).toUpperCase()}${tail.slice(1)}` : 'List'
  }
  if (p === 'workflow') return 'CreateDefinition'
  if (tail) return `${tail.charAt(0).toUpperCase()}${tail.slice(1)}`
  return 'Invoke'
}

function deriveRestEndpoint(capability: string): string {
  const parts = String(capability || '').split('.').filter(Boolean)
  const start = parts.findIndex((item) => item === 'corex')
  const base = start >= 0 ? parts.slice(start + 1) : parts
  const trimmed = base.filter((item) => !['read', 'list', 'query', 'search', 'manage', 'create', 'add', 'update', 'edit', 'patch', 'delete', 'remove'].includes(item))
  const path = trimmed.length ? trimmed.join('/') : 'resource'
  return `/api/v1/${path}`
}

function deriveDefaultPayload(capability: string, protocol: string, action: string): string {
  const p = normalizeProtocol(protocol)
  if (p === 'rest') {
    const upperAction = String(action || '').toUpperCase()
    const method = upperAction.includes('DELETE')
      ? 'DELETE'
      : upperAction.includes('UPDATE')
        ? 'PUT'
        : upperAction.includes('CREATE')
          ? 'POST'
          : 'GET'
    return JSON.stringify(
      {
        method,
        endpoint: deriveRestEndpoint(capability),
        headers: {
          'Content-Type': 'application/json',
        },
        query: {},
        body: method === 'GET' ? {} : {},
      },
      null,
      2,
    )
  }
  if (p === 'workflow') {
    return JSON.stringify(
      {
        workflow: {
          template: capability || 'workflow.template',
        },
        payload: {},
      },
      null,
      2,
    )
  }
  return JSON.stringify(
    {
      endpoint: 'powerx.module.v1.Service',
      rpc: action || 'Method',
      metadata: {},
      body: {},
    },
    null,
    2,
  )
}

export default function CapabilityLabPage() {
  const searchParams = useSearchParams()
  const initialSource = (searchParams.get('source') || 'all').trim() || 'all'

  const [loadingCatalog, setLoadingCatalog] = useState(true)
  const [catalogError, setCatalogError] = useState('')
  const [catalog, setCatalog] = useState<CatalogEntry[]>([])

  const [source, setSource] = useState(initialSource)
  const [moduleName, setModuleName] = useState('')
  const [capabilityId, setCapabilityId] = useState('')
  const [preferredProtocol, setPreferredProtocol] = useState('rest')
  const [action, setAction] = useState('List')
  const [payloadText, setPayloadText] = useState('{}')

  const [invoking, setInvoking] = useState(false)
  const [status, setStatus] = useState('等待调用')
  const [traceId, setTraceId] = useState('—')
  const [resultText, setResultText] = useState('')
  const [errorText, setErrorText] = useState('')
  const [history, setHistory] = useState<InvokeHistory[]>([])

  const sourceOptions = useMemo(() => [
    { value: 'all', label: 'all' },
    { value: 'corex', label: 'corex · PowerX 底座' },
    { value: 'plugin', label: 'plugin · 插件能力' },
  ], [])

  const moduleOptions = useMemo<ModuleOption[]>(() => {
    const counts = new Map<string, number>()
    catalog.forEach((item) => {
      const m = pickModule(item)
      if (!m) return
      counts.set(m, (counts.get(m) || 0) + 1)
    })
    return [...counts.entries()]
      .map(([value, count]) => ({
        value,
        count,
        label: `${value} · ${count}`,
      }))
      .sort((a, b) => a.value.localeCompare(b.value))
  }, [catalog])

  const capabilityOptions = useMemo<CapabilityOption[]>(() => {
    return catalog
      .map((item) => {
        const capability = pickCapabilityId(item)
        const moduleValue = pickModule(item)
        const kind = normalizeKind(item)
        return {
          value: capability,
          module: moduleValue,
          kind,
          label: `${capability} · ${kind}`,
        }
      })
      .filter((item) => !!item.value)
      .filter((item) => !moduleName || item.module === moduleName)
      .sort((a, b) => a.value.localeCompare(b.value))
  }, [catalog, moduleName])

  const selectedCapabilityEntry = useMemo(() => {
    return catalog.find((item) => pickCapabilityId(item) === capabilityId) || null
  }, [catalog, capabilityId])

  const supportedProtocols = useMemo(() => {
    return extractSupportedProtocols(selectedCapabilityEntry)
  }, [selectedCapabilityEntry])

  useEffect(() => {
    const loadCatalog = async () => {
      setLoadingCatalog(true)
      setCatalogError('')
      try {
        const entries = await listCapabilitiesCatalog(source === 'all' ? '' : source)
        setCatalog((entries || []) as CatalogEntry[])
      } catch (err) {
        setCatalogError(err instanceof ApiError ? err.message : '能力目录加载失败')
      } finally {
        setLoadingCatalog(false)
      }
    }
    void loadCatalog()
  }, [source])

  useEffect(() => {
    if (!moduleOptions.length) {
      if (moduleName) setModuleName('')
      return
    }
    if (!moduleOptions.some((item) => item.value === moduleName)) {
      setModuleName(moduleOptions[0].value)
    }
  }, [moduleName, moduleOptions])

  useEffect(() => {
    if (!capabilityOptions.length) {
      if (capabilityId) setCapabilityId('')
      return
    }
    if (!capabilityOptions.some((item) => item.value === capabilityId)) {
      setCapabilityId(capabilityOptions[0].value)
    }
  }, [capabilityId, capabilityOptions])

  useEffect(() => {
    if (!supportedProtocols.length) return
    if (!supportedProtocols.includes(preferredProtocol)) {
      setPreferredProtocol(supportedProtocols.includes('rest') ? 'rest' : supportedProtocols[0])
    }
  }, [preferredProtocol, supportedProtocols])

  useEffect(() => {
    if (!supportedProtocols.length) return
    setPreferredProtocol(supportedProtocols.includes('rest') ? 'rest' : supportedProtocols[0])
  }, [capabilityId, supportedProtocols])

  useEffect(() => {
    if (!capabilityId || !preferredProtocol) return
    const nextAction = deriveDefaultAction(capabilityId, preferredProtocol)
    setAction(nextAction)
    setPayloadText(deriveDefaultPayload(capabilityId, preferredProtocol, nextAction))
  }, [capabilityId, preferredProtocol])

  const invoke = async () => {
    let payload: Record<string, unknown>
    try {
      payload = JSON.parse(payloadText) as Record<string, unknown>
    } catch {
      setErrorText('Payload 不是有效 JSON')
      setStatus('调用失败')
      return
    }

    setInvoking(true)
    setErrorText('')
    setResultText('')
    setStatus('调用中')

    try {
      const body = {
        capabilityId: capabilityId || undefined,
        action: action || undefined,
        preferredProtocol: preferredProtocol || undefined,
        payload,
      }
      const response = await invokeCapability(body)
      const nextTrace = String(response.traceId || response.trace_id || `trace-${Date.now()}`)
      const nextStatus = String(response.status || '成功')
      setTraceId(nextTrace)
      setStatus(nextStatus)
      setResultText(JSON.stringify(response, null, 2))
      setHistory((prev) => [
        {
          id: String(Date.now()),
          capabilityId,
          protocol: preferredProtocol,
          status: nextStatus,
          traceId: nextTrace,
          success: true,
          timestamp: nowLabel(),
        },
        ...prev,
      ].slice(0, 5))
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : '调用失败'
      setStatus('调用失败')
      setErrorText(msg)
      setHistory((prev) => [
        {
          id: String(Date.now()),
          capabilityId,
          protocol: preferredProtocol,
          status: '失败',
          traceId: '—',
          success: false,
          timestamp: nowLabel(),
        },
        ...prev,
      ].slice(0, 5))
    } finally {
      setInvoking(false)
    }
  }

  return (
    <main className="px-admin-page" data-testid="capability-lab-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <h1 className="px-admin-title">Capability Lab</h1>
          <p className="px-admin-subtitle">调试插件侧能力封装，验证宿主/Skeleton 对接 PowerX Gateway 的链路，并查看 TraceId、Mock 提示与契约告警。</p>

          <div className="px-cap-lab-grid" style={{ marginTop: 16 }}>
            <section className="px-admin-card" style={{ margin: 0 }}>
              <h2 className="px-admin-card-title" style={{ fontSize: 20 }}>调用配置</h2>

              <div style={{ marginTop: 12, display: 'grid', gap: 10 }}>
                <label>
                  <div className="px-admin-card-text">能力来源（source）</div>
                  <select className="px-input" value={source} onChange={(e) => setSource(e.target.value)}>
                    {sourceOptions.map((item) => (
                      <option key={item.value} value={item.value}>{item.label}</option>
                    ))}
                  </select>
                </label>

                <label>
                  <div className="px-admin-card-text">Capability 模块</div>
                  <select className="px-input" value={moduleName} onChange={(e) => setModuleName(e.target.value)}>
                    {moduleOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                  </select>
                </label>

                <label>
                  <div className="px-admin-card-text">Capability ID *</div>
                  <select className="px-input" value={capabilityId} onChange={(e) => setCapabilityId(e.target.value)}>
                    {capabilityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                  </select>
                </label>
                <p className="px-admin-card-text">
                  模块 {moduleName || '-'} · 共 {capabilityOptions.length} 个能力
                </p>
                <p className="px-admin-card-text">
                  支持协议：
                  {supportedProtocols.map((item) => (
                    <span key={item} className="px-cap-tag" style={{ marginLeft: 8 }}>{protocolLabel(item)}</span>
                  ))}
                </p>

                <label>
                  <div className="px-admin-card-text">协议（preferredProtocol）*</div>
                  <select className="px-input" value={preferredProtocol} onChange={(e) => setPreferredProtocol(e.target.value)}>
                    {supportedProtocols.map((item) => (
                      <option key={item} value={item}>{protocolLabel(item)}</option>
                    ))}
                  </select>
                </label>

                <label>
                  <div className="px-admin-card-text">Action *</div>
                  <input className="px-input" value={action} onChange={(e) => setAction(e.target.value)} />
                </label>

                <label>
                  <div className="px-admin-card-text">Payload JSON *</div>
                  <textarea className="px-code" style={{ width: '100%', minHeight: 220, marginTop: 6 }} value={payloadText} onChange={(e) => setPayloadText(e.target.value)} />
                </label>

                {catalogError ? <p className="px-alert px-alert-danger">{catalogError}</p> : null}
                {loadingCatalog ? <p className="px-admin-card-text">加载能力目录中...</p> : null}

                <div className="px-admin-toolbar" style={{ marginTop: 0 }}>
                  <button type="button" className="px-btn" disabled={invoking || !capabilityId} onClick={() => void invoke()}>
                    {invoking ? '调用中...' : '开始调用'}
                  </button>
                </div>
              </div>
            </section>

            <section style={{ display: 'grid', gap: 14, alignContent: 'start' }}>
              <article className="px-admin-card" style={{ margin: 0 }}>
                <h2 className="px-admin-card-title" style={{ fontSize: 20 }}>调用结果</h2>
                <p style={{ marginTop: 10 }}>状态：<strong>{status}</strong></p>
                <p>Trace ID：<strong>{traceId}</strong></p>
                {errorText ? <p className="px-alert px-alert-danger" style={{ marginTop: 10 }}>{errorText}</p> : null}
                <pre className="px-code" style={{ marginTop: 10, minHeight: 180 }}>{resultText || '暂无结果'}</pre>
              </article>

              <article className="px-admin-card" style={{ margin: 0 }}>
                <h2 className="px-admin-card-title" style={{ fontSize: 20 }}>最近记录</h2>
                {history.length === 0 ? (
                  <p className="px-admin-card-text">调用记录会显示在这里，最多保留最近 5 条。</p>
                ) : (
                  <div style={{ marginTop: 10, display: 'grid', gap: 8 }}>
                    {history.map((item) => (
                      <div key={item.id} className="px-admin-card" style={{ margin: 0, padding: 10 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
                          <strong>{item.capabilityId || '—'}</strong>
                          <span>{item.timestamp}</span>
                        </div>
                        <p style={{ marginTop: 6 }}>协议：{item.protocol} · 状态：{item.status} · Trace：{item.traceId}</p>
                      </div>
                    ))}
                  </div>
                )}
              </article>
            </section>
          </div>
        </article>
      </section>
    </main>
  )
}
