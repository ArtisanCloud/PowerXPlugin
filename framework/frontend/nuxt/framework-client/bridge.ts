export type PowerXThemeKey = "light" | "dark" | "system";

export interface PowerXSyncPayload {
  source: "powerx";
  type: "sync";
  locale: string;
  theme: PowerXThemeKey;
  hostOrigin?: string;
  pluginId?: string;
  instanceId?: string;
}

export interface PowerXThemePayload {
  source: "powerx";
  type: "theme";
  theme: PowerXThemeKey;
}

export interface PowerXLocalePayload {
  source: "powerx";
  type: "locale";
  locale: string;
}

export interface PowerXAuthTokenPayload {
  source: "powerx";
  type: "auth-token";
  accessToken?: string;
  access_token?: string;
  refreshToken?: string;
  refresh_token?: string;
  tokenType?: string;
  token_type?: string;
  expiresIn?: number;
  expires_in?: number;
  expiresAt?: number;
  expires_at?: number;
  scope?: string;
  pluginId?: string;
  plugin_id?: string;
  hostOrigin?: string;
  host_origin?: string;
  tenantUuid?: string;
  tenant_uuid?: string;
  ctx?: string;
  ctxSig?: string;
  x_powerx_ctx?: string;
  x_powerx_ctx_sig?: string;
}

export type PowerXHostMessage =
  | PowerXSyncPayload
  | PowerXThemePayload
  | PowerXLocalePayload
  | PowerXAuthTokenPayload;

export interface PowerXPluginReadyPayload {
  source: "plugin";
  type: "ready";
  pluginId?: string;
  instanceId?: string;
}

export interface PowerXPluginRequestSyncPayload {
  source: "plugin";
  type: "request-sync";
}

export interface PowerXPluginAuthTokenRequestPayload {
  source: "powerx-plugin";
  type: "auth-token:request";
  pluginId?: string;
  instanceId?: string;
}

export interface PowerXPluginPingPayload {
  source: "plugin";
  type: "ping";
  ts: number;
}

export type PowerXFullscreenAction = "enter" | "exit" | "toggle";

export interface PowerXPluginFullscreenPayload {
  source: "powerx-plugin";
  type: "fullscreen:request";
  action: PowerXFullscreenAction;
  pluginId?: string;
  instanceId?: string;
  route?: string;
  reason?: string;
}

export type PowerXPluginToHost =
  | PowerXPluginReadyPayload
  | PowerXPluginRequestSyncPayload
  | PowerXPluginAuthTokenRequestPayload
  | PowerXPluginPingPayload
  | PowerXPluginFullscreenPayload;

export interface PowerXBridgeLogger {
  info?: (...args: any[]) => void;
  warn?: (...args: any[]) => void;
  error?: (...args: any[]) => void;
}

export interface PowerXBridgeOptions {
  pluginId?: string;
  instanceId?: string;
  debug?: boolean;
  allowedOrigins?: string[];
  logger?: PowerXBridgeLogger;
  onTheme?: (theme: PowerXThemeKey) => void;
  onLocale?: (locale: string) => void;
  onSync?: (payload: PowerXSyncPayload) => void;
  onAuthToken?: (payload: PowerXAuthTokenPayload) => void;
}

const BRIDGE_KEY = "__POWERX_FRAMEWORK_BRIDGE__";
const AUTH_TOKEN_REQUEST_CACHE_KEY = "__POWERX_AUTH_TOKEN_REQUEST_CACHE__";
const AUTH_TOKEN_REQUEST_DEDUPE_MS = 1_000;

function getWindow(): Window | null {
  return typeof window === "undefined" ? null : window;
}

function defaultAllowedOrigins() {
  const win = getWindow();
  const origins: string[] = [];
  if (!win) return origins;
  try {
    if (document.referrer) origins.push(new URL(document.referrer).origin);
  } catch {}
  try {
    origins.push(win.location.origin);
  } catch {}
  return Array.from(new Set(origins.filter(Boolean)));
}

