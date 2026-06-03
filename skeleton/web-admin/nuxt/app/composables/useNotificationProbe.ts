import { getAuthToken, getTenantUuid, resolveApiBase } from "~/composables/api/_base";
import { useApiClient } from "~/composables/api/_client";
import { resolveFrontendRuntimeMode } from "~/utils/runtime-mode";
import { createPluginWsClient } from "@artisan-cloud/plugin-framework-client";
import { createFrameworkLogger } from "@artisan-cloud/plugin-framework-client";

export type NotificationWsState =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "error";

export interface NotificationProbeEvent {
  id: string;
  topic: string;
  type: string;
  title: string;
  message: string;
  payload: any;
  receivedAt: string;
}

export interface NotificationWsDiag {
  subSent: boolean;
  ackOK: boolean;
  eventOK: boolean;
  welcomeOK: boolean;
  connectedOK: boolean;
  lastAckReqID: string;
  lastEventTopic: string;
  lastEventTraceID: string;
}

const MAX_EVENTS = 50;
const RECONNECT_DELAY_MS = 1200;
const logger = createFrameworkLogger("notification-probe");

function buildWsURL(scope: string) {
  if (typeof window === "undefined") return "";
  const token = getAuthToken();
  if (!token) return "";
  const runtimeConfig = useRuntimeConfig();
  const runtimeMode = resolveFrontendRuntimeMode();
  const pluginId =
    String(runtimeConfig.public?.powerxPluginId || "").trim() ||
    "com.powerx.plugins.base";
  const isGatewayScope = scope === "gateway";
  const wsBaseURL = String(
    isGatewayScope
      ? runtimeConfig.public?.pxWsBaseUrl ||
          runtimeConfig.public?.wsOrigin ||
          runtimeConfig.public?.powerxCoreBase ||
          (typeof window !== "undefined" ? window.location.origin : "")
      : runtimeConfig.public?.wsOrigin ||
          resolveApiBase() ||
          (typeof window !== "undefined" ? window.location.origin : "")
  ).trim();
  const wsPath = String(
    runtimeConfig.public?.wsUrl || runtimeConfig.public?.wsPath || "/api/ws"
  ).trim();
  const ws = createPluginWsClient({
    pluginId,
    apiBaseURL: resolveApiBase(),
    wsBaseURL,
    wsPath,
    insidePowerX: isGatewayScope ? true : runtimeMode.insidePowerX,
    token,
    tenantUuid: getTenantUuid(),
  });
  const url = ws.buildURL();
  logger.debug("ws url built", {
    mode: runtimeMode.mode,
    scope,
    insidePowerX: runtimeMode.insidePowerX,
    wsBaseURL,
    wsPath,
    wsURL: url,
    tenantUuid: getTenantUuid(),
  });
  return url;
}

function resolveDefaultTopic(fallback?: string) {
  return String(fallback || "_topic.system.notification").trim();
}

function parseIncomingEvent(topic: string, payload: any): NotificationProbeEvent {
  const now = new Date().toISOString();
  const eventType = String(payload?.type || payload?.event_type || "event");
  const title = String(payload?.title || payload?.name || "WS Event");
  const message = String(payload?.message || payload?.detail || "");
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    topic,
    type: eventType,
    title,
    message,
    payload,
    receivedAt: now,
  };
}

