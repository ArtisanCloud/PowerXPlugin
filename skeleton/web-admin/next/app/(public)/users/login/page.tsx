'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useEffect, useMemo, useState } from 'react'
import { login } from '@/lib/api/auth'
import { ApiError } from '@/lib/api/normalizeApiError'
import { setSessionTokens } from '@/lib/auth/session'
import { setAuthState } from '@/lib/stores/templates'

function sanitizeRedirectTo(raw: string | null, insidePowerX: boolean): string {
  if (!raw || !raw.trim()) return '/intro'
  const redirect = raw.trim()

  if (/^https?:\/\//i.test(redirect)) return '/intro'

  const pathOnly = redirect.startsWith('/') ? redirect : `/${redirect}`
  if (/\/users\/login(?:$|[?#/])/i.test(pathOnly)) return '/intro'

  if (!insidePowerX) {
    const match = pathOnly.match(/^\/_p\/[^/]+\/admin(\/.*)?$/)
    if (match) {
      const stripped = match[1] || '/intro'
      if (/\/users\/login(?:$|[?#/])/i.test(stripped)) return '/intro'
      return stripped
    }
  }

  return pathOnly
}

export default function LoginPage() {
  const router = useRouter()
  const pathname = usePathname()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [redirectRaw, setRedirectRaw] = useState<string | null>(null)
  const [remember, setRemember] = useState(false)

  const delegatedMode = process.env.NEXT_PUBLIC_DELEGATED_IAM === '1'
  const insidePowerX = pathname.startsWith('/_p/')

  useEffect(() => {
    const url = new URL(window.location.href)
    setRedirectRaw(url.searchParams.get('redirect'))
  }, [])

  const redirectTo = useMemo(
    () => sanitizeRedirectTo(redirectRaw, insidePowerX),
    [insidePowerX, redirectRaw]
  )

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (delegatedMode) {
      setError('当前为委托鉴权模式，已禁用本地登录。')
      return
    }
    if (!identifier.trim() || !password) {
      setError('请输入账号和密码。')
      return
    }

    setLoading(true)
    setError('')

    try {
      const data = await login({
        tenant: '',
        identifier: identifier.trim(),
        password,
        remember,
      })

      setSessionTokens(data.access_token, data.refresh_token, data.expires_at)
      setAuthState({
        isAuthenticated: true,
        accessToken: data.access_token,
        expiresAt: data.expires_at,
      })

      router.push(redirectTo)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('登录失败，请稍后重试。')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="px-auth-page">
      <div className="px-auth-shell">
        <Link href="/" className="px-auth-back">&lt; 返回首页</Link>

        <div className="px-auth-card">
          <header className="px-auth-header">
            <h1 className="px-auth-logo">PowerX</h1>
            <p className="px-auth-subtitle">使用 PowerX 账号登录，继续管理插件。</p>
          </header>

          <div className="px-auth-body">
            {error ? (
              <p role="alert" data-testid="login-error" className="px-alert px-alert-danger">
                {error}
              </p>
            ) : null}

            {delegatedMode ? (
              <p role="alert" className="px-alert px-alert-warning">
                委托鉴权模式下，本地用户名密码登录不可用。
              </p>
            ) : null}

            <form onSubmit={handleSubmit} className="px-form">
              <div className="px-form-row">
                <label className="px-form-label" htmlFor="identifier">邮箱或手机号</label>
                <input
                  data-testid="login-username"
                  id="identifier"
                  className="px-input"
                  name="identifier"
                  value={identifier}
                  disabled={loading || delegatedMode}
                  onChange={(event) => setIdentifier(event.target.value)}
                />
              </div>

              <div className="px-form-row">
                <label className="px-form-label" htmlFor="password">密码</label>
                <input
                  data-testid="login-password"
                  id="password"
                  className="px-input"
                  name="password"
                  type="password"
                  value={password}
                  disabled={loading || delegatedMode}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </div>

              <div className="px-auth-links">
                <label className="px-checkbox-row">
                  <input checked={remember} onChange={(event) => setRemember(event.target.checked)} type="checkbox" />
                  记住登录状态
                </label>
                <Link href="/users/forgot-password">忘记密码?</Link>
              </div>

              <button type="submit" data-testid="login-submit" className="px-auth-submit" disabled={loading || delegatedMode}>
                {loading ? '登录中...' : '登录'}
              </button>
            </form>

            <div className="px-auth-foot">
              还没有账号? <Link href="/users/register">立即注册</Link>
            </div>
          </div>
        </div>
      </div>
    </main>
  )
}
