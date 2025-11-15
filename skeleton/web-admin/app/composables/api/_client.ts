// 统一创建 $fetch 实例（单例）+ 便捷方法

import { resolveApiBase, getAuthToken, getTenantId } from "./_base";
import { useAuth } from "~/composables/useAuth";
import { useRouter, useToast } from "#imports";

type Json = Record<string, any>;

let _client: typeof $fetch | null = null;
let _baseURL: string | null = null;
let _clientEnv: "client" | "server" | null = null;

export function useApiClient() {
  const env = typeof window === "undefined" ? "server" : "client";
  if (_client && _clientEnv === env) {
    return { client: _client, baseURL: _baseURL! };
  }

  const baseURL = resolveApiBase();
  const router = useRouter();
  _baseURL = baseURL;

  const baseClient = $fetch.create({
    baseURL,
    timeout: 30_000,
  });

  const prepareOptions = async (options?: Record<string, any>) => {
    const next: Record<string, any> = options ? { ...options } : {};
    const headers =
      options?.headers instanceof Headers
        ? new Headers(options.headers)
        : new Headers((options?.headers as HeadersInit) || undefined);
    next.headers = headers;
    const skipAuth = Boolean((options as any)?.skipAuth);
    const auth = useAuth();

    if (!headers.has("Accept")) {
      headers.set("Accept", "application/json");
    }

    const isFormData =
      next.body &&
      typeof FormData !== "undefined" &&
      next.body instanceof FormData;

    if (!isFormData && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    let authToken: string | null = null;
    if (!skipAuth) {
      authToken = (next as any).authToken || (next as any).token;
      if (!authToken) {
        authToken = (await auth.ensureFreshToken()) || getAuthToken();
      }
    }

    if (authToken && !headers.has("Authorization")) {
      headers.set(
        "Authorization",
        /^Bearer\\s/i.test(String(authToken))
          ? String(authToken)
          : `Bearer ${authToken}`
      );
    }

    if (!headers.has("X-Tenant-ID")) {
      const tenant = (next as any).tenantId || getTenantId();
      if (tenant) headers.set("X-Tenant-ID", String(tenant));
    }

    return next;
  };

  const toast = process.client ? useToast() : null;

  const handleAuthError = (response?: { status?: number; _data?: any }) => {
    if (!response) return;
    const auth = useAuth();
    if (response.status === 503) {
      console.error("API error:", response.status, response._data);
      const message = response._data?.message || "宿主认证不可用，请稍后重试";
      auth.failClosed?.(message);
      if (process.client) {
        const redirect = window.location.pathname + window.location.search;
        router?.push({ path: "/users/login", query: { redirect } });
      }
      return;
    }
    if (response.status === 401) {
      console.error("API error:", response.status, response._data);
      auth.clearAuth();
      if (process.client) {
        toast?.add?.({
          title: "登录状态已失效",
          description: "请重新登录后再试",
          color: "red",
        });
      }
    }
  };

  const invokeClient = async (request: any, options?: any, raw = false) => {
    const prepared = await prepareOptions(options);
    try {
      return await (raw
        ? baseClient.raw(request, prepared)
        : baseClient(request, prepared));
    } catch (error: any) {
      handleAuthError(error?.response);
      throw error;
    }
  };

  const client = ((request: any, options?: any) =>
    invokeClient(request, options)) as typeof baseClient;

  client.raw = (request: any, options?: any) => invokeClient(request, options, true);
  client.native = baseClient.native;

  _client = client;
  _clientEnv = env;
  return { client, baseURL };
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
