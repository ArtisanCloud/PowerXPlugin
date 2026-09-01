export interface PluginSSEOptions {
  pluginId: string;
  apiBaseURL?: string;
  hostBaseURL?: string;
  insidePowerX?: boolean;
  token?: string;
  tenantUuid?: string;
  withCredentials?: boolean;
}

export interface PluginSSEConnectOptions {
  path: string;
  params?: Record<string, unknown>;
  token?: string;
  tenantUuid?: string;
  withCredentials?: boolean;
  forwardEventsToMessage?: string[];
  onOpen?: (event: Event) => void;
  onError?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  events?: Record<string, (event: MessageEvent) => void>;
}

export interface PluginSSEStreamEvent {
  event: string;
  data: string;
  payload: unknown;
  raw: string;
}

export interface PluginSSEStreamOptions {
  path: string;
  params?: Record<string, unknown>;
  token?: string;
  tenantUuid?: string;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  onEvent?: (event: PluginSSEStreamEvent) => void;
  onError?: (error: Error) => void;
}

export interface PluginSSEClient {
  buildURL(
    path: string,
    params?: Record<string, unknown>,
    input?: { token?: string; tenantUuid?: string },
  ): string;
  connect(options: PluginSSEConnectOptions): EventSource;
  stream(options: PluginSSEStreamOptions): Promise<void>;
}

export type PluginSSEStatus = "idle" | "connecting" | "connected" | "error" | "closed";

export interface PluginSSEState {
  status: PluginSSEStatus;
  url: string;
  lastEventName: string;
  lastEventAt: number;
}

export interface PluginSSEManagedConnection {
  connect(): void;
  close(): void;
  getState(): PluginSSEState;
}

const trimSlashes = (value: string) => value.replace(/^\/+|\/+$/g, "");

const appendParams = (url: URL, params?: Record<string, unknown>) => {
  for (const [key, value] of Object.entries(params || {})) {
    if (value === undefined || value === null) continue;
    url.searchParams.set(key, String(value));
  }
};

const normalizeBearer = (token: string) => {
  const clean = String(token || "").trim();
  if (!clean) return "";
  return /^Bearer\s/i.test(clean) ? clean : `Bearer ${clean}`;
};

export function createPluginSSEClient(
  options: PluginSSEOptions,
): PluginSSEClient {
  const apiBase = String(
    options.apiBaseURL || `/_p/${options.pluginId}/api/v1`,
  ).replace(/\/+$/, "");
  const hostBase = String(options.hostBaseURL || "").replace(/\/+$/, "");

  const resolveBase = () => {
    if (typeof window === "undefined") return apiBase;
    if (options.insidePowerX && hostBase) {
      return hostBase;
    }
    if (!apiBase) return window.location.origin;
    return new URL(apiBase, window.location.origin).toString();
  };

  const buildURL = (
    path: string,
    params?: Record<string, unknown>,
    input?: { token?: string; tenantUuid?: string },
  ) => {
    const rawPath = String(path || "").trim();
    const cleanPath =
      rawPath.startsWith("http://") || rawPath.startsWith("https://")
        ? rawPath
        : trimSlashes(rawPath);
    const base = resolveBase();
    const url =
      rawPath.startsWith("http://") || rawPath.startsWith("https://")
        ? new URL(cleanPath)
        : new URL(cleanPath, `${base}/`);
    appendParams(url, params);

    const token = normalizeBearer(String(input?.token || options.token || ""));
    const tenantUuid = String(
      input?.tenantUuid || options.tenantUuid || "",
    ).trim();
    if (token) url.searchParams.set("authorization", token);
    if (tenantUuid) url.searchParams.set("tenant_uuid", tenantUuid);
    return url.toString();
  };

  const connect = (connectOptions: PluginSSEConnectOptions) => {
    const url = buildURL(connectOptions.path, connectOptions.params, {
      token: connectOptions.token,
      tenantUuid: connectOptions.tenantUuid,
    });
    const source = new EventSource(url, {
      withCredentials:
        connectOptions.withCredentials ?? options.withCredentials ?? true,
    });
    if (connectOptions.onOpen) source.onopen = connectOptions.onOpen;
    if (connectOptions.onError) source.onerror = connectOptions.onError;
    if (connectOptions.onMessage) source.onmessage = connectOptions.onMessage;
    if (connectOptions.forwardEventsToMessage && connectOptions.onMessage) {
      for (const eventName of connectOptions.forwardEventsToMessage) {
        if (!eventName) continue;
        source.addEventListener(
          eventName,
          connectOptions.onMessage as EventListener,
        );
      }
    }
    for (const [eventName, handler] of Object.entries(
      connectOptions.events || {},
    )) {
      if (!eventName || !handler) continue;
      source.addEventListener(eventName, handler as EventListener);
    }
    return source;
  };

  const stream = async (streamOptions: PluginSSEStreamOptions) => {
    try {
      const url = buildURL(streamOptions.path, streamOptions.params, {
        tenantUuid: streamOptions.tenantUuid,
      });
      const headers: Record<string, string> = {
        Accept: "text/event-stream",
        "Cache-Control": "no-cache",
        ...(streamOptions.headers || {}),
      };
      const token = normalizeBearer(
        String(streamOptions.token || options.token || ""),
      );
      if (token) {
        headers.Authorization = token;
      }
      const tenantUuid = String(
        streamOptions.tenantUuid || options.tenantUuid || "",
      ).trim();
      if (tenantUuid) {
        headers.tenant_uuid = tenantUuid;
      }
      const response = await fetch(url, {
        method: "GET",
        headers,
        signal: streamOptions.signal,
        credentials: options.withCredentials === false ? "same-origin" : "include",
      });
      if (!response.ok) {
        throw new Error(`SSE stream failed: HTTP ${response.status}`);
      }
      if (!response.body) {
        throw new Error("SSE stream response body is empty");
      }
      await readSSEStream(response.body, streamOptions.onEvent);
    } catch (cause) {
      const error = cause instanceof Error ? cause : new Error(String(cause));
      streamOptions.onError?.(error);
      throw error;
    }
  };

  return {
    buildURL,
    connect,
    stream,
  };
}

