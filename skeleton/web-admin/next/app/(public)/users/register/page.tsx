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
    <main className="px-auth-page">
      <div className="px-auth-shell">
        <Link href="/users/login" className="px-auth-back">&lt; 返回登录</Link>
        <div className="px-auth-card">
          <header className="px-auth-header">
            <h1 className="px-auth-logo">PowerX</h1>
            <p className="px-auth-subtitle">创建账号并接入插件管理。</p>
          </header>

          <div className="px-auth-body">
            {error ? <p role="alert" className="px-alert px-alert-danger">{error}</p> : null}

            {success ? (
              <div data-testid="register-success" className="px-auth-foot" style={{ marginTop: 0, borderTop: 'none', paddingTop: 0 }}>
                <p>注册成功，{countdown} 秒后跳转登录页。</p>
                <Link href="/users/login">立即前往登录</Link>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="px-form">
                <div className="px-form-row">
                  <label className="px-form-label">用户名</label>
                  <input className="px-input" data-testid="register-username" value={username} onChange={(event) => setUsername(event.target.value)} />
                </div>
                <div className="px-form-row">
                  <label className="px-form-label">邮箱</label>
                  <input className="px-input" data-testid="register-email" type="email" value={email} onChange={(event) => setEmail(event.target.value)} />
                </div>
                <div className="px-form-row">
                  <label className="px-form-label">密码</label>
                  <input className="px-input" data-testid="register-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
                </div>
                <div className="px-form-row">
                  <label className="px-form-label">确认密码</label>
                  <input className="px-input" data-testid="register-confirm-password" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
                </div>
                <label className="px-checkbox-row">
                  <input data-testid="register-agree" type="checkbox" checked={agree} onChange={(event) => setAgree(event.target.checked)} />
                  我已阅读并同意条款
                </label>
                <button data-testid="register-submit" className="px-auth-submit" type="submit" disabled={loading}>
                  {loading ? '提交中...' : '注册'}
                </button>
              </form>
            )}

            <div className="px-auth-foot">
              已有账号? <Link href="/users/login">立即登录</Link>
            </div>
          </div>
        </div>
      </div>
    </main>
  )
}
