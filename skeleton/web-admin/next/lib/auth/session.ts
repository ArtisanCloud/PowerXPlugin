const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const EXPIRES_AT_KEY = 'expires_at'

function isBrowser(): boolean {
  return typeof window !== 'undefined'
}

export function setSessionTokens(accessToken: string, refreshToken: string, expiresAt: number): void {
  if (!isBrowser()) return
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  localStorage.setItem(EXPIRES_AT_KEY, String(expiresAt))
  document.cookie = `${ACCESS_TOKEN_KEY}=${accessToken}; Path=/; SameSite=Lax`
}

export function clearSessionTokens(): void {
  if (!isBrowser()) return
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(EXPIRES_AT_KEY)
  document.cookie = `${ACCESS_TOKEN_KEY}=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Lax`
}

export function getAccessToken(): string | null {
  if (!isBrowser()) return null
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function getRefreshToken(): string | null {
  if (!isBrowser()) return null
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function getExpiresAt(): number | null {
  if (!isBrowser()) return null
  const raw = localStorage.getItem(EXPIRES_AT_KEY)
  if (!raw) return null
  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

export function isTokenExpired(now = Date.now()): boolean {
  const expiresAt = getExpiresAt()
  if (!expiresAt) return true
  return now >= expiresAt * 1000
}
