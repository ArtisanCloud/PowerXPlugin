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
    <main style={{ padding: 24 }} data-testid="operations-page">
      <h1>Operations</h1>
      {error ? <p role="alert">{error}</p> : null}
      <pre data-testid="operations-json">{JSON.stringify(data || {}, null, 2)}</pre>
    </main>
  )
}
