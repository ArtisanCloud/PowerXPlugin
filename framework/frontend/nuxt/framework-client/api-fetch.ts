import { getPowerXHostSession, normalizeBearerToken, PowerXHostSessionError, type PowerXHostSession, type PowerXHostSessionSnapshot } from "./host-session";
import { classifyPowerXApiAuthError } from './api-errors';

export type PowerXRequestParams = Record<string, any> | URLSearchParams;

export type PowerXApiClientRequestOptions = Record<string, any> & {
  params?: PowerXRequestParams;
  silentAuthError?: boolean;
};

export type PowerXApiAuthAdapter = {
  getToken?: () => string | null | undefined;
  ensureFreshToken?: () => Promise<string | null | undefined>;
  clearAuth?: () => void;
  failClosed?: (message?: string) => void;
};

export type PowerXApiFetchClient = {
  <T>(request: any, options?: PowerXApiClientRequestOptions): Promise<T>;
  raw<T>(request: any, options?: PowerXApiClientRequestOptions): Promise<Response & { _data?: T }>;
  native: typeof fetch;
  request<T>(request: any, options?: PowerXApiClientRequestOptions): Promise<T>;
  get<T>(request: any, options?: PowerXApiClientRequestOptions): Promise<T>;
  delete<T>(request: any, options?: PowerXApiClientRequestOptions): Promise<T>;
  post<T>(request: any, data?: any, options?: PowerXApiClientRequestOptions): Promise<T>;
  put<T>(request: any, data?: any, options?: PowerXApiClientRequestOptions): Promise<T>;
  patch<T>(request: any, data?: any, options?: PowerXApiClientRequestOptions): Promise<T>;
};

export interface CreatePowerXApiFetchClientOptions {
  fetcher: any;
  baseURL: string;
  pluginId: string;
  insidePowerX?: boolean;
  getAuth?: () => Promise<PowerXApiAuthAdapter | null> | PowerXApiAuthAdapter | null;
  getFallbackToken?: () => string | null | undefined;
  getTenantUuid?: () => string | null | undefined;
  hostSession?: PowerXHostSession;
  onAuthError?: (input: {
    response?: { status?: number; _data?: any };
    request?: any;
    prepared?: Record<string, any>;
  }) => Promise<void> | void;
  logger?: {
    info?: (...args: any[]) => void;
    error?: (...args: any[]) => void;
  };
}

function searchParamsToObject(params: URLSearchParams) {
  const obj: Record<string, string> = {};
  params.forEach((value, key) => {
    obj[key] = value;
  });
  return obj;
}

export function normalizePowerXRequestOptions(
  options?: PowerXApiClientRequestOptions | null,
): Record<string, any> | undefined {
  if (!options) return options || undefined;

  const next: Record<string, any> = { ...options };
  if (next.query instanceof URLSearchParams) {
    next.query = searchParamsToObject(next.query);
  } else if (next.query && typeof next.query === "object") {
    next.query = { ...next.query };
  }

  if (next.params) {
    const params =
      next.params instanceof URLSearchParams
        ? searchParamsToObject(next.params)
        : { ...next.params };
    next.query = { ...(next.query || {}), ...params };
    delete next.params;
  }
  return next;
}

function isFormData(value: any): value is FormData {
  return typeof FormData !== "undefined" && value instanceof FormData;
}

function isBlob(value: any): value is Blob {
  return typeof Blob !== "undefined" && value instanceof Blob;
}

function isArrayBuffer(value: any): value is ArrayBuffer {
  return typeof ArrayBuffer !== "undefined" && value instanceof ArrayBuffer;
}

function isUrlEncoded(value: any): value is URLSearchParams {
  return typeof URLSearchParams !== "undefined" && value instanceof URLSearchParams;
}

export function normalizePowerXBodyPayload(body: any) {
  if (body === undefined) return undefined;
  if (
    typeof body === "string" ||
    isFormData(body) ||
    isBlob(body) ||
    isArrayBuffer(body) ||
    isUrlEncoded(body)
  ) {
    return body;
  }
  return JSON.stringify(body);
}

