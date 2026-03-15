'use client'

import { useEffect, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { listIamRoles, type IamRole } from '@/lib/api/iam'

export default function IamRolesPage() {
  const [items, setItems] = useState<IamRole[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await listIamRoles()
        if (active) setItems(payload.list)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载角色失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main style={{ padding: 24 }} data-testid="iam-roles-page">
      <h1>IAM 角色</h1>
      {error ? <p role="alert">{error}</p> : null}
      <ul data-testid="iam-roles-list">
        {items.map((item) => (
          <li key={item.id}>{item.name}</li>
        ))}
      </ul>
    </main>
  )
}
