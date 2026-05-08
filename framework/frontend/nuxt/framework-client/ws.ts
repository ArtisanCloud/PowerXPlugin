export interface PluginWsOptions {
  pluginId: string;
  apiBaseURL?: string;
  wsBaseURL?: string;
  hostBaseURL?: string;
  wsPath?: string;
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
  const wsBase = String(options.wsBaseURL || "").trim();
  const hostBase = String(options.hostBaseURL || "").trim();
  const wsPathRaw = String(options.wsPath || "/api/ws").trim() || "/api/ws";

  const buildURL = () => {
    if (typeof window === "undefined") return "";
    let url: URL;
    if (wsBase) {
      const parsedBase = new URL(wsBase, window.location.origin);
      const wsProtocol = parsedBase.protocol === "https:" ? "wss:" : parsedBase.protocol;
      url = new URL(`${wsProtocol}//${parsedBase.host}${wsPathRaw}`);
    } else if (options.insidePowerX) {
      // Embedded mode prefers explicit hostBaseURL, but falls back to current origin
      // to avoid hard-fail when page-level runtime options are missing.
      const resolvedHostBase = hostBase || window.location.origin;
      const parsedHost = new URL(resolvedHostBase, window.location.origin);
      const wsProtocol = parsedHost.protocol === "https:" ? "wss:" : parsedHost.protocol;
      url = new URL(`${wsProtocol}//${parsedHost.host}${wsPathRaw}`);
    } else {
      const parsed = new URL(base, window.location.origin);
      const protocol = parsed.protocol === "https:" ? "wss:" : "ws:";
      const wsPath = normalizeWsPath(parsed.pathname || "");
      url = new URL(`${protocol}//${parsed.host}${wsPath}`);
    }
    const token = String(options.token || "").trim();
    const tenant = String(options.tenantUuid || "").trim();
    if (token) {
      const authValue = /^Bearer\s/i.test(token) ? token : `Bearer ${token}`;
      url.searchParams.set("authorization", authValue);
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
