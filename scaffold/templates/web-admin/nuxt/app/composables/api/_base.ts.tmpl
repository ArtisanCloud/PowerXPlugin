// 解析 API 基址 + 获取 Token/Tenant 的小工具

export function resolveApiBase(pathname?: string): string {
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

  // 兜底：runtimeConfig.public.apiBaseUrl
  const cfg =
    (globalThis as any).__NUXT__?.config?.public ??
    (typeof useRuntimeConfig === "function"
      ? (useRuntimeConfig() as any).public
      : undefined);

  return cfg?.apiBaseUrl || "http://localhost:8086/api/v1";
}

export function getAuthToken(): string | undefined {
  // TODO: 换成你的 Pinia/Cookie 逻辑
  if (typeof document !== "undefined") {
    const m = document.cookie.match(/(?:^|;\s*)token=([^;]+)/);
    if (m) return decodeURIComponent(m[1]);
  }
  return undefined;
}

export function getTenantId(): string | undefined {
  // TODO: 换成你的 Pinia/Cookie 逻辑
  if (typeof document !== "undefined") {
    const m = document.cookie.match(/(?:^|;\s*)tenant_id=([^;]+)/);
    if (m) return decodeURIComponent(m[1]);
  }
  const cfg =
    typeof useRuntimeConfig === "function" ? useRuntimeConfig() : ({} as any);
  return (cfg.public as any)?.defaultTenantId;
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
