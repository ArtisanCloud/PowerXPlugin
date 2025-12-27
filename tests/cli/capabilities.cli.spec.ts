import test from "node:test";
import assert from "node:assert/strict";
import os from "node:os";
import path from "node:path";
import fs from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { runCapabilitiesLintCommand } from "../../tools/cli/src/commands/capabilities/lint";
import { runCapabilitiesSubmitCommand } from "../../tools/cli/src/commands/capabilities/submit";
import { runCapabilitiesQuotaCommand } from "../../tools/cli/src/commands/capabilities/quota";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

test("capabilities lint + submit + quota pipeline smoke", async (t) => {
  const fixtureDir = await fs.mkdtemp(path.join(os.tmpdir(), "px-cap-smoke-"));
  const sourceDir = path.join(repoRoot, "examples/com.powerx.demo");
  if (!(await exists(sourceDir))) {
    t.skip("examples/com.powerx.demo fixture missing");
    return;
  }
  await fs.cp(sourceDir, fixtureDir, { recursive: true });
  const manifestPath = path.join(fixtureDir, "plugin.yaml");

  const lint = await runCapabilitiesLintCommand({ manifestPath });
  assert.equal(lint.errors.length, 0);
  assert.equal(lint.checked > 0, true);

  const logs: Array<{ url: string; body: any }> = [];
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (url: RequestInfo | URL, init?: RequestInit) => {
    const parsedUrl = typeof url === "string" ? url : url.toString();
    const body = init?.body ? JSON.parse(init.body as string) : undefined;
    logs.push({ url: parsedUrl, body });
    const payload = {
      id: body?.manifestEntry?.id ?? "demo",
      status: "approved",
      submitted_at: new Date().toISOString(),
    };
    return new Response(JSON.stringify(payload), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  try {
    await runCapabilitiesSubmitCommand({
      manifestPath,
      baseUrl: "https://dev-api.powerx.local",
      token: "demo-token",
      rootDir: fixtureDir,
    });

    await runCapabilitiesQuotaCommand({
      capabilityId: "com.powerx.demo.templates.create",
      tenantId: "demo-tenant",
      baseUrl: "https://dev-api.powerx.local",
      token: "demo-token",
      manifestPath,
    });
  } finally {
    globalThis.fetch = originalFetch;
  }

  assert.ok(logs.some((entry) => entry.url.includes("/internal/plugins/capabilities")));
  const statePath = path.join(fixtureDir, ".px-plugin/capabilities.json");
  const stateRaw = await fs.readFile(statePath, "utf8");
  const state = JSON.parse(stateRaw);
  assert.ok(state.entries["com.powerx.demo.templates.create"]);

  const quotaSample = path.join(
    fixtureDir,
    "dist/capabilities/com.powerx.demo.templates.create/samples/tenant-demo-tenant-quota.postman.json",
  );
  assert.equal(await exists(quotaSample), true);
});

test("capabilities quota infers capability from manifest", async (t) => {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), "px-cap-quota-"));
  const manifestPath = path.join(dir, "plugin.yaml");
  await fs.writeFile(
    manifestPath,
    [
      "id: com.powerx.demo",
      "capabilities:",
      "  provides:",
      "    - id: com.powerx.demo.single",
    ].join("\n"),
    "utf8",
  );

  const logs: Array<{ url: string; body: any }> = [];
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (url: RequestInfo | URL, init?: RequestInit) => {
    const parsedUrl = typeof url === "string" ? url : url.toString();
    const body = init?.body ? JSON.parse(init.body as string) : undefined;
    logs.push({ url: parsedUrl, body });
    return new Response(JSON.stringify({ id: "quota-demo" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  try {
    await runCapabilitiesQuotaCommand({
      tenantId: "tenant-1",
      baseUrl: "https://dev-api.powerx.local",
      token: "demo-token",
      manifestPath,
    });
  } finally {
    globalThis.fetch = originalFetch;
  }

  assert.equal(logs.length > 0, true);
  assert.equal(logs[0].body.capabilityId, "com.powerx.demo.single");
  const sample = path.join(
    dir,
    "dist/capabilities/com.powerx.demo.single/samples/tenant-tenant-1-quota.http",
  );
  assert.equal(await exists(sample), true);
});

async function exists(file: string) {
  try {
    await fs.access(file);
    return true;
  } catch {
    return false;
  }
}
