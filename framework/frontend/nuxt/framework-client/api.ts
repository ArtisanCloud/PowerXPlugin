import { createPowerXApiFetchClient, normalizePowerXBodyPayload, type PowerXApiAuthAdapter } from './api-fetch';
import { isPowerXEmbedded } from './bridge';

export interface PluginApiOptions {
  pluginId: string; baseURL?: string; tenantUuid?: string; insidePowerX?: boolean;
  auth?: PowerXApiAuthAdapter;
}
export interface PluginApi {
  get<T>(path: string, init?: RequestInit): Promise<T>;
  post<T>(path: string, body: unknown, init?: RequestInit): Promise<T>;
  put<T>(path: string, body: unknown, init?: RequestInit): Promise<T>;
  delete<T>(path: string, init?: RequestInit): Promise<T>;
}

/** The native-fetch entrypoint shares the same session and retry contract. */
export function usePluginApi(options: PluginApiOptions): PluginApi {
  const prefix = (options.baseURL ?? `/_p/${options.pluginId}/api/v1`).replace(/\/$/, '');
  const transport = (raw: boolean) => async (route: string, init: Record<string, any>) => {
    const url = /^https?:\/\//.test(route) ? route : `${prefix}/${route.replace(/^\//, '')}`;
    const response = await fetch(url, init as RequestInit);
    let data: unknown;
    let parseError: unknown;
    if (response.status !== 204 && response.headers.get('content-type')?.includes('application/json')) {
      try { data = await response.clone().json(); } catch (cause) { parseError = cause; }
    }
    if (!response.ok || parseError) {
      const error = new Error(response.ok ? 'PLUGIN_API_RESPONSE_INVALID' : 'PLUGIN_API_HTTP_ERROR', { cause: parseError }) as Error & { status: number; response: Response & { _data: unknown } };
      error.status = response.status; error.response = Object.assign(response, { _data: data }); throw error;
    }
    if (raw) return response;
    if (response.status !== 204 && data === undefined) throw new Error('PLUGIN_API_RESPONSE_INVALID');
    return data;
  };
  const fetcher = transport(false) as any;
  fetcher.raw = transport(true); fetcher.native = fetch;
  const { client } = createPowerXApiFetchClient({
    pluginId: options.pluginId, baseURL: prefix, insidePowerX: isPowerXEmbedded(options.insidePowerX),
    fetcher: { create: () => fetcher }, getAuth: () => options.auth || null, getTenantUuid: () => options.tenantUuid,
  });
  const request = <T>(method: string, route: string, init?: RequestInit) => client<T>(route, { ...init, method });
  const body = (value: unknown) => value == null ? undefined : normalizePowerXBodyPayload(value);
  return {
    get: (route, init) => request('GET', route, init),
    post: (route, value, init) => request('POST', route, { ...init, body: body(value) }),
    put: (route, value, init) => request('PUT', route, { ...init, body: body(value) }),
    delete: (route, init) => request('DELETE', route, init),
  };
}
