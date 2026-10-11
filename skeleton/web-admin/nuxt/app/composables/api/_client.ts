// 统一创建 $fetch 实例（单例）+ 便捷方法

import { resolveApiBase, getAuthToken, getTenantUuid } from "./_base";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";
import {
  createPowerXApiFetchClient,
  classifyPowerXApiAuthError,
  getPowerXHostSession,
  isPowerXEmbedded,
  type PowerXApiClientRequestOptions,
  type PowerXApiFetchClient,
} from "@artisan-cloud/plugin-framework-client";
import { useRuntimeConfig, useToast } from "#imports";
import { PLUGIN_ID } from "~/utils/powerx-bridge";
import { createFrameworkLogger } from "@artisan-cloud/plugin-framework-client";

type Json = Record<string, any>;

let _client: PowerXApiFetchClient | null = null;
let _baseURL: string | null = null;
let _clientEnv: "client" | "server" | null = null;
const logger = createFrameworkLogger("api.client");

export function useApiClient() {
  const env = typeof window === "undefined" ? "server" : "client";
  if (_client && _clientEnv === env) {
    return {
      client: _client,
      baseURL: _baseURL!,
      request: _client.request,
      get: _client.get,
      post: _client.post,
      put: _client.put,
      patch: _client.patch,
      delete: _client.delete,
    };
  }

  const baseURL = resolveApiBase();
  const toast = process.client ? useToast() : null;
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX = isPowerXEmbedded(runtimeConfig.public?.insidePowerX);
  const translate = (key: string) => useNuxtApp().$i18n.t(key);
  _baseURL = baseURL;

  const resolveAuth = async () => {
    if (!process.client) return null;
    const mod = await import("~/composables/useAuth");
    return mod.useAuth();
  };

  const handleAuthError = async ({
    response,
    request,
    prepared,
  }: {
    response?: { status?: number; _data?: any };
    request?: any;
    prepared?: Record<string, any>;
  }) => {
    if (!response || prepared?.silentAuthError) return;
    if (insidePowerX && !getPowerXHostSession(PLUGIN_ID).isCurrent(prepared?.__powerxHostSnapshot)) return;
    const { kind, code } = classifyPowerXApiAuthError(response);
    if (response.status !== 401 && kind !== 'identity-unavailable') return;
    const sessionAuthError = kind === 'user-session-expired' || kind === 'user-session-invalid';
    const hasAuthHeader = Boolean(
      prepared?.headers instanceof Headers
        ? prepared.headers.get("Authorization")
        : (prepared?.headers as any)?.Authorization
    );
    const onLoginPage =
      process.client &&
      /\/users\/login(?:$|[?#/])/i.test(window.location.pathname);
    if (!insidePowerX && hasAuthHeader && sessionAuthError) {
      const auth = await resolveAuth();
      auth?.clearAuth?.("token_invalid", true);
    }
    logger.error('api auth request rejected', { status: response.status, code });
    if (process.client && !onLoginPage) {
      const descriptionKey = sessionAuthError ? 'hostSession.required'
        : kind === 'identity-proof' ? 'hostSession.identityProofRejected'
        : kind === 'service-credential' ? 'hostSession.serviceCredentialRejected'
        : kind === 'identity-unavailable' ? 'hostSession.upstreamUnavailable'
        : 'hostSession.requestFailed';
      toast?.add?.({
        title: translate('hostSession.requestRejected'),
        description: translate(descriptionKey),
        color: "red",
      });
    }
  };

  const { client } = createPowerXApiFetchClient({
    fetcher: $fetch,
    baseURL,
    pluginId: PLUGIN_ID,
    insidePowerX,
    getAuth: resolveAuth,
    getFallbackToken: insidePowerX ? undefined : getAuthToken,
    getTenantUuid: () => resolveTenantUUIDForRequest() || getTenantUuid(),
    onAuthError: handleAuthError,
    logger,
  });

  _client = client;
  _clientEnv = env;
  return {
    client,
    baseURL,
    request: client.request,
    get: client.get,
    post: client.post,
    put: client.put,
    patch: client.patch,
    delete: client.delete,
  };
}

// 常用 CRUD 便捷封装
export function apiGet<T>(path: string, query?: Json, init?: any) {
  const { client } = useApiClient();
  return client<T>(path, { method: "GET", query, ...init });
}
export function apiPost<T>(path: string, body?: any, init?: any) {
  const { client } = useApiClient();
  const payload =
    body instanceof FormData ? body : body ? JSON.stringify(body) : undefined;
  return client<T>(path, { method: "POST", body: payload, ...init });
}
export function apiPut<T>(path: string, body?: any, init?: any) {
  const { client } = useApiClient();
  const payload =
    body instanceof FormData ? body : body ? JSON.stringify(body) : undefined;
  return client<T>(path, { method: "PUT", body: payload, ...init });
}
export function apiPatch<T>(path: string, body?: any, init?: any) {
  const { client } = useApiClient();
  const payload =
    body instanceof FormData ? body : body ? JSON.stringify(body) : undefined;
  return client<T>(path, { method: "PATCH", body: payload, ...init });
}
export function apiDel<T>(path: string, init?: any) {
  const { client } = useApiClient();
  return client<T>(path, { method: "DELETE", ...init });
}