// createManagedPluginSSEConnection owns a single EventSource lifecycle. It is
// intentionally separate from fetch SSE, whose caller owns an AbortController.
export function createManagedPluginSSEConnection(
  client: PluginSSEClient,
  options: PluginSSEConnectOptions,
  onState?: (state: PluginSSEState) => void,
): PluginSSEManagedConnection {
  let source: EventSource | null = null;
  const state: PluginSSEState = { status: "idle", url: "", lastEventName: "", lastEventAt: 0 };
  const emit = () => onState?.({ ...state });
  const close = () => {
    if (source) source.close();
    source = null;
    state.status = "closed";
    emit();
  };
  return {
    connect() {
      close();
      state.status = "connecting";
      state.url = client.buildURL(options.path, options.params, { token: options.token, tenantUuid: options.tenantUuid });
      emit();
      source = client.connect({
        ...options,
        onOpen: (event) => {
          state.status = "connected";
          emit();
          options.onOpen?.(event);
        },
        onError: (event) => {
          state.status = "error";
          emit();
          options.onError?.(event);
        },
        onMessage: (event) => {
          state.lastEventName = "message";
          state.lastEventAt = Date.now();
          emit();
          options.onMessage?.(event);
        },
        events: Object.fromEntries(Object.entries(options.events || {}).map(([eventName, handler]) => [eventName, (event: MessageEvent) => {
          state.lastEventName = eventName;
          state.lastEventAt = Date.now();
          emit();
          handler(event);
        }])),
      });
    },
    close,
    getState: () => ({ ...state }),
  };
}

export async function readSSEStream(
  body: ReadableStream<Uint8Array>,
  onEvent?: (event: PluginSSEStreamEvent) => void,
) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let splitIndex = findSSEBoundary(buffer);
      while (splitIndex >= 0) {
        const chunk = buffer.slice(0, splitIndex);
        buffer = buffer.slice(skipBoundary(buffer, splitIndex));
        emitSSEChunk(chunk, onEvent);
        splitIndex = findSSEBoundary(buffer);
      }
    }
    buffer += decoder.decode();
    if (buffer.trim()) {
      emitSSEChunk(buffer, onEvent);
    }
  } finally {
    reader.releaseLock();
  }
}

function findSSEBoundary(buffer: string) {
  const lf = buffer.indexOf("\n\n");
  const crlf = buffer.indexOf("\r\n\r\n");
  if (lf < 0) return crlf;
  if (crlf < 0) return lf;
  return Math.min(lf, crlf);
}

function skipBoundary(buffer: string, index: number) {
  return buffer.startsWith("\r\n\r\n", index) ? index + 4 : index + 2;
}

function emitSSEChunk(
  chunk: string,
  onEvent?: (event: PluginSSEStreamEvent) => void,
) {
  const raw = chunk.trim();
  if (!raw) return;
  const lines = raw.split(/\r?\n/);
  const eventName = lines
    .find((line) => line.startsWith("event:"))
    ?.replace(/^event:\s*/, "")
    .trim() || "message";
  const data = lines
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.replace(/^data:\s*/, ""))
    .join("\n")
    .trim();
  let payload: unknown = {};
  if (data) {
    try {
      payload = JSON.parse(data);
    } catch {
      payload = { raw: data };
    }
  }
  onEvent?.({ event: eventName, data, payload, raw });
}
