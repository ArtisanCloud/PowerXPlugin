'use client'

import { useEffect, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { getIamSettings } from '@/lib/api/iam'

export default function IamSettingsPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getIamSettings()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载 IAM 设置失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main style={{ padding: 24 }} data-testid="iam-settings-page">
      <h1>IAM 设置</h1>
      {error ? <p role="alert">{error}</p> : null}
      <pre data-testid="iam-settings-json">{JSON.stringify(data || {}, null, 2)}</pre>
    </main>
  )
}
