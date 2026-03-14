export type HostBridgeContext = {
  insidePowerX: boolean
  pluginId?: string
}

export function parseHostContext(pathname?: string): HostBridgeContext {
  if (!pathname) return { insidePowerX: false }

  const match = pathname.match(/^\/_p\/([^/]+)/)
  if (!match) return { insidePowerX: false }

  return {
    insidePowerX: true,
    pluginId: match[1],
  }
}

export async function requestDelegatedToken(): Promise<string | null> {
  if (typeof window === 'undefined') return null

  try {
    const response = await fetch('/_p/_internal/sts/exchange', { method: 'POST' })
    if (!response.ok) return null
    const data = (await response.json()) as { token?: string }
    return data.token || null
  } catch {
    return null
  }
}
