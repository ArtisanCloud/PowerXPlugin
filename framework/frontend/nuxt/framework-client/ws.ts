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

export type PluginWsBusStatus =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "error";

export interface PluginWsBusMessage {
  type?: string;
  topic?: string;
  payload?: any;
  trace_id?: string;
  request_id?: string;
}

export interface PluginWsBusState {
  status: PluginWsBusStatus;
  connected: boolean;
  tenantUuid: string;
  welcomeReceived: boolean;
  subscribeSentAt: number;
  ackReceivedAt: number;
  eventReceivedAt: number;
  lastAckRequestID: string;
  lastEventTopic: string;
  lastEventTraceID: string;
  reconnectAttempts: number;
  maxReconnectAttempts: number;
  lastCloseCode: number;
  lastCloseReason: string;
  lastUrl: string;
  lastUrlSource: string;
}

export interface PluginWsBusClientOptions extends PluginWsOptions {
  buildURL?: () => string | { url: string; source?: string };
  reconnectIntervalMs?: number;
  maxReconnectAttempts?: number;
  debug?: boolean;
  logger?: {
    info?: (message: string, data?: Record<string, any>) => void;
    warn?: (message: string, data?: Record<string, any>) => void;
    error?: (message: string, data?: Record<string, any>) => void;
  };
  onStatus?: (state: PluginWsBusState) => void;
  onMark?: (stage: string, extra?: Record<string, any>) => void;
  onEvent?: (message: PluginWsBusMessage) => void;
}

