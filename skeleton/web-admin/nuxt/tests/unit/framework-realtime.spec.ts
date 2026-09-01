import { afterEach, describe, expect, it, vi } from "vitest";

import {
  createManagedPluginSSEConnection,
  createPluginSSEClient,
  createPluginWsBusClient,
  readSSEStream,
} from "../../../../../framework/frontend/nuxt/framework-client";

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  onopen: ((event: Event) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  closed = false;
  constructor(readonly url: string) { FakeEventSource.instances.push(this); }
  addEventListener() {}
  close() { this.closed = true; }
}

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSED = 3;
  readyState = FakeWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  closed = false;
  constructor(readonly url: string) { FakeWebSocket.instances.push(this); }
  send() {}
  close() { this.closed = true; this.readyState = FakeWebSocket.CLOSED; }
}

afterEach(() => {
  vi.unstubAllGlobals();
  FakeEventSource.instances = [];
  FakeWebSocket.instances = [];
});

describe("framework realtime", () => {
  it("resolves a proxy SSE URL with tenant context", () => {
    vi.stubGlobal("window", { location: { origin: "https://host.test" } });
    const client = createPluginSSEClient({ pluginId: "com.example.demo", apiBaseURL: "/_p/com.example.demo/api/v1", token: "token", tenantUuid: "tenant-1" });
    expect(client.buildURL("/mcp/sse", { session_id: "session-1" })).toBe("https://host.test/_p/com.example.demo/api/v1/mcp/sse?session_id=session-1&authorization=Bearer+token&tenant_uuid=tenant-1");
  });

  it("closes the managed SSE source", () => {
    vi.stubGlobal("window", { location: { origin: "https://host.test" } });
    vi.stubGlobal("EventSource", FakeEventSource);
    const client = createPluginSSEClient({ pluginId: "com.example.demo", apiBaseURL: "/api/v1" });
    const connection = createManagedPluginSSEConnection(client, { path: "/mcp/sse" });
    connection.connect();
    connection.close();
    expect(FakeEventSource.instances[0].closed).toBe(true);
    expect(connection.getState().status).toBe("closed");
  });

  it("closes WS when the member context changes", () => {
    vi.stubGlobal("window", { location: { origin: "https://host.test" } });
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const client = createPluginWsBusClient({ pluginId: "com.example.demo", tenantUuid: "tenant-1", memberUuid: "member-1" });
    client.connect();
    client.setContext({ tenantUuid: "tenant-1", memberUuid: "member-2" });
    expect(FakeWebSocket.instances[0].closed).toBe(true);
    expect(client.getState().memberUuid).toBe("member-2");
  });

  it("parses an MCP SSE progress event", async () => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode('event: progress\ndata: {"percent":50}\n\n'));
        controller.close();
      },
    });
    const events: Array<{ event: string; payload: unknown }> = [];
    await readSSEStream(body, (event) => events.push(event));
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({ event: "progress", payload: { percent: 50 } });
  });
});
