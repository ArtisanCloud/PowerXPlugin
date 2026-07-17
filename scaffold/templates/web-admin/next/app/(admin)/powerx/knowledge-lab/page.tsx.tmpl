'use client'

import { useEffect, useMemo, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { getKnowledgeProvider, searchKnowledge, type KnowledgeProviderInfo, type KnowledgeSearchResult } from '@/lib/api/knowledge'

function formatJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2)
}

export default function KnowledgeLabPage() {
  const [provider, setProvider] = useState<KnowledgeProviderInfo | null>(null)
  const [providerError, setProviderError] = useState('')
  const [loadingProvider, setLoadingProvider] = useState(true)

  const [tenantUUID, setTenantUUID] = useState('')
  const [query, setQuery] = useState('PowerX framework knowledge provider')
  const [topK, setTopK] = useState(5)
  const [filtersText, setFiltersText] = useState('{}')

  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState('')
  const [result, setResult] = useState<KnowledgeSearchResult | null>(null)

  const providerSummary = useMemo(() => {
    if (!provider) return '未连接'
    return [provider.provider, provider.source, provider.mode].filter(Boolean).join(' / ') || '已连接'
  }, [provider])

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getKnowledgeProvider()
        if (active) setProvider(payload)
      } catch (err) {
        if (!active) return
        setProviderError(err instanceof ApiError ? err.message : '加载知识库 Provider 失败')
      } finally {
        if (active) setLoadingProvider(false)
      }
    })()
    return () => {
      active = false
    }
  }, [])

  async function runSearch() {
    setSearchError('')
    setResult(null)

    const trimmed = query.trim()
    if (!trimmed) {
      setSearchError('请输入检索问题')
      return
    }

    let filters: Record<string, unknown> = {}
    try {
      filters = filtersText.trim() ? JSON.parse(filtersText) as Record<string, unknown> : {}
    } catch {
      setSearchError('Filters 必须是合法 JSON')
      return
    }

    setSearching(true)
    try {
      const payload = await searchKnowledge({
        tenant_uuid: tenantUUID.trim() || undefined,
        query: trimmed,
        top_k: topK,
        filters,
      })
      setResult(payload)
    } catch (err) {
      setSearchError(err instanceof ApiError ? err.message : '知识库检索失败')
    } finally {
      setSearching(false)
    }
  }

  return (
    <main className="px-admin-page" data-testid="knowledge-lab-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <h1 className="px-admin-title">Knowledge Lab</h1>
          <p className="px-admin-subtitle">调试 framework knowledge provider，验证 local/mock/delegated 检索链路与返回结构。</p>

          <div className="px-cap-lab-grid" style={{ marginTop: 16 }}>
            <section className="px-admin-card" style={{ margin: 0 }}>
              <h2 className="px-admin-card-title" style={{ fontSize: 20 }}>检索配置</h2>

              <div style={{ marginTop: 12, display: 'grid', gap: 10 }}>
                <label>
                  <div className="px-admin-card-text">Tenant UUID</div>
                  <input className="px-input" value={tenantUUID} onChange={(event) => setTenantUUID(event.target.value)} placeholder="可选；生产 delegated 场景建议传入" />
                </label>

                <label>
                  <div className="px-admin-card-text">Query *</div>
                  <textarea className="px-code" style={{ width: '100%', minHeight: 110, marginTop: 6 }} value={query} onChange={(event) => setQuery(event.target.value)} />
                </label>

                <label>
                  <div className="px-admin-card-text">Top K</div>
                  <input className="px-input" type="number" min={1} max={20} value={topK} onChange={(event) => setTopK(Number(event.target.value) || 5)} />
                </label>

                <label>
                  <div className="px-admin-card-text">Filters JSON</div>
                  <textarea className="px-code" style={{ width: '100%', minHeight: 130, marginTop: 6 }} value={filtersText} onChange={(event) => setFiltersText(event.target.value)} />
                </label>

                {searchError ? <p className="px-alert px-alert-danger">{searchError}</p> : null}

                <div className="px-admin-toolbar" style={{ marginTop: 0 }}>
                  <button type="button" className="px-btn" disabled={searching} onClick={() => void runSearch()}>
                    {searching ? '检索中...' : '开始检索'}
                  </button>
                </div>
              </div>
            </section>

            <section style={{ display: 'grid', gap: 14, alignContent: 'start' }}>
              <article className="px-admin-card" style={{ margin: 0 }}>
                <h2 className="px-admin-card-title" style={{ fontSize: 20 }}>Provider</h2>
                <p style={{ marginTop: 10 }}>状态：<strong>{loadingProvider ? '加载中...' : providerSummary}</strong></p>
                {providerError ? <p className="px-alert px-alert-danger" style={{ marginTop: 10 }}>{providerError}</p> : null}
                <pre className="px-code" style={{ marginTop: 10, minHeight: 160 }}>{formatJSON(provider)}</pre>
              </article>

              <article className="px-admin-card" style={{ margin: 0 }}>
                <h2 className="px-admin-card-title" style={{ fontSize: 20 }}>检索结果</h2>
                <pre className="px-code" style={{ marginTop: 10, minHeight: 260 }}>{result ? formatJSON(result) : '暂无结果'}</pre>
              </article>
            </section>
          </div>
        </article>
      </section>
    </main>
  )
}
