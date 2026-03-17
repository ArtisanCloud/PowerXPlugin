'use client'

import { useEffect, useState } from 'react'
import { getOperationsOverview } from '@/lib/api/operations'
import { ApiError } from '@/lib/api/normalizeApiError'

export default function OperationsPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getOperationsOverview()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载 Operations 失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main className="px-admin-page" data-testid="operations-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <h1 className="px-admin-title">Operations</h1>
          <p className="px-admin-subtitle">运营指标与运行状态。</p>
          {error ? <p role="alert" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>{error}</p> : null}
          <pre className="px-code" data-testid="operations-json" style={{ marginTop: 14 }}>{JSON.stringify(data || {}, null, 2)}</pre>
        </article>
      </section>
    </main>
  )
}