export interface PluginWsBusClient {
  connect(): void;
  disconnect(): void;
  setContext(input: { tenantUuid?: string; token?: string }): void;
  subscribe(topics: Array<string | null | undefined>): void;
  unsubscribe(topics: Array<string | null | undefined>): void;
  getState(): PluginWsBusState;
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
      const wsProtocol =
        parsedBase.protocol === "https:" ? "wss:" : parsedBase.protocol;
      url = new URL(`${wsProtocol}//${parsedBase.host}${wsPathRaw}`);
    } else if (options.insidePowerX) {
      // Embedded mode prefers explicit hostBaseURL, but falls back to current origin
      // to avoid hard-fail when page-level runtime options are missing.
      const resolvedHostBase = hostBase || window.location.origin;
      const parsedHost = new URL(resolvedHostBase, window.location.origin);
      const wsProtocol =
        parsedHost.protocol === "https:" ? "wss:" : parsedHost.protocol;
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

const cloneState = (state: PluginWsBusState): PluginWsBusState => ({
  ...state,
});

const cleanTopics = (topics: Array<string | null | undefined>) =>
  Array.from(
    new Set(
      (Array.isArray(topics) ? topics : [])
        .map((item) => String(item || "").trim())
        .filter(Boolean),
    ),
  );

export function createPluginWsBusClient(
  options: PluginWsBusClientOptions,
): PluginWsBusClient {
  let ws: WebSocket | null = null;
  let retryTimer: ReturnType<typeof setTimeout> | null = null;
  let manualClosing = false;
  let token = String(options.token || "").trim();
  let tenantUuid = String(options.tenantUuid || "").trim();
  const queuedTopics = new Set<string>();
  const sentTopics = new Set<string>();
  const reconnectIntervalMs = Math.max(
    250,
    Number(options.reconnectIntervalMs || 1500),
  );
  const state: PluginWsBusState = {
    status: "idle",
    connected: false,
    tenantUuid,
    welcomeReceived: false,
    subscribeSentAt: 0,
    ackReceivedAt: 0,
    eventReceivedAt: 0,
    lastAckRequestID: "",
    lastEventTopic: "",
    lastEventTraceID: "",
    reconnectAttempts: 0,
    maxReconnectAttempts: Math.max(
      0,
      Number(options.maxReconnectAttempts || 0),
    ),
    lastCloseCode: 0,
    lastCloseReason: "",
    lastUrl: "",
    lastUrlSource: "",
  };

  const emitStatus = () => options.onStatus?.(cloneState(state));
  const emitMark = (stage: string, extra?: Record<string, any>) =>
    options.onMark?.(stage, extra);
  const logInfo = (message: string, data?: Record<string, any>) => {
    if (options.debug) options.logger?.info?.(message, data);
  };

  const buildURL = () => {
    if (typeof window === "undefined") return { url: "", source: "" };
    if (options.buildURL) {
      const resolved = options.buildURL();
      if (typeof resolved === "string") return { url: resolved, source: "" };
      return {
        url: String(resolved?.url || ""),
        source: String(resolved?.source || ""),
      };
    }
    const client = createPluginWsClient({
      ...options,
      token,
      tenantUuid,
    });
    return { url: client.buildURL(), source: "framework" };
  };

  const flushQueuedTopics = () => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    const topics = Array.from(queuedTopics).filter(
      (topic) => topic && !sentTopics.has(topic),
    );
    if (topics.length === 0) return;
    topics.forEach((topic) => sentTopics.add(topic));
    ws.send(JSON.stringify({ type: "subscribe", topics }));
    state.subscribeSentAt = Date.now();
    emitMark("subscribe_sent", { topics });
    emitStatus();
  };

  const disconnect = () => {
    manualClosing = true;
    if (retryTimer) {
      clearTimeout(retryTimer);
      retryTimer = null;
    }
    if (ws) {
      ws.onopen = null;
      ws.onclose = null;
      ws.onerror = null;
      ws.onmessage = null;
      ws.close();
      ws = null;
    }
    state.connected = false;
    state.status = "idle";
    sentTopics.clear();
    emitStatus();
  };

  const connect = () => {
    if (!tenantUuid) return;
    if (
      ws &&
      (ws.readyState === WebSocket.OPEN ||
        ws.readyState === WebSocket.CONNECTING)
    ) {
      flushQueuedTopics();
      return;
    }
    disconnect();
    manualClosing = false;
    state.status = "connecting";
    state.tenantUuid = tenantUuid;
    state.welcomeReceived = false;
    state.subscribeSentAt = 0;
    state.ackReceivedAt = 0;
    state.eventReceivedAt = 0;
    state.lastAckRequestID = "";
    state.lastEventTopic = "";
    state.lastEventTraceID = "";
    state.lastCloseCode = 0;
    state.lastCloseReason = "";
    const resolved = buildURL();
    state.lastUrl = resolved.url;
    state.lastUrlSource = resolved.source;
    emitStatus();
    logInfo("connect", { url: state.lastUrl, source: state.lastUrlSource });
    try {
      ws = new WebSocket(state.lastUrl);
    } catch {
      ws = null;
      state.status = "error";
      emitStatus();
      retryTimer = setTimeout(connect, reconnectIntervalMs);
      return;
    }

    ws.onopen = () => {
      state.connected = true;
      state.status = "connected";
      state.reconnectAttempts = 0;
      emitMark("onopen");
      emitStatus();
      flushQueuedTopics();
    };
    ws.onclose = (event) => {
      state.connected = false;
      state.lastCloseCode = Number(event?.code || 0);
      state.lastCloseReason = String(event?.reason || "").trim();
      state.status = "reconnecting";
      sentTopics.clear();
      emitStatus();
      if (manualClosing) return;
      if (state.lastCloseCode === 1000) {
        state.status = "idle";
        emitMark("closed_normal", {
          code: state.lastCloseCode,
          reason: state.lastCloseReason,
        });
        emitStatus();
        return;
      }
      if (
        state.maxReconnectAttempts > 0 &&
        state.reconnectAttempts >= state.maxReconnectAttempts
      ) {
        state.status = "error";
        emitMark("reconnect_exhausted", {
          attempts: state.reconnectAttempts,
          code: state.lastCloseCode,
          reason: state.lastCloseReason,
        });
        emitStatus();
        return;
      }
      state.reconnectAttempts += 1;
      emitMark("reconnect_scheduled", {
        attempt: state.reconnectAttempts,
        code: state.lastCloseCode,
        reason: state.lastCloseReason,
      });
      retryTimer = setTimeout(connect, reconnectIntervalMs);
    };
    ws.onerror = () => {
      state.connected = false;
      state.status = "error";
      emitStatus();
    };
    ws.onmessage = (event) => {
      if (!event?.data) return;
      try {
        const msg = JSON.parse(String(event.data)) as PluginWsBusMessage;
        const msgType = String(msg?.type || "").toLowerCase();
        logInfo("inbound", {
          type: msgType || "unknown",
          topic: String(msg?.topic || "").trim(),
          trace_id: String(
            msg?.trace_id || msg?.payload?.trace_id || "",
          ).trim(),
          req_id: String(
            msg?.payload?.req_id ||
              msg?.payload?.request_id ||
              msg?.request_id ||
              "",
          ).trim(),
        });
        if (msgType === "welcome") {
          state.welcomeReceived = true;
          emitMark("welcome");
          emitStatus();
        } else if (msgType === "ack") {
          state.ackReceivedAt = Date.now();
          state.lastAckRequestID = String(
            msg?.payload?.req_id ||
              msg?.payload?.request_id ||
              msg?.request_id ||
              "",
          ).trim();
          emitMark("ack_received", { topics: msg?.payload?.topics });
          emitStatus();
        } else if (msgType === "event") {
          state.eventReceivedAt = Date.now();
          state.lastEventTopic = String(msg?.topic || "").trim();
          state.lastEventTraceID = String(
            msg?.trace_id || msg?.payload?.trace_id || "",
          ).trim();
          emitMark("event_received", { topic: msg?.topic || "" });
          emitStatus();
        }
        options.onEvent?.(msg);
      } catch {
        // Ignore malformed payloads from custom gateways.
      }
    };
  };

  return {
    connect,
    disconnect,
    setContext(input) {
      const nextToken = String(input.token || "").trim();
      const nextTenantUuid = String(input.tenantUuid || "").trim();
      const changed = nextToken !== token || nextTenantUuid !== tenantUuid;
      token = nextToken;
      tenantUuid = nextTenantUuid;
      state.tenantUuid = tenantUuid;
      if (
        changed &&
        ws &&
        (ws.readyState === WebSocket.OPEN ||
          ws.readyState === WebSocket.CONNECTING)
      ) {
        disconnect();
      }
    },
    subscribe(topics) {
      for (const topic of cleanTopics(topics)) queuedTopics.add(topic);
      if ((!ws || ws.readyState === WebSocket.CLOSED) && tenantUuid) {
        state.reconnectAttempts = 0;
        connect();
        return;
      }
      flushQueuedTopics();
    },
    unsubscribe(topics) {
      const cleaned = cleanTopics(topics);
      cleaned.forEach((topic) => {
        queuedTopics.delete(topic);
        sentTopics.delete(topic);
      });
      if (ws && ws.readyState === WebSocket.OPEN && cleaned.length > 0) {
        ws.send(JSON.stringify({ type: "unsubscribe", topics: cleaned }));
        emitMark("unsubscribe_sent", { topics: cleaned });
      }
    },
    getState() {
      return cloneState(state);
    },
  };
}