export class PowerXBridgeClient {
  public pluginId?: string;
  public instanceId?: string;
  public debug: boolean;
  public onTheme?: (theme: PowerXThemeKey) => void;
  public onLocale?: (locale: string) => void;
  public onSync?: (payload: PowerXSyncPayload) => void;
  public onAuthToken?: (payload: PowerXAuthTokenPayload) => void;

  private allowedOrigins: Set<string>;
  private bound = false;
  private stopped = false;
  private lastHostOrigin?: string;
  private logger: PowerXBridgeLogger;

  constructor(options: PowerXBridgeOptions = {}) {
    this.pluginId = options.pluginId;
    this.instanceId = options.instanceId;
    this.debug = Boolean(options.debug);
    this.onTheme = options.onTheme;
    this.onLocale = options.onLocale;
    this.onSync = options.onSync;
    this.onAuthToken = options.onAuthToken;
    this.logger = options.logger || console;
    this.allowedOrigins = new Set([
      ...defaultAllowedOrigins(),
      ...(options.allowedOrigins || []).filter(Boolean),
    ]);
    this.log("init", {
      pluginId: this.pluginId,
      instanceId: this.instanceId,
      allowedOrigins: Array.from(this.allowedOrigins),
      location: getWindow()?.location?.origin,
      referrer: typeof document === "undefined" ? "" : document.referrer,
    });
  }

  start() {
    const win = getWindow();
    if (!win || this.bound || this.stopped) return;
    this.bound = true;
    win.addEventListener("message", this.handle, false);
    this.ready();
  }

  stop() {
    const win = getWindow();
    if (!win || !this.bound) return;
    this.stopped = true;
    this.bound = false;
    win.removeEventListener("message", this.handle, false);
    this.log("stopped");
  }

  ready() {
    this.sendToParent({
      source: "plugin",
      type: "ready",
      pluginId: this.pluginId,
      instanceId: this.instanceId,
    });
  }

  requestSync() {
    this.sendToParent({ source: "plugin", type: "request-sync" });
  }

  requestAuthToken(instanceId?: string) {
    this.sendToParent({
      source: "powerx-plugin",
      type: "auth-token:request",
      pluginId: this.pluginId,
      instanceId: instanceId || this.instanceId,
    });
  }

  ping() {
    this.sendToParent({ source: "plugin", type: "ping", ts: Date.now() });
  }

  requestFullscreen(options: PowerXFullscreenRequestOptions = {}) {
    return this.sendFullscreenRequest("enter", options);
  }

  exitFullscreen(options: PowerXFullscreenRequestOptions = {}) {
    return this.sendFullscreenRequest("exit", options);
  }

  toggleFullscreen(options: PowerXFullscreenRequestOptions = {}) {
    return this.sendFullscreenRequest("toggle", options);
  }

  private handle = (event: MessageEvent<any>) => {
    const data = event.data as PowerXHostMessage;
    if (!data || data.source !== "powerx") return;
    if (!this.isAllowedOrigin(event.origin)) {
      this.log("drop message: origin not allowed", { origin: event.origin, type: data.type });
      return;
    }

    switch (data.type) {
      case "sync":
        this.lastHostOrigin = data.hostOrigin || event.origin;
        this.onSync?.(data);
        break;
      case "locale":
        this.onLocale?.(data.locale);
        break;
      case "theme":
        this.onTheme?.(data.theme);
        break;
      case "auth-token":
        this.lastHostOrigin = data.hostOrigin || data.host_origin || event.origin;
        this.onAuthToken?.(data);
        break;
    }
  };

  private sendToParent(payload: PowerXPluginToHost) {
    const win = getWindow();
    const target = win?.parent || win?.top;
    if (!win || !target || target === win) {
      this.log("no parent window; skip postMessage", payload);
      return false;
    }
    target.postMessage(payload, this.lastHostOrigin || "*");
    return true;
  }

