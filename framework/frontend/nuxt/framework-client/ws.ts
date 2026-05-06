export interface PluginWsOptions {
  pluginId: string;
  apiBaseURL?: string;
  insidePowerX?: boolean;
  token?: string;
  tenantUuid?: string;
}

export interface PluginWsClient {
  buildURL(): string;
  connect(): WebSocket;
}

function normalizeWsPath(pathname: string) {
  const clean = pathname.replace(/\/+$/, "");
  if (!clean || clean === "/") return "/api/ws";
  if (clean.endsWith("/api/v1")) return clean.replace(/\/api\/v1$/, "/api/ws");
  if (clean.endsWith("/api")) return `${clean}/ws`;
  return "/api/ws";
}

export function createPluginWsClient(options: PluginWsOptions): PluginWsClient {
  const base = (options.apiBaseURL || `/_p/${options.pluginId}/api/v1`).trim();

  const buildURL = () => {
    if (typeof window === "undefined") return "";
    const parsed = new URL(base, window.location.origin);
    const protocol = parsed.protocol === "https:" ? "wss:" : "ws:";
    const wsPath = normalizeWsPath(parsed.pathname || "");
    const url = new URL(`${protocol}//${parsed.host}${wsPath}`);
    const token = String(options.token || "").trim();
    const tenant = String(options.tenantUuid || "").trim();
    if (token) {
      url.searchParams.set(
        "authorization",
        /^Bearer\s/i.test(token) ? token : `Bearer ${token}`
      );
    }
    if (tenant) {
      url.searchParams.set("tenant_uuid", tenant);
    }
    return url.toString();
  };

  return {
    buildURL,
    connect: () => new WebSocket(buildURL()),
  };
}

