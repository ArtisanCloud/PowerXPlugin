'use client'

import { useEffect, useState } from 'react'
import { getSecurityOverview } from '@/lib/api/operations'
import { ApiError } from '@/lib/api/normalizeApiError'

export default function SecurityPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getSecurityOverview()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载 Security 失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main className="px-admin-page" data-testid="security-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <h1 className="px-admin-title">Security</h1>
          <p className="px-admin-subtitle">安全概览与策略状态。</p>
          {error ? <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>{error}</p> : null}
          <pre className="px-code" data-testid="security-json" style={{ marginTop: 14 }}>{JSON.stringify(data || {}, null, 2)}</pre>
        </article>
      </section>
    </main>
  )
}
