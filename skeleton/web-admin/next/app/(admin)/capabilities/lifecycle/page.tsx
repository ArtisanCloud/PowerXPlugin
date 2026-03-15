'use client'

import { useEffect, useState } from 'react'
import { getCapabilityLifecycle } from '@/lib/api/capabilities'
import { ApiError } from '@/lib/api/normalizeApiError'

export default function CapabilitiesLifecyclePage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getCapabilityLifecycle()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载生命周期失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main style={{ padding: 24 }} data-testid="capability-lifecycle-page">
      <h1>Capabilities 生命周期</h1>
      {error ? <p role="alert">{error}</p> : null}
      <pre data-testid="capability-lifecycle-json">{JSON.stringify(data || {}, null, 2)}</pre>
    </main>
  )
}
