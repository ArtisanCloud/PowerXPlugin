export interface PluginApiOptions {
  pluginId: string
  baseURL?: string
}

export interface PluginApi {
  get<T>(path: string, init?: RequestInit): Promise<T>
  post<T>(path: string, body: unknown, init?: RequestInit): Promise<T>
}

const defaultHeaders = {
  'Content-Type': 'application/json'
}

export function usePluginApi(options: PluginApiOptions): PluginApi {
  const prefix = (options.baseURL ?? `/_p/${options.pluginId}/api/v1`).replace(/\/$/, '')

  const request = async <T>(method: string, route: string, init?: RequestInit) => {
    const response = await fetch(`${prefix}${route}`, {
      method,
      headers: defaultHeaders,
      ...init
    })
    if (!response.ok) {
      let message = `request failed: ${response.status}`
      try {
        const body = await response.clone().json()
        if (typeof body.message === 'string') {
          message = body.message
        } else if (typeof body.error === 'string') {
          message = body.error
        }
      } catch {
        // ignore parsing errors, fall back to default message
      }
      const error = new Error(message)
      ;(error as any).status = response.status
      throw error
    }
    return (await response.json()) as T
  }

  return {
    get: (route, init) => request('GET', route, init),
    post: (route, body, init) =>
      request('POST', route, { body: JSON.stringify(body), ...init })
  }
}
