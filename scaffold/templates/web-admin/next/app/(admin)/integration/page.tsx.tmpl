'use client'

import { useEffect, useState } from 'react'
import { getIntegrationSettings } from '@/lib/api/operations'
import { ApiError } from '@/lib/api/normalizeApiError'

export default function IntegrationPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getIntegrationSettings()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载 Integration 失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main className="px-admin-page" data-testid="integration-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <h1 className="px-admin-title">Integration</h1>
          <p className="px-admin-subtitle">外部集成配置与联通状态。</p>
          {error ? <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>{error}</p> : null}
          <pre className="px-code" data-testid="integration-json" style={{ marginTop: 14 }}>{JSON.stringify(data || {}, null, 2)}</pre>
        </article>
      </section>
    </main>
  )
}
