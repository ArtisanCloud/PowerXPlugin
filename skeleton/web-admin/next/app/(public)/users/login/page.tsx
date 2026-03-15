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
    <main style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 16, background: 'linear-gradient(145deg, #eff6ff 0%, #ffffff 40%, #ecfeff 100%)' }}>
      <div style={{ width: 'min(460px, 100%)', background: '#fff', border: '1px solid #e2e8f0', borderRadius: 16, boxShadow: '0 25px 50px rgba(15, 23, 42, 0.12)', padding: 24 }}>
        <h1 style={{ marginTop: 0 }}>PowerX 登录</h1>
        <p style={{ color: '#475569' }}>登录后可进入 Next 管理端并联调 Gin 契约接口。</p>

        {error ? (
          <p role="alert" data-testid="login-error" style={{ color: '#b91c1c', background: '#fef2f2', border: '1px solid #fecaca', borderRadius: 8, padding: 10 }}>
            {error}
          </p>
        ) : null}

        {delegatedMode ? (
          <p role="alert" style={{ color: '#92400e', background: '#fffbeb', border: '1px solid #fde68a', borderRadius: 8, padding: 10 }}>
            委托鉴权模式下，本地用户名密码登录不可用。
          </p>
        ) : null}

        <form onSubmit={handleSubmit} style={{ display: 'grid', gap: 12 }}>
          <label>
            账号
            <input
              data-testid="login-username"
              name="identifier"
              value={identifier}
              disabled={loading || delegatedMode}
              onChange={(event) => setIdentifier(event.target.value)}
              style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }}
            />
          </label>

          <label>
            密码
            <input
              data-testid="login-password"
              name="password"
              type="password"
              value={password}
              disabled={loading || delegatedMode}
              onChange={(event) => setPassword(event.target.value)}
              style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }}
            />
          </label>

          <button type="submit" data-testid="login-submit" disabled={loading || delegatedMode} style={{ padding: '10px 14px' }}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>

        <div style={{ marginTop: 12, display: 'flex', gap: 12, fontSize: 14 }}>
          <Link href="/users/forgot-password">忘记密码</Link>
          <Link href="/users/register">注册账号</Link>
        </div>
      </div>
    </main>
  )
}
