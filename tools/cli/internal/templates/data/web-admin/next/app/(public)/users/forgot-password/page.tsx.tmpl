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
    <main className="px-auth-page">
      <div className="px-auth-shell">
        <Link href="/users/login" className="px-auth-back">&lt; 返回登录</Link>
        <div className="px-auth-card">
          <header className="px-auth-header">
            <h1 className="px-auth-logo">PowerX</h1>
            <p className="px-auth-subtitle">通过邮箱完成密码重置。</p>
          </header>

          <div className="px-auth-body">
            {sent ? (
              <div data-testid="forgot-success" className="px-auth-foot" style={{ marginTop: 0, borderTop: 'none', paddingTop: 0 }}>
                <p>重置邮件已发送至 {email}，请查收邮箱。</p>
                <Link href="/users/login">返回登录</Link>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="px-form">
                {error ? <p role="alert" className="px-alert px-alert-danger">{error}</p> : null}
                <div className="px-form-row">
                  <label className="px-form-label">邮箱</label>
                  <input
                    data-testid="forgot-email"
                    className="px-input"
                    type="email"
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                  />
                </div>
                <button data-testid="forgot-submit" className="px-auth-submit" type="submit" disabled={loading}>
                  {loading ? '发送中...' : '发送重置邮件'}
                </button>
              </form>
            )}

            <div className="px-auth-foot">
              <Link href="/users/login">返回登录</Link>
              {' · '}
              <Link href="/users/register">创建新账号</Link>
            </div>
          </div>
        </div>
      </div>
    </main>
  )
}
