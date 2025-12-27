import { ref } from 'vue'
import { useToast } from '#imports'
import { usePowerXCapability } from '~/composables/usePowerXCapability'
import type {
  PowerXCapabilityBridgeError,
  PowerXCapabilityResponse
} from '~/plugins/powerx-capability.client'

export interface CapabilityLabHistoryEntry {
  id: string
  capabilityId: string
  action: string
  payloadText: string
  traceId?: string | null
  duration: number
  success: boolean
  warnings?: string[]
  error?: string
  rawText?: string
}

export interface CapabilityLabInvokeRequest {
  capabilityId: string
  action: string
  payload: Record<string, any>
  payloadText: string
  headers?: Record<string, string>
  requestId?: string
  apiBase?: string
  preferredProtocol?: string
}

const now = () => {
  if (typeof performance !== 'undefined' && typeof performance.now === 'function') {
    return performance.now()
  }
  return Date.now()
}

const generateHistoryId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `hist-${Date.now()}`
}

const normalizeWarnings = (value: string[] | string | null | undefined) => {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.filter(Boolean)
  }
  if (typeof value === 'string') {
    return value.trim() ? [value] : []
  }
  return []
}

export const useCapabilityLab = () => {
  const toast = useToast()
  const { invoke, loading } = usePowerXCapability()

  const result = ref<PowerXCapabilityResponse | null>(null)
  const warnings = ref<string[]>([])
  const errorMessage = ref('')
  const lastTraceId = ref<string | null>(null)
  const durationMs = ref<number | null>(null)
  const history = ref<CapabilityLabHistoryEntry[]>([])
  const errorDetails = ref<any>(null)

  const safeStringify = (value: any) => {
    if (value === null || value === undefined) return ''
    if (typeof value === 'string') return value
    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return String(value)
    }
  }

  const pushHistory = (
    request: CapabilityLabInvokeRequest,
    overrides: Omit<
      CapabilityLabHistoryEntry,
      'id' | 'capabilityId' | 'action' | 'payloadText' | 'rawText'
    > & { rawText?: string }
  ) => {
    history.value = [
      {
        id: generateHistoryId(),
        capabilityId: request.capabilityId,
        action: request.action,
        payloadText: request.payloadText,
        rawText: overrides.rawText,
        ...overrides
      },
      ...history.value
    ].slice(0, 5)
  }

  const clearHistory = () => {
    history.value = []
  }

  const invokeCapability = async (request: CapabilityLabInvokeRequest) => {
    const started = now()
    try {
      const response = await invoke(request.capabilityId, request.action, request.payload, {
        headers: request.headers,
        requestId: request.requestId,
        apiBase: request.apiBase,
        preferredProtocol: request.preferredProtocol,
        notifyOnSuccess: false,
        notifyOnError: false
      })

      durationMs.value = now() - started
      result.value = response
      warnings.value = normalizeWarnings(response.warnings)
      lastTraceId.value = response.traceId ?? null
      errorMessage.value = ''
      errorDetails.value = null
      pushHistory(request, {
        success: true,
        traceId: response.traceId,
        warnings: warnings.value,
        duration: durationMs.value ?? 0,
        rawText: safeStringify(response.raw ?? response.data)
      })
      toast.add({
        title: '调用成功',
        description: response.traceId ? `TraceId: ${response.traceId}` : undefined,
        color: 'green'
      })
    } catch (err: any) {
      const bridgeError = err as PowerXCapabilityBridgeError
      durationMs.value = now() - started
      result.value = null
      const warningList = normalizeWarnings(bridgeError?.warnings)
      warnings.value = warningList
      lastTraceId.value = bridgeError?.traceId ?? null
      errorMessage.value = bridgeError?.message || '能力调用失败'
      errorDetails.value = bridgeError?.details ?? null
      pushHistory(request, {
        success: false,
        traceId: bridgeError?.traceId,
        error: errorMessage.value,
        warnings: warningList,
        duration: durationMs.value ?? 0,
        rawText: safeStringify(bridgeError?.details)
      })
      toast.add({
        title: '调用失败',
        description: bridgeError?.traceId
          ? `TraceId: ${bridgeError.traceId}`
          : bridgeError?.message,
        color: 'red'
      })
      throw err
    }
  }

  return {
    invokeCapability,
    clearHistory,
    loading,
    result,
    warnings,
    errorMessage,
    lastTraceId,
    durationMs,
    history,
    errorDetails
  }
}
