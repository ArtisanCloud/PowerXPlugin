import { normalizeBearerToken, type PowerXHostCtx } from "./host-session";

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

export type PowerXApiFetchClient = any & {
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
  getHostCtx?: (key: string) => PowerXHostCtx | undefined;
  requestHostToken?: () => void;
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

function resolveCtxKey(pluginId: string) {
  if (typeof window === "undefined") return null;
  return `${window.location.origin}::${pluginId}`;
}

async function preparePowerXRequest(
  options: CreatePowerXApiFetchClientOptions,
  request: any,
  input?: Record<string, any>,
) {
  const next: Record<string, any> = input ? { ...input } : {};
  const headers =
    input?.headers instanceof Headers
      ? new Headers(input.headers)
      : new Headers((input?.headers as HeadersInit) || undefined);
  next.headers = headers;

  if (!headers.has("Accept")) headers.set("Accept", "application/json");
  if (isJsonBody(next.body) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let authToken: string | null | undefined = null;
  if (!next.skipAuth) {
    const auth = await options.getAuth?.();
    authToken = next.authToken || next.token || auth?.getToken?.();
    if (!authToken) authToken = await auth?.ensureFreshToken?.();
    if (!authToken) authToken = options.getFallbackToken?.();
  }
  if (authToken && !headers.has("Authorization")) {
    headers.set("Authorization", normalizeBearerToken(String(authToken)));
  }

  const ctxKey = resolveCtxKey(String(next.pluginId || options.pluginId));
  const ctx = ctxKey ? options.getHostCtx?.(ctxKey) : undefined;
  if (ctx?.ctx && !headers.has("X-PowerX-CTX")) {
    headers.set("X-PowerX-CTX", ctx.ctx);
  }
  if (ctx?.ctxSig && !headers.has("X-PowerX-CTX-SIG")) {
    headers.set("X-PowerX-CTX-SIG", ctx.ctxSig);
  }

  if (!headers.has("tenant_uuid")) {
    const tenantUuid = next.tenantUuid || options.getTenantUuid?.() || ctx?.tenantUuid;
    if (tenantUuid) headers.set("tenant_uuid", String(tenantUuid));
  }

  return { prepared: next, request };
}

export function createPowerXApiFetchClient(
  options: CreatePowerXApiFetchClientOptions,
) {
  const baseClient = options.fetcher.create({
    baseURL: options.baseURL,
    timeout: 30_000,
  });

  const invoke = async (
    request: any,
    rawOptions?: PowerXApiClientRequestOptions,
    raw = false,
  ) => {
    const normalized = normalizePowerXRequestOptions(rawOptions);
    const { prepared } = await preparePowerXRequest(options, request, normalized);
    try {
      return await (raw ? baseClient.raw(request, prepared) : baseClient(request, prepared));
    } catch (error: any) {
      const status = error?.response?.status;
      const retried = Boolean(prepared?.__powerxBridgeRetried);
      if (options.insidePowerX && status === 401 && !retried && options.requestHostToken) {
        options.requestHostToken();
        await new Promise((resolve) => setTimeout(resolve, 300));
        const retryOptions = {
          ...(normalized || {}),
          __powerxBridgeRetried: true,
        };
        const retry = await preparePowerXRequest(options, request, retryOptions);
        try {
          return await (raw
            ? baseClient.raw(request, retry.prepared)
            : baseClient(request, retry.prepared));
        } catch (retryError: any) {
          await options.onAuthError?.({
            response: retryError?.response,
            request,
            prepared: retry.prepared,
          });
          throw retryError;
        }
      }
      await options.onAuthError?.({
        response: error?.response,
        request,
        prepared,
      });
      throw error;
    }
  };

  const client = ((request: any, requestOptions?: PowerXApiClientRequestOptions) =>
    invoke(request, requestOptions)) as PowerXApiFetchClient;
  client.raw = (request: any, requestOptions?: PowerXApiClientRequestOptions) =>
    invoke(request, requestOptions, true);
  client.native = baseClient.native;
  applyHttpShortcuts(client);
  return { client, baseClient };
}
