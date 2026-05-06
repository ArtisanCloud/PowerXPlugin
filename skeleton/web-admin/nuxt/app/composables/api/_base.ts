// 解析 API 基址 + 获取 Token/Tenant 的小工具

export function resolveApiBase(pathname?: string): string {
  const runtimePublic =
    (typeof useRuntimeConfig === "function"
      ? (useRuntimeConfig() as any)?.public
      : undefined) ?? (globalThis as any).__NUXT__?.config?.public;

  if (runtimePublic?.apiBaseUrl) {
    return runtimePublic.apiBaseUrl;
  }

  const p =
    pathname ??
    (typeof window !== "undefined" ? window.location.pathname : "") ??
    "";

  // 识别 PowerX：支持 `/<locale>/_p/<plugin-id>/admin/...`
  const segments = p.split("/").filter(Boolean);
  const idx = segments.indexOf("_p");
  if (idx >= 0) {
    const pluginId = segments[idx + 1];
    const scope = segments[idx + 2];
    if (pluginId && scope === "admin") {
      return `/_p/${pluginId}/api/v1`;
    }
  }

  return "/api/v1";
}

export function getAuthToken(): string | undefined {
  const runtimePublic =
    (typeof useRuntimeConfig === "function"
      ? (useRuntimeConfig() as any)?.public
      : undefined) ?? (globalThis as any).__NUXT__?.config?.public;
  const insidePowerX =
    runtimePublic?.insidePowerX === true || runtimePublic?.insidePowerX === "true";

  // 优先读取 localStorage，保持与 useAuth/Bridge 注入兼容
  if (typeof window !== "undefined") {
    try {
      const localStorageCandidates = [
        "access_token",
        "__px_access_token",
        "token",
      ];
      for (const key of localStorageCandidates) {
        const storedToken = window.localStorage?.getItem(key);
        if (storedToken) {
          if ((window as any).__PX_AUTH_DEBUG__) {
            console.info("[api/_base] getAuthToken from localStorage", {
              key,
              prefix: String(storedToken).slice(0, 24),
              insidePowerX,
            });
          }
          return storedToken;
        }
      }
    } catch (error) {
      console.warn("[PowerXPlugin] failed to read localStorage auth token", error);
    }
  }

  // cookie 回退：
  // - insidePowerX: 允许宿主上下文 cookie
  // - standalone: 仅使用本地 token cookie（由 useAuth.setAuth 写入）
  if (typeof document !== "undefined") {
    const cookieCandidates = insidePowerX
      ? ["px_ctx_jwt", "token"]
      : ["token"];
    for (const name of cookieCandidates) {
      const match = document.cookie.match(
        new RegExp(`(?:^|;\\s*)${name}=([^;]+)`, "i")
      );
      if (match) {
        const token = decodeURIComponent(match[1]);
        if (typeof window !== "undefined" && (window as any).__PX_AUTH_DEBUG__) {
          console.info("[api/_base] getAuthToken from cookie", {
            key: name,
            prefix: String(token).slice(0, 24),
            insidePowerX,
          });
        }
        return token;
      }
    }
  }

  return undefined;
}

export function getTenantUuid(): string | undefined {
  // TODO: 换成你的 Pinia/Cookie 逻辑
  if (typeof document !== "undefined") {
    const m = document.cookie.match(/(?:^|;\s*)tenant_uuid=([^;]+)/);
    if (m) return decodeURIComponent(m[1]);
  }
  const cfg =
    typeof useRuntimeConfig === "function" ? useRuntimeConfig() : ({} as any);
  const publicCfg = cfg.public as any;
  return publicCfg?.defaultTenantUuid || publicCfg?.defaultTenantId;
}

// 通用类型定义
export interface Page<T> {
  list: T[];
  page_index: number;
  page_size: number;
  total: number;
  // 兼容旧字段，方便前端在未调整完之前继续使用
  items?: T[];
  page?: number;
  limit?: number;
}

export interface ApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  code?: number;
}

export interface ListQuery {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  order?: "asc" | "desc";
}
