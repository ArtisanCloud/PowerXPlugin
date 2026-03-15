'use client'

import { useEffect, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { getIamOverview } from '@/lib/api/iam'

export default function IamOverviewPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await getIamOverview()
        if (active) setData(payload)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载 IAM 概览失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main style={{ padding: 24 }} data-testid="iam-overview-page">
      <h1>IAM 概览</h1>
      {error ? <p role="alert">{error}</p> : null}
      <pre data-testid="iam-overview-json">{JSON.stringify(data || {}, null, 2)}</pre>
    </main>
  )
}
