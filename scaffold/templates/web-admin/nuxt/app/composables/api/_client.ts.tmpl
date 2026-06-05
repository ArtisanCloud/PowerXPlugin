// 统一创建 $fetch 实例（单例）+ 便捷方法

import { resolveApiBase, getAuthToken, getTenantUuid } from "./_base";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";
import {
  createPowerXApiFetchClient,
  requestPowerXHostAuthToken,
  type PowerXApiClientRequestOptions,
  type PowerXApiFetchClient,
} from "@artisan-cloud/plugin-framework-client";
import { useRouter, useRuntimeConfig, useToast } from "#imports";
import { useHostCtxStore } from "~/stores/hostCtx";
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
  const router = useRouter();
  const toast = process.client ? useToast() : null;
  const hostCtxStore = process.client ? useHostCtxStore() : null;
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX =
    runtimeConfig.public?.insidePowerX === true ||
    runtimeConfig.public?.insidePowerX === "true";
  _baseURL = baseURL;

  const resolveAuth = async () => {
    if (!process.client) return null;
    const mod = await import("~/composables/useAuth");
    return mod.useAuth();
  };

  const readErrorPayload = (response?: { _data?: any }) => {
    const data = response?._data || {};
    const code = String(data?.error?.code || data?.code || "").trim();
    const message = String(
      data?.error?.message || data?.message || data?.error || ""
    ).trim();
    return { code, message };
  };

  const isDelegatedAuthUnavailable = (response?: { status?: number; _data?: any }) => {
    if (!response || response.status !== 503) return false;
    const { code, message } = readErrorPayload(response);
    if (!message) return false;
    const lower = message.toLowerCase();
    if (lower.includes("宿主认证") || lower.includes("delegated auth")) return true;
    return code === "SERVICE_UNAVAILABLE" && lower.includes("auth");
  };

  const isStsAuthError = (response?: { _data?: any }) => {
    const { code } = readErrorPayload(response);
    return code === "AUTH_STS_EXPIRED";
  };

  const isSessionAuthError = (response?: { _data?: any }) => {
    const { code } = readErrorPayload(response);
    return [
      "AUTH_TOKEN_INVALID",
      "AUTH_TOKEN_EXPIRED",
      "AUTH_STS_EXPIRED",
      "TOKEN_EXPIRED",
      "TOKEN_INVALID",
    ].includes(code);
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
    const auth = await resolveAuth();
    if (!auth) return;
    if (isDelegatedAuthUnavailable(response)) {
      logger.error("api delegated auth unavailable", {
        status: response.status,
        data: response._data,
      });
      const { message } = readErrorPayload(response);
      auth.failClosed?.(message || "宿主认证不可用，请稍后重试");
      if (process.client && !insidePowerX) {
        const redirect = window.location.pathname + window.location.search;
        router?.push({ path: "/users/login", query: { redirect } });
      }
      return;
    }
    if (response.status !== 401) return;

    const stsExpired = isStsAuthError(response);
    const sessionAuthError = isSessionAuthError(response);
    const hasAuthHeader = Boolean(
      prepared?.headers instanceof Headers
        ? prepared.headers.get("Authorization")
        : (prepared?.headers as any)?.Authorization
    );
    const onLoginPage =
      process.client &&
      /\/users\/login(?:$|[?#/])/i.test(window.location.pathname);
    if (!insidePowerX && hasAuthHeader && sessionAuthError) {
      auth.clearAuth?.("token_invalid", true);
    }
    if (process.client && !onLoginPage) {
      toast?.add?.({
        title: sessionAuthError
          ? stsExpired
            ? "STS 令牌已过期"
            : "登录状态已失效"
          : "请求未授权",
        description: !hasAuthHeader
          ? "该请求缺少 Authorization，已保留当前登录态"
          : stsExpired
          ? "短期 STS 令牌已失效，请重新登录刷新上下文"
          : sessionAuthError
            ? "请重新登录后再试"
            : `当前账号不能访问该接口：${String(request || "")}`,
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
    getFallbackToken: getAuthToken,
    getTenantUuid: () => resolveTenantUUIDForRequest() || getTenantUuid(),
    getHostCtx: (key) => hostCtxStore?.getCtx(key),
    requestHostToken: () => requestPowerXHostAuthToken(PLUGIN_ID),
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