  private sendFullscreenRequest(
    action: PowerXFullscreenAction,
    options: PowerXFullscreenRequestOptions = {},
  ) {
    return this.sendToParent(createFullscreenPayload(action, {
      pluginId: options.pluginId || this.pluginId,
      instanceId: options.instanceId || this.instanceId,
      route: options.route,
      reason: options.reason,
    }));
  }

  private isAllowedOrigin(origin: string) {
    if (this.allowedOrigins.has("*")) return true;
    if (this.allowedOrigins.has(origin)) return true;
    return false;
  }

  private log(...args: any[]) {
    if (!this.debug) return;
    this.logger.info?.("[PowerXBridge]", ...args);
  }
}

export function initPowerXBridge(options: PowerXBridgeOptions = {}) {
  const win = getWindow();
  if (!win) {
    throw new Error("PowerX bridge requires a browser window");
  }
  const globalObject = win as any;
  if (globalObject[BRIDGE_KEY]) {
    const client = globalObject[BRIDGE_KEY] as PowerXBridgeClient;
    client.onLocale = options.onLocale || client.onLocale;
    client.onTheme = options.onTheme || client.onTheme;
    client.onSync = options.onSync || client.onSync;
    client.onAuthToken = options.onAuthToken || client.onAuthToken;
    return client;
  }
  const client = new PowerXBridgeClient(options);
  client.start();
  globalObject[BRIDGE_KEY] = client;
  return client;
}

export function requestPowerXHostAuthToken(pluginId: string, instanceId?: string) {
  const win = getWindow();
  if (!win?.parent) return;
  const resolvedInstanceId =
    instanceId ||
    (typeof location === "undefined" ? undefined : location.pathname + location.search);
  const requestKey = `${pluginId}::${resolvedInstanceId || ""}`;
  const globalObject = win as any;
  const cache: Map<string, number> =
    globalObject[AUTH_TOKEN_REQUEST_CACHE_KEY] ||
    (globalObject[AUTH_TOKEN_REQUEST_CACHE_KEY] = new Map<string, number>());
  const now = Date.now();
  const lastRequestedAt = cache.get(requestKey) || 0;
  if (now - lastRequestedAt < AUTH_TOKEN_REQUEST_DEDUPE_MS) {
    return;
  }
  cache.set(requestKey, now);
  const payload: PowerXPluginAuthTokenRequestPayload = {
    source: "powerx-plugin",
    type: "auth-token:request",
    pluginId,
    instanceId: resolvedInstanceId,
  };
  win.parent.postMessage(payload, "*");
}

export interface PowerXFullscreenRequestOptions {
  pluginId?: string;
  instanceId?: string;
  route?: string;
  reason?: string;
}

export function requestPowerXHostFullscreen(
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  return postPowerXHostFullscreen("enter", pluginId, options);
}

export function exitPowerXHostFullscreen(
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  return postPowerXHostFullscreen("exit", pluginId, options);
}

export function togglePowerXHostFullscreen(
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  return postPowerXHostFullscreen("toggle", pluginId, options);
}

function postPowerXHostFullscreen(
  action: PowerXFullscreenAction,
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  const win = getWindow();
  if (!win?.parent || win.parent === win) return false;
  const payload = createFullscreenPayload(action, {
    ...options,
    pluginId,
  });
  win.parent.postMessage(payload, "*");
  return true;
}

function createFullscreenPayload(
  action: PowerXFullscreenAction,
  options: PowerXFullscreenRequestOptions,
): PowerXPluginFullscreenPayload {
  return {
    source: "powerx-plugin",
    type: "fullscreen:request",
    action,
    pluginId: options.pluginId,
    instanceId:
      options.instanceId ||
      (typeof location === "undefined" ? undefined : location.pathname + location.search),
    route:
      options.route ||
      (typeof location === "undefined" ? undefined : location.pathname + location.search),
    reason: options.reason,
  };
}
