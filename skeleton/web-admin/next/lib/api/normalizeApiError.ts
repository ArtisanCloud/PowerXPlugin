export type ApiErrorShape = {
  code?: number
  message?: string
  details?: unknown
}

export class ApiError extends Error {
  code: number
  details?: unknown

  constructor(message: string, code = 500, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.details = details
  }
}

export async function normalizeApiError(response: Response): Promise<ApiError> {
  let payload: ApiErrorShape | null = null

  try {
    payload = (await response.json()) as ApiErrorShape
  } catch {
    payload = null
  }

  const message = payload?.message || `HTTP ${response.status}`
  const code = payload?.code ?? response.status

  return new ApiError(message, code, payload?.details)
}