function applyHttpShortcuts(client: PowerXApiFetchClient) {
  client.request = <T>(request: any, options?: PowerXApiClientRequestOptions) =>
    client(request, options) as Promise<T>;

  client.get = <T>(request: any, options?: PowerXApiClientRequestOptions) =>
    client(request, { ...(options || {}), method: "GET" }) as Promise<T>;

  client.delete = <T>(request: any, options?: PowerXApiClientRequestOptions) =>
    client(request, { ...(options || {}), method: "DELETE" }) as Promise<T>;

  const withBody =
    (method: "POST" | "PUT" | "PATCH") =>
    <T>(request: any, data?: any, options?: PowerXApiClientRequestOptions) => {
      const init: Record<string, any> = { ...(options || {}), method };
      const body = normalizePowerXBodyPayload(data);
      if (body !== undefined) init.body = body;
      return client(request, init) as Promise<T>;
    };

  client.post = withBody("POST");
  client.put = withBody("PUT");
  client.patch = withBody("PATCH");
  return client;
}

function isJsonBody(body: any) {
  return body !== undefined && !isFormData(body);
}

function hostRequestSignal(session: PowerXHostSession, snapshot: PowerXHostSessionSnapshot, external?: AbortSignal) {
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (external?.aborted) abort();
  external?.addEventListener('abort', abort, { once: true });
  const unsubscribe = session.subscribe(() => { if (!session.isCurrent(snapshot)) abort(); });
  return { signal: controller.signal, dispose: () => { unsubscribe(); external?.removeEventListener('abort', abort); } };
}

