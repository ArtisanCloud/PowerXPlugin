'use client'

import { useEffect, useState } from 'react'
import { getCapabilityRegister } from '@/lib/api/capabilities'
import { ApiError } from '@/lib/api/normalizeApiError'

export default function CapabilitiesRegisterPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getCapabilityRegister()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载注册信息失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main style={{ padding: 24 }} data-testid="capability-register-page">
      <h1>Capabilities 注册</h1>
      {error ? <p role="alert">{error}</p> : null}
      <pre data-testid="capability-register-json">{JSON.stringify(data || {}, null, 2)}</pre>
    </main>
  )
}
