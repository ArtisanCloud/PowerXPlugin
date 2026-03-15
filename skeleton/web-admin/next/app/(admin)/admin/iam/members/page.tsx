'use client'

import { useEffect, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { listIamMembers, type IamMember } from '@/lib/api/iam'

export default function IamMembersPage() {
  const [items, setItems] = useState<IamMember[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const payload = await listIamMembers()
        if (active) setItems(payload.list)
      } catch (err) {
        if (!active) return
        setError(err instanceof ApiError ? err.message : '加载成员失败')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <main style={{ padding: 24 }} data-testid="iam-members-page">
      <h1>IAM 成员</h1>
      {error ? <p role="alert">{error}</p> : null}
      <ul data-testid="iam-members-list">
        {items.map((item) => (
          <li key={item.id}>{item.display_name || item.username}</li>
        ))}
      </ul>
    </main>
  )
}
