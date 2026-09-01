import { expect, test } from "@playwright/test";
import { gotoWithFallback, seedAuthStorage } from "./_utils";

const sessionID = "20000000-0000-4000-8000-000000000001";

test.describe("Framework MCP realtime stream", () => {
  test("registers an MCP session and receives its managed SSE events", async ({ page }) => {
    await seedAuthStorage(page);
    const streamRequests: string[] = [];

    await page.route("**/api/v1/admin/capabilities/register/template", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { namespace: "com.example", sensitivity_options: [], async_modes: [], tag_suggestions: [], field_hints: {}, schema_placeholders: { input: "{}", output: "{}" }, protocol_samples: {}, identifier_example: "com.example.workflow" } }),
      }),
    );
    await page.route("**/api/v1/admin/capabilities", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [{ id: "com.example.workflow", version: "1.0.0", descriptor: "plugin.d/capabilities.yaml", module: "example", kind: "workflow", tags: [], checksum: "checksum", execution: { mode: "async" } }] }),
      }),
    );
    await page.route("**/api/v1/admin/runtime/sessions/register", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { id: sessionID, runtime_assignment_id: "assignment-1", tenant_uuid: "tenant-e2e", state: "registering", missed_heartbeats: 0, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" } }),
      });
    });
    await page.route("**/api/v1/mcp/sse**", async (route) => {
      streamRequests.push(route.request().url());
      await route.fulfill({
        status: 200,
        contentType: "text/event-stream",
        body: "data: {\"type\":\"progress\",\"timestamp\":\"2026-01-01T00:00:00Z\",\"payload\":{\"percent\":50}}\n\n",
      });
    });

    await gotoWithFallback(page, "/capabilities/register", page.getByTestId("mcp-debug-capability"));
    await page.getByTestId("mcp-debug-capability").click();
    await page.getByTestId("mcp-session-register").click();
    await expect(page.getByTestId("mcp-stream-toggle")).toBeEnabled();
    await page.getByTestId("mcp-stream-toggle").click();

    await expect(page.getByTestId("mcp-stream-events")).toContainText("progress");
    expect(streamRequests).toHaveLength(1);
    expect(new URL(streamRequests[0]).searchParams.get("session_id")).toBe(sessionID);
  });
});
