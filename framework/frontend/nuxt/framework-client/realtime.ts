import {
  createPluginSSEClient,
  type PluginSSEClient,
  type PluginSSEConnectOptions,
  type PluginSSEOptions,
  type PluginSSEStreamOptions,
} from "./sse";
import {
  createPluginWsBusClient,
  createPluginWsClient,
  type PluginWsBusClient,
  type PluginWsBusClientOptions,
  type PluginWsOptions,
} from "./ws";

export interface RealtimeContext {
  token?: string;
  tenantUuid?: string;
  memberUuid?: string;
}

export interface RealtimeClientOptions extends PluginSSEOptions, RealtimeContext {}

export interface RealtimeClient {
  setContext(context: RealtimeContext): void;
  getContext(): Readonly<RealtimeContext>;
  connectSSE(options: PluginSSEConnectOptions): EventSource;
  connectWS(options?: Omit<PluginWsOptions, "pluginId" | "apiBaseURL" | "token" | "tenantUuid">): WebSocket;
  streamSSE(options: PluginSSEStreamOptions): Promise<void>;
  createWSBus(options?: Omit<PluginWsBusClientOptions, "pluginId" | "apiBaseURL" | "token" | "tenantUuid" | "memberUuid">): PluginWsBusClient;
  close(): void;
}

const normalize = (value?: string) => String(value || "").trim();

// createRealtimeClient is the only entrypoint business code should use for
// browser realtime transports. Context replacement deliberately closes all
// prior connections so a tenant/member switch cannot retain old subscriptions.
export function createRealtimeClient(options: RealtimeClientOptions): RealtimeClient {
  let context: RealtimeContext = {
    token: normalize(options.token),
    tenantUuid: normalize(options.tenantUuid),
    memberUuid: normalize(options.memberUuid),
  };
  const sources = new Set<EventSource>();
  const sockets = new Set<WebSocket>();
  const wsClients = new Set<PluginWsBusClient>();

  const sseClient = (): PluginSSEClient => createPluginSSEClient({
    ...options,
    token: context.token,
    tenantUuid: context.tenantUuid,
  });
  const close = () => {
    sources.forEach((source) => source.close());
    sources.clear();
    sockets.forEach((socket) => socket.close());
    sockets.clear();
    wsClients.forEach((client) => client.disconnect());
    wsClients.clear();
  };
  const setContext = (next: RealtimeContext) => {
    const normalized: RealtimeContext = {
      token: normalize(next.token),
      tenantUuid: normalize(next.tenantUuid),
      memberUuid: normalize(next.memberUuid),
    };
    if (normalized.token === context.token && normalized.tenantUuid === context.tenantUuid && normalized.memberUuid === context.memberUuid) return;
    close();
    context = normalized;
  };

  return {
    setContext,
    getContext: () => ({ ...context }),
    connectSSE: (connectOptions) => {
      const source = sseClient().connect({ ...connectOptions, token: connectOptions.token || context.token, tenantUuid: connectOptions.tenantUuid || context.tenantUuid });
      sources.add(source);
      return source;
    },
    connectWS: (wsOptions = {}) => {
      const socket = createPluginWsClient({
        ...wsOptions,
        pluginId: options.pluginId,
        apiBaseURL: options.apiBaseURL,
        hostBaseURL: options.hostBaseURL,
        insidePowerX: options.insidePowerX,
        token: context.token,
        tenantUuid: context.tenantUuid,
      }).connect();
      sockets.add(socket);
      return socket;
    },
    streamSSE: (streamOptions) => sseClient().stream({ ...streamOptions, token: streamOptions.token || context.token, tenantUuid: streamOptions.tenantUuid || context.tenantUuid }),
    createWSBus: (wsOptions = {}) => {
      const client = createPluginWsBusClient({ ...wsOptions, pluginId: options.pluginId, apiBaseURL: options.apiBaseURL, hostBaseURL: options.hostBaseURL, insidePowerX: options.insidePowerX, token: context.token, tenantUuid: context.tenantUuid, memberUuid: context.memberUuid });
      wsClients.add(client);
      return client;
    },
    close,
  };
}