export function useNotificationProbe(scope = "default", defaultTopic = "_topic.system.notification") {
  let wsConn: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let closedByClient = false;
  const subscribedTopics = new Set<string>();

  const statePrefix = `notifications.${scope}`;
  const wsState = useState<NotificationWsState>(`${statePrefix}.ws.state`, () => "idle");
  const wsError = useState<string>(`${statePrefix}.ws.error`, () => "");
  const lastEventAt = useState<string>(`${statePrefix}.lastEventAt`, () => "");
  const lastEventTopic = useState<string>(`${statePrefix}.lastEventTopic`, () => "");
  const unreadCount = useState<number>(`${statePrefix}.unreadCount`, () => 0);
  const events = useState<NotificationProbeEvent[]>(`${statePrefix}.events`, () => []);
  const wsDiag = useState<NotificationWsDiag>(`${statePrefix}.ws.diag`, () => ({
    subSent: false,
    ackOK: false,
    eventOK: false,
    welcomeOK: false,
    connectedOK: false,
    lastAckReqID: "",
    lastEventTopic: "",
    lastEventTraceID: "",
  }));

  const resetDiag = () => {
    wsDiag.value.subSent = false;
    wsDiag.value.ackOK = false;
    wsDiag.value.eventOK = false;
    wsDiag.value.welcomeOK = false;
    wsDiag.value.connectedOK = false;
    wsDiag.value.lastAckReqID = "";
    wsDiag.value.lastEventTopic = "";
    wsDiag.value.lastEventTraceID = "";
  };

  const wsStateLabel = computed(() => {
    switch (wsState.value) {
      case "connected":
        return "已连接";
      case "connecting":
        return "连接中";
      case "reconnecting":
        return "重连中";
      case "error":
        return "异常";
      default:
        return "未连接";
    }
  });

  const wsStateColor = computed(() => {
    switch (wsState.value) {
      case "connected":
        return "success";
      case "connecting":
      case "reconnecting":
        return "warning";
      case "error":
        return "error";
      default:
        return "neutral";
    }
  });

  const pushEvent = (event: NotificationProbeEvent) => {
    const next = [event, ...events.value];
    events.value = next.slice(0, MAX_EVENTS);
    unreadCount.value += 1;
    lastEventAt.value = event.receivedAt;
    lastEventTopic.value = event.topic;
    wsDiag.value.eventOK = true;
    wsDiag.value.lastEventTopic = event.topic;
    wsDiag.value.lastEventTraceID = String(
      event?.payload?.trace_id || event?.payload?.traceID || ""
    );
  };

  const subscribeTopic = (topic: string) => {
    const clean = String(topic || "").trim();
    if (!clean) return;
    if (subscribedTopics.has(clean)) return;
    subscribedTopics.add(clean);
    if (wsConn && wsConn.readyState === WebSocket.OPEN) {
      wsDiag.value.subSent = true;
      wsConn.send(JSON.stringify({ type: "subscribe", topics: [clean] }));
    }
  };

  const connect = () => {
    if (typeof window === "undefined") return;
    if (wsConn && (wsConn.readyState === WebSocket.OPEN || wsConn.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const wsURL = buildWsURL(scope);
    if (!wsURL) {
      wsState.value = "error";
      wsError.value = "缺少登录 token，已跳过 WS 连接";
      logger.warn("ws connect skipped: missing token");
      return;
    }

    closedByClient = false;
    wsState.value = wsState.value === "idle" ? "connecting" : "reconnecting";
    wsError.value = "";
    resetDiag();

    const socket = new WebSocket(wsURL);
    wsConn = socket;
    logger.info("ws connecting", { mode: resolveFrontendRuntimeMode().mode, tenantUuid: getTenantUuid(), wsURL });

    socket.onopen = () => {
      wsState.value = "connected";
      wsError.value = "";
      wsDiag.value.connectedOK = true;
      const topic = resolveDefaultTopic(defaultTopic);
      if (topic) {
        subscribedTopics.add(topic);
      }
      if (subscribedTopics.size > 0) {
        wsDiag.value.subSent = true;
        logger.info("ws subscribe sent", { topic: Array.from(subscribedTopics).join(",") });
        socket.send(
          JSON.stringify({
            type: "subscribe",
            topics: Array.from(subscribedTopics),
          })
        );
      }
    };

    socket.onmessage = (event) => {
      let data: any = null;
      try {
        data = JSON.parse(event.data);
      } catch {
        return;
      }
      const msgType = String(data?.type || "").toLowerCase();
      if (msgType === "welcome") {
        wsDiag.value.welcomeOK = true;
        logger.info("ws welcome received");
        return;
      }
      if (msgType === "ack") {
        wsDiag.value.ackOK = true;
        wsDiag.value.lastAckReqID = String(
          data?.req_id || data?.request_id || data?.payload?.req_id || ""
        );
        logger.info("ws ack received", { requestId: wsDiag.value.lastAckReqID });
        return;
      }
      if (msgType === "event") {
        const topic = String(data?.topic || "");
        const parsed = parseIncomingEvent(topic, data?.payload || {});
        pushEvent(parsed);
        logger.info("ws event received", {
          topic,
          traceId: String(data?.payload?.trace_id || data?.payload?.traceID || ""),
        });
        return;
      }
      if (msgType === "error") {
        wsError.value = String(data?.message || "ws error");
        logger.warn("ws error message", { detail: wsError.value });
      }
    };

    socket.onerror = () => {
      wsState.value = "error";
      wsError.value = "WebSocket 连接异常";
      logger.error("ws connection error", { wsURL });
    };

    socket.onclose = () => {
      wsConn = null;
      if (closedByClient) {
        wsState.value = "idle";
        resetDiag();
        logger.info("ws closed by client");
        return;
      }
      wsState.value = "reconnecting";
      resetDiag();
      logger.warn("ws closed, scheduling reconnect", { wsURL });
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      reconnectTimer = setTimeout(() => {
        connect();
      }, RECONNECT_DELAY_MS);
    };
  };

  const disconnect = () => {
    closedByClient = true;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (wsConn) {
      wsConn.close();
      wsConn = null;
    }
    wsState.value = "idle";
    resetDiag();
  };

  const markAllRead = () => {
    unreadCount.value = 0;
  };

  const clearEvents = () => {
    events.value = [];
    unreadCount.value = 0;
    lastEventAt.value = "";
    lastEventTopic.value = "";
    wsDiag.value.eventOK = false;
    wsDiag.value.lastEventTopic = "";
    wsDiag.value.lastEventTraceID = "";
  };

  const sendTestNotification = async (
    message?: string,
    options?: { topic?: string; tenantUUID?: string }
  ) => {
    const apiClient = useApiClient();
    const topic = String(options?.topic || "").trim();
    const tenantUUID = String(options?.tenantUUID || "").trim();
    const body = {
      message: String(message || "WebSocket probe from navbar"),
      ...(topic ? { topic } : {}),
      ...(tenantUUID ? { tenant_uuid: tenantUUID } : {}),
    };
    const resp: any = await apiClient.post("/admin/notifications/test", body);
    const resolvedTopic = String(resp?.data?.topic || "").trim();
    if (resolvedTopic) {
      subscribeTopic(resolvedTopic);
    }
    return resp;
  };

  return {
    wsState: readonly(wsState),
    wsStateLabel,
    wsStateColor,
    wsError: readonly(wsError),
    wsDiag: readonly(wsDiag),
    lastEventAt: readonly(lastEventAt),
    lastEventTopic: readonly(lastEventTopic),
    unreadCount: readonly(unreadCount),
    events: readonly(events),
    connect,
    disconnect,
    subscribeTopic,
    markAllRead,
    clearEvents,
    sendTestNotification,
  };
}
