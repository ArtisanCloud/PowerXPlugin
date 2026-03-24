import { test, expect } from "@playwright/test";
import { gotoWithFallback, seedAuthStorage } from "./_utils";

const tenantUUID = "00000000-0000-0000-0000-000000000001";
const notifyTopic = `plugin.notify.tenant.${tenantUUID}`;

const userContextPayload = {
  success: true,
  data: {
    is_root: true,
    current_tenant_uuid: tenantUUID,
    current_member_id: 1,
    user: {
      id: 1,
      email: "admin@local.test",
      phone: "",
      display_name: "E2E Admin",
      avatar_url: "",
      status: 1,
    },
    members: [
      {
        tenant_uuid: tenantUUID,
        tenant_name: "E2E Tenant",
        member_id: 1,
        is_admin: true,
      },
    ],
    roles: ["superadmin"],
    permissions: ["*"],
  },
};

test.describe("Notification WS Probe", () => {
  test("shows unread badge and event item after test notification", async ({ page }) => {
    await page.addInitScript(() => {
      class FakeWebSocket {
        static CONNECTING = 0;
        static OPEN = 1;
        static CLOSING = 2;
        static CLOSED = 3;
        static sockets: any[] = [];

        readyState = FakeWebSocket.CONNECTING;
        onopen: ((ev: Event) => void) | null = null;
        onclose: ((ev: CloseEvent) => void) | null = null;
        onerror: ((ev: Event) => void) | null = null;
        onmessage: ((ev: MessageEvent) => void) | null = null;
        subscriptions = new Set<string>();

        constructor(_url: string) {
          FakeWebSocket.sockets.push(this);
          queueMicrotask(() => {
            this.readyState = FakeWebSocket.OPEN;
            this.onopen?.(new Event("open"));
          });
        }

        send(raw: string) {
          try {
            const data = JSON.parse(raw);
            const type = String(data?.type || "").toLowerCase();
            const topics = Array.isArray(data?.topics) ? data.topics : [];
            if (type === "subscribe") {
              topics.forEach((topic: string) => this.subscriptions.add(String(topic)));
            }
            if (type === "unsubscribe") {
              topics.forEach((topic: string) => this.subscriptions.delete(String(topic)));
            }
          } catch {
            // ignore parse failure in test mock
          }
        }

        close() {
          this.readyState = FakeWebSocket.CLOSED;
          this.onclose?.(new CloseEvent("close"));
        }
      }

      // @ts-ignore
      window.WebSocket = FakeWebSocket;
      // @ts-ignore
      window.__E2E_WS_PUSH = (topic: string, payload: any) => {
        for (const socket of FakeWebSocket.sockets) {
          if (socket.readyState !== FakeWebSocket.OPEN) continue;
          if (!socket.subscriptions.has(topic)) continue;
          socket.onmessage?.(
            new MessageEvent("message", {
              data: JSON.stringify({
                type: "event",
                topic,
                payload,
              }),
            })
          );
        }
      };
      // @ts-ignore
      window.__E2E_WS_TOPICS = () => {
        const all = new Set<string>();
        for (const socket of FakeWebSocket.sockets) {
          socket.subscriptions.forEach((topic: string) => all.add(topic));
        }
        return Array.from(all);
      };
    });

    await seedAuthStorage(page);

    await page.route("**/admin/user/auth/me/context", async (route) => {
      await route.fulfill({
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(userContextPayload),
      });
    });

    await page.route("**/admin/notifications/test", async (route) => {
      await route.fulfill({
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          success: true,
          data: {
            ok: true,
            topic: notifyTopic,
            tenant_uuid: tenantUUID,
            trace_id: "trace-e2e-notify",
          },
        }),
      });
    });

    await gotoWithFallback(page, "/intro", page.locator("[data-test='navbar-notify-button']"));

    await page.locator("[data-test='navbar-notify-button']").click();
    await expect(page.locator("[data-test='navbar-notify-ws-state']")).toContainText(/已连接|连接中/);

    await page.locator("[data-test='notify-send-test']").click();
    await expect
      .poll(async () =>
        page.evaluate(() =>
          // @ts-ignore
          (window.__E2E_WS_TOPICS?.() || []) as string[]
        )
      )
      .toContain(notifyTopic);

    await page.evaluate(({ topic }) => {
      // @ts-ignore
      window.__E2E_WS_PUSH?.(topic, {
        type: "notification.test",
        title: "WS Test Notification",
        message: "from e2e",
      });
    }, { topic: notifyTopic });

    await expect(page.locator("[data-test='navbar-notify-unread']")).toBeVisible();
    await expect(page.locator("[data-test='navbar-notify-item']").first()).toContainText("from e2e");
  });
});