async function preparePowerXRequest(options: CreatePowerXApiFetchClientOptions, request: any, input?: Record<string, any>) {
  const next: Record<string, any> = input ? { ...input } : {};
  const headers = new Headers(input?.headers as HeadersInit | undefined);
  next.headers = headers;
  if (!headers.has('Accept')) headers.set('Accept', 'application/json');
  if (isJsonBody(next.body) && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  if (options.insidePowerX) {
    const path = String(request?.url || request);
    if (/\/admin\/user\/auth\/(login|refresh|logout)(?:$|[?#])/.test(path)) throw new PowerXHostSessionError('AUTH_SESSION_HOST_MANAGED');
    if (path.startsWith('//') || (/^[a-z][a-z0-9+.-]*:/i.test(path) && !/^https?:\/\//i.test(path))) throw new PowerXHostSessionError('PLUGIN_API_SCOPE_INVALID');
    const origin = window.location.origin;
    const base = new URL(options.baseURL.replace(/\/$/, '') + '/', origin);
    if (base.pathname !== `/_p/${options.pluginId}/api/v1/`) throw new PowerXHostSessionError('PLUGIN_API_SCOPE_INVALID');
    const target = new URL(/^https?:\/\//.test(path) ? path : base.href + path.replace(/^\//, ''), origin);
    if (target.origin !== base.origin || !target.pathname.startsWith(base.pathname)) throw new PowerXHostSessionError('PLUGIN_API_SCOPE_INVALID');
    const session = options.hostSession || getPowerXHostSession(options.pluginId);
    headers.delete('X-PowerX-CTX'); headers.delete('X-PowerX-CTX-SIG');
    headers.delete('X-PowerX-Identity-Context');
    if (next.skipAuth) { headers.delete('Authorization'); return { prepared: next, session, snapshot: null }; }
    let snapshot: PowerXHostSessionSnapshot;
    if (next.hostSnapshot) {
      snapshot = next.hostSnapshot;
      if (!session.isContextSnapshot(snapshot) || !/\/admin\/user\/auth\/me\/context$/.test(target.pathname)) throw new PowerXHostSessionError('AUTH_CONTEXT_REQUEST_INVALID');
    } else snapshot = await session.ensureSnapshot({ signal: input?.signal });
    const tenantUuid = snapshot.identity ? snapshot.identity.tenantUuid : snapshot.tenantUuid;
    if (next.tenantUuid && next.tenantUuid !== tenantUuid) throw new PowerXHostSessionError('AUTH_TENANT_MISMATCH');
    headers.set('Authorization', normalizeBearerToken(snapshot.accessToken));
    headers.delete('tenant_uuid');
    if (tenantUuid) headers.set('tenant_uuid', tenantUuid);
    delete next.authToken; delete next.token; delete next.hostSnapshot;
    next.retry = 0;
    return { prepared: next, session, snapshot };
  }
  if (!next.skipAuth) {
    const auth = await options.getAuth?.();
    let token = next.authToken || next.token || auth?.getToken?.();
    if (!token) token = await auth?.ensureFreshToken?.();
    if (!token) token = options.getFallbackToken?.();
    if (token && !headers.has('Authorization')) headers.set('Authorization', normalizeBearerToken(String(token)));
  }
  if (!headers.has('tenant_uuid')) {
    const tenantUuid = next.tenantUuid || options.getTenantUuid?.();
    if (tenantUuid) headers.set('tenant_uuid', String(tenantUuid));
  }
  return { prepared: next, session: null, snapshot: null };
}

export function createPowerXApiFetchClient(options: CreatePowerXApiFetchClientOptions) {
  const baseClient = options.fetcher.create({ baseURL: options.baseURL, timeout: 30_000 });
  const invoke = async (request: any, rawOptions?: PowerXApiClientRequestOptions, raw = false) => {
    const normalized = normalizePowerXRequestOptions(rawOptions);
    let original: PowerXHostSessionSnapshot | null = null;
    for (let attempt = 0; attempt < 2; attempt++) {
      const { prepared, session, snapshot } = await preparePowerXRequest(options, request, normalized);
      if (original && !session!.isSameSession(original)) throw new PowerXHostSessionError('AUTH_SESSION_CHANGED');
      original ||= snapshot;
      const connection = session && snapshot ? hostRequestSignal(session, snapshot, normalized?.signal) : null;
      if (connection) {
        prepared.signal = connection.signal;
        if (connection.signal.aborted) { connection.dispose(); throw new PowerXHostSessionError('AUTH_REQUEST_ABORTED'); }
      }
      try {
        const response = await (raw ? baseClient.raw(request, prepared) : baseClient(request, prepared));
        if (snapshot && (!session!.isCurrent(snapshot) || snapshot.expiresAt <= Date.now())) throw new PowerXHostSessionError('AUTH_SESSION_CHANGED');
        return response;
      } catch (error: any) {
        connection?.dispose();
        if (snapshot && !session!.isSameSession(snapshot)) throw new PowerXHostSessionError('AUTH_SESSION_CHANGED');
        if (session && snapshot && classifyPowerXApiAuthError(error?.response).kind === 'user-session-expired' && attempt === 0 && !normalized?.hostSnapshot && !prepared.skipAuth && !normalized?.signal?.aborted) {
          const current = session.getReadySnapshot();
          if (!current || current === snapshot) await session.ensureSnapshot({ force: true, expected: snapshot, signal: normalized?.signal });
          continue;
        }
        if (snapshot && !session!.isCurrent(snapshot)) throw new PowerXHostSessionError('AUTH_SESSION_CHANGED');
        prepared.__powerxHostSnapshot = snapshot;
        if (!normalized?.hostSnapshot) await options.onAuthError?.({ response: error?.response, request, prepared });
        if (snapshot && !session!.isCurrent(snapshot)) throw new PowerXHostSessionError('AUTH_SESSION_CHANGED');
        const failure = classifyPowerXApiAuthError(error?.response);
        if (session && snapshot && (failure.kind === 'user-session-invalid' || failure.kind === 'user-session-expired')) session.invalidate('PLUGIN_AUTH_REQUIRED');
        throw error;
      } finally { connection?.dispose(); }
    }
    throw new PowerXHostSessionError('PLUGIN_AUTH_REQUIRED');
  };
  const client = ((request: any, requestOptions?: PowerXApiClientRequestOptions) => invoke(request, requestOptions)) as PowerXApiFetchClient;
  client.raw = (request: any, requestOptions?: PowerXApiClientRequestOptions) => invoke(request, requestOptions, true);
  client.native = options.insidePowerX ? client.raw : baseClient.native;
  applyHttpShortcuts(client);
  return { client, baseClient };
}
