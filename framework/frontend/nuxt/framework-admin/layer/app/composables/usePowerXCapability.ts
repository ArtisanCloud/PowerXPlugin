import { useNuxtApp } from '#app'

interface PowerXCapabilityRequest {
  capabilityId: string
  action: string
  payload?: Record<string, any> | null
  headers?: Record<string, string>
  requestId?: string
  signal?: AbortSignal
  apiBase?: string
  endpoint?: string
  preferredProtocol?: string
  metadata?: Record<string, any>
}

interface PowerXCapabilityResponse {
  traceId?: string
  status?: string
  data?: Record<string, any> | null
  errors?: Record<string, any> | Record<string, any>[] | null
  warnings?: string[] | null
  raw?: Record<string, any> | Record<string, any>[] | string | null
}

interface PowerXCapabilityBridge {
  invoke(request: PowerXCapabilityRequest): Promise<PowerXCapabilityResponse>
}

export interface UsePowerXCapabilityOptions extends Partial<Omit<PowerXCapabilityRequest, 'capabilityId' | 'action' | 'payload'>> {
  payload?: Record<string, any> | null
}

export type UsePowerXCapabilityInvoker = (
  capabilityId: string,
  action: string,
  payload?: Record<string, any> | null,
  options?: UsePowerXCapabilityOptions
) => Promise<PowerXCapabilityResponse>

export const usePowerXCapability = (bridge?: PowerXCapabilityBridge): UsePowerXCapabilityInvoker => {
  const nuxtApp = useNuxtApp()
  const client = bridge ?? nuxtApp.$powerxCapability
  if (!client) {
    throw new Error('PowerX capability bridge is not initialized. Ensure the powerx-capability plugin is registered.')
  }

  return async (capabilityId, action, payload, options) => {
    const mergedPayload = options?.payload ?? payload ?? {}

    return client.invoke({
      capabilityId,
      action,
      payload: mergedPayload,
      headers: options?.headers,
      requestId: options?.requestId,
      signal: options?.signal,
      apiBase: options?.apiBase,
      endpoint: options?.endpoint
    })
  }
}
