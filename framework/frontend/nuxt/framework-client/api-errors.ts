export interface PowerXApiErrorResponse {
  status?: number;
  _data?: unknown;
}

export type PowerXApiAuthErrorKind =
  | 'user-session-expired'
  | 'user-session-invalid'
  | 'identity-proof'
  | 'service-credential'
  | 'identity-unavailable'
  | 'forbidden'
  | 'unknown';

export interface PowerXApiAuthError {
  kind: PowerXApiAuthErrorKind;
  code?: string;
}

function record(value: unknown): Record<string, unknown> | undefined {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : undefined;
}

/** Read structured Core/Framework error fields, never human message text. */
export function readPowerXApiErrorCode(response?: PowerXApiErrorResponse): string | undefined {
  const data = record(response?._data);
  const error = record(data?.error) || data;
  const details = record(error?.details);
  for (const value of [details?.reason_code, error?.reason_code, details?.code, error?.error_code, error?.code]) {
    if (typeof value === 'string' && value.length > 0) return value;
  }
  return undefined;
}

/** An upstream identity/service failure does not invalidate the browser session. */
export function classifyPowerXApiAuthError(response?: PowerXApiErrorResponse): PowerXApiAuthError {
  const code = readPowerXApiErrorCode(response);
  let kind: PowerXApiAuthErrorKind = 'unknown';
  if (response?.status === 401) {
    switch (code) {
      case 'AUTH_TOKEN_EXPIRED':
      case 'TOKEN_EXPIRED':
      case 'IAM_CONTEXT_USER_SESSION_EXPIRED':
        kind = 'user-session-expired';
        break;
      case 'AUTH_TOKEN_INVALID':
      case 'TOKEN_INVALID':
      case 'IAM_CONTEXT_USER_SESSION_INVALID':
        kind = 'user-session-invalid';
        break;
      case 'IAM_CONTEXT_PROOF_REQUIRED':
      case 'IAM_CONTEXT_PROOF_INVALID':
      case 'IAM_CONTEXT_PROOF_EXPIRED':
      case 'IAM_CONTEXT_WS_HANDLE_REQUIRED':
      case 'IAM_CONTEXT_WS_HANDLE_INVALID':
      case 'IAM_CONTEXT_WS_HANDLE_EXPIRED':
        kind = 'identity-proof';
        break;
      case 'AUTH_STS_EXPIRED':
      case 'IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID':
      case 'IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED':
        kind = 'service-credential';
        break;
    }
  } else if (response?.status === 403) {
    kind = 'forbidden';
  } else if (response?.status === 503 || response?.status === 424) {
    if (code === 'IAM_CONTEXT_UNAVAILABLE' || code === 'IAM_UPSTREAM_DEPENDENCY') kind = 'identity-unavailable';
  }
  return { kind, code };
}
