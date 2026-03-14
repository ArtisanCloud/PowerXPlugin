'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { ApiError } from '@/lib/api/normalizeApiError'
import { register } from '@/lib/api/auth'

export default function RegisterPage() {
  const router = useRouter()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [agree, setAgree] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [countdown, setCountdown] = useState(3)

  useEffect(() => {
    if (!success || countdown <= 0) return
    const timer = window.setTimeout(() => setCountdown((prev) => prev - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [success, countdown])

  useEffect(() => {
    if (success && countdown <= 0) {
      router.push('/users/login')
    }
  }, [success, countdown, router])

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!username.trim() || !email.trim() || !password) {
      setError('请填写完整注册信息。')
      return
    }
    if (password !== confirmPassword) {
      setError('两次密码输入不一致。')
      return
    }
    if (!agree) {
      setError('请先阅读并同意条款。')
      return
    }

    setLoading(true)
    setError('')
    try {
      await register({
        username: username.trim(),
        email: email.trim(),
        password,
        display_name: username.trim(),
      })
      setSuccess(true)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('注册失败，请稍后重试。')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <main style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 16 }}>
      <div style={{ width: 'min(480px, 100%)', border: '1px solid #e2e8f0', borderRadius: 16, background: '#fff', padding: 24 }}>
        <h1 style={{ marginTop: 0 }}>注册账号</h1>
        {error ? <p role="alert" style={{ color: '#b91c1c' }}>{error}</p> : null}

        {success ? (
          <div data-testid="register-success">
            <p>注册成功，{countdown} 秒后跳转登录页。</p>
            <Link href="/users/login">立即前往登录</Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit} style={{ display: 'grid', gap: 12 }}>
            <label>
              用户名
              <input data-testid="register-username" value={username} onChange={(event) => setUsername(event.target.value)} style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }} />
            </label>
            <label>
              邮箱
              <input data-testid="register-email" type="email" value={email} onChange={(event) => setEmail(event.target.value)} style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }} />
            </label>
            <label>
              密码
              <input data-testid="register-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }} />
            </label>
            <label>
              确认密码
              <input data-testid="register-confirm-password" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }} />
            </label>
            <label style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input data-testid="register-agree" type="checkbox" checked={agree} onChange={(event) => setAgree(event.target.checked)} />
              我已阅读并同意条款
            </label>
            <button data-testid="register-submit" type="submit" disabled={loading} style={{ padding: '10px 14px' }}>
              {loading ? '提交中...' : '注册'}
            </button>
          </form>
        )}
      </div>
    </main>
  )
}
