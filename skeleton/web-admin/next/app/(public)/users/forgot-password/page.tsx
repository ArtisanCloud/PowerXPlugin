'use client'

import Link from 'next/link'
import { useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { requestPasswordReset } from '@/lib/api/auth'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [sent, setSent] = useState(false)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!email.trim()) {
      setError('请输入邮箱。')
      return
    }

    setLoading(true)
    setError('')
    try {
      await requestPasswordReset({ email: email.trim() })
      setSent(true)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('发送失败，请稍后重试。')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <main style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 16 }}>
      <div style={{ width: 'min(460px, 100%)', border: '1px solid #e2e8f0', borderRadius: 16, background: '#fff', padding: 24 }}>
        <h1 style={{ marginTop: 0 }}>忘记密码</h1>

        {sent ? (
          <div data-testid="forgot-success">
            <p>重置邮件已发送至 {email}，请查收邮箱。</p>
            <Link href="/users/login">返回登录</Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit} style={{ display: 'grid', gap: 12 }}>
            {error ? <p role="alert" style={{ color: '#b91c1c' }}>{error}</p> : null}
            <label>
              邮箱
              <input
                data-testid="forgot-email"
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }}
              />
            </label>
            <button data-testid="forgot-submit" type="submit" disabled={loading} style={{ padding: '10px 14px' }}>
              {loading ? '发送中...' : '发送重置邮件'}
            </button>
          </form>
        )}
      </div>
    </main>
  )
}
