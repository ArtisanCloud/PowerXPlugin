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
    <main style={{ padding: 24 }} data-testid="security-page">
      <h1>Security</h1>
      {error ? <p role="alert">{error}</p> : null}
      <pre data-testid="security-json">{JSON.stringify(data || {}, null, 2)}</pre>
    </main>
  )
}
