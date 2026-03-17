const API_PREFIX = '/api/v1'

export type RuntimeMode = 'host' | 'standalone'

export function isInsidePowerX(pathname?: string): boolean {
  if (!pathname) return false
  return pathname.startsWith('/_p/')
}

export function resolveRuntimeMode(pathname?: string): RuntimeMode {
  return isInsidePowerX(pathname) ? 'host' : 'standalone'
}

export function resolveApiBase(pathname?: string): string {
  if (!pathname) return API_PREFIX

  const hostMatch = pathname.match(/^\/_p\/([^/]+)/)
  if (hostMatch) {
    return `/_p/${hostMatch[1]}${API_PREFIX}`
  }

  return API_PREFIX
}
