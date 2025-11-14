// 统一创建 $fetch 实例（单例）+ 便捷方法

import { resolveApiBase, getAuthToken, getTenantId } from "./_base";
import { useAuth } from "~/composables/useAuth";
import { useRouter } from "#imports";

type Json = Record<string, any>;

let _client: typeof $fetch | null = null;
let _baseURL: string | null = null;

export function useApiClient() {
  if (_client) return { client: _client, baseURL: _baseURL! };

  const baseURL = resolveApiBase();
  const router = useRouter();
  _baseURL = baseURL;

  const client = $fetch.create({
    baseURL,
    timeout: 30_000,
    async onRequest({ options }) {
      const headers = (options.headers ||= {}) as Record<string, string>;
      const skipAuth = Boolean((options as any).skipAuth);
      const auth = useAuth();

      if (!("Accept" in headers)) headers["Accept"] = "application/json";

      const isFormData =
        options.body &&
        typeof FormData !== "undefined" &&
        options.body instanceof FormData;

      if (!isFormData && !("Content-Type" in headers)) {
        headers["Content-Type"] = "application/json";
      }

      let authToken: string | null = null;
      if (!skipAuth) {
        authToken = (options as any).authToken || (options as any).token;
        if (!authToken) {
          authToken = (await auth.ensureFreshToken()) || getAuthToken();
        }
      }

      if (authToken && !headers["Authorization"]) {
        headers["Authorization"] = /^Bearer\\s/i.test(String(authToken))
          ? String(authToken)
          : `Bearer ${authToken}`;
      }

      // 多租户
      if (!headers["X-Tenant-ID"]) {
        const tenant = (options as any).tenantId || getTenantId();
        if (tenant) headers["X-Tenant-ID"] = String(tenant);
      }
    },
    onResponseError({ response }) {
      console.error("API error:", response.status, response._data);
      const auth = useAuth();
      if (response.status === 503) {
        const message = response._data?.message || "宿主认证不可用，请稍后重试";
        auth.failClosed?.(message);
        if (process.client) {
          const redirect = window.location.pathname + window.location.search;
          router?.push({ path: "/users/login", query: { redirect } });
        }
        return;
      }
      if (response.status === 401) {
        auth.clearAuth();
      }
    },
  });

  _client = client;
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
