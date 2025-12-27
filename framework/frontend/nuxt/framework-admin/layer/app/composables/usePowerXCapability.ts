import { useNuxtApp } from '#app'
import type {
  PowerXCapabilityBridge,
  PowerXCapabilityRequest,
  PowerXCapabilityResponse
} from '../plugins/powerx-capability.client'

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
