#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { spawnSync } from "node:child_process";
import { Command } from "commander";
import crypto from "node:crypto";

const DEFAULT_CANONICAL_MANIFEST = "./skeleton/plugin.yaml";
const DEFAULT_LEGACY_MANIFEST = "./plugin.yaml";
const DEFAULT_ENDPOINT = "/tenant/invocations";
const DEFAULT_TIMEOUT_MS = 15000;

const program = new Command();
program
  .name("run-from-package")
  .description("Validate capabilities manifest, optionally invoke Gateway for debugging")
  .option("--mode <mode>", "运行模式：host | skeleton", "host")
  .option("--manifest <path>", "自定义 manifest 路径")
  .option("--env-file <path>", "额外加载的 env 文件")
  .option("--cap <capabilityId>", "调用指定 capabilityId 验证 Gateway 调用链")
  .option("--action <action>", "能力 action（默认 List）", "List")
  .option("--payload <jsonOrFile>", "能力 payload JSON 或 @file 路径", "{}")
  .option("--api-base <url>", "Gateway Base 覆盖，默认读取 PX_GATEWAY_BASE_URL")
  .option("--endpoint <path>", "Gateway Endpoint 路径", DEFAULT_ENDPOINT)
  .option("--tenant <uuid>", "Tenant UUID 覆盖，默认读取 PX_TENANT_UUID")
  .option("--tool-token <token>", "Tool Token 覆盖，默认读取 PX_TOOL_TOKEN/PX_PLUGIN_TOOL_TOKEN")
  .option("--request-id <value>", "自定义 Request ID")
  .option("--timeout <ms>", "HTTP 超时时间 (ms)", String(DEFAULT_TIMEOUT_MS))
  .option("--use-mock <module>", "跳过真实 Gateway，返回指定模块的 Mock 响应");

const opts = program.parse(process.argv).opts();

async function main() {
  const normalizedMode = normalizeMode(opts.mode);
  applyEnvFiles(normalizedMode, opts.envFile);
  if (opts.use-mock) {
    process.env.PX_USE_MOCK = opts.use-mock;
    console.log(`[capabilities] PX_USE_MOCK 已设置为 ${opts.use-mock}`);
  }

  const manifestInfo = resolveManifestPath(normalizedMode, opts.manifest);
  console.log(
    `[capabilities] 使用 manifest ${manifestInfo.display} （mode=${normalizedMode}, resolved=${manifestInfo.resolved}）`,
  );

  runValidation(manifestInfo.resolved, normalizedMode, program.args);

  if (opts.cap) {
    await invokeCapability({
      capabilityId: opts.cap,
      action: opts.action ?? "List",
      payloadArg: opts.payload,
      gatewayBase: opts.apiBase ?? process.env.PX_GATEWAY_BASE_URL,
      endpoint: opts.endpoint ?? DEFAULT_ENDPOINT,
      tenantUUID: opts.tenant ?? process.env.PX_TENANT_UUID,
      toolToken: opts.toolToken ?? process.env.PX_TOOL_TOKEN ?? process.env.PX_PLUGIN_TOOL_TOKEN,
      requestId: opts.requestId,
      timeoutMs: Number(opts.timeout ?? DEFAULT_TIMEOUT_MS),
      useMockModule: opts.useMock,
    });
  }
}

main().catch((error) => {
  console.error("[capabilities] 脚本执行失败:", error?.message ?? error);
  process.exit(1);
});

function normalizeMode(mode) {
  return String(mode || "host").trim().toLowerCase();
}

function applyEnvFiles(mode, explicitEnvFile) {
  const candidates = [];
  if (explicitEnvFile) {
    candidates.push(explicitEnvFile);
  }
  if (mode === "skeleton") {
    candidates.push("./skeleton/.env.local");
  }

  const loaded = [];
  for (const candidate of candidates) {
    if (!candidate) continue;
    const resolved = path.resolve(process.cwd(), candidate);
    if (!fs.existsSync(resolved)) continue;
    const applied = injectEnvFromFile(resolved);
    if (applied > 0) {
      loaded.push(`${candidate} (+${applied})`);
    }
  }
  if (loaded.length) {
    console.log(`[capabilities] 已从 ${loaded.join(", ")} 注入环境变量`);
  }
}

function injectEnvFromFile(file) {
  const content = fs.readFileSync(file, "utf8");
  const lines = content.split(/\r?\n/);
  let applied = 0;
  for (const rawLine of lines) {
    if (!rawLine) continue;
    let line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    if (line.startsWith("export ")) {
      line = line.slice(7).trim();
    }
    const idx = line.indexOf("=");
    if (idx === -1) continue;
    const key = line.slice(0, idx).trim();
    if (!key || process.env[key]) continue;
    let value = line.slice(idx + 1).trim();
    if (!value) {
      process.env[key] = "";
      applied += 1;
      continue;
    }
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    process.env[key] = value;
    applied += 1;
  }
  return applied;
}

function resolveManifestPath(mode, userSpecified) {
  const cwd = process.cwd();
  const canonical = path.resolve(cwd, DEFAULT_CANONICAL_MANIFEST);
  const legacy = path.resolve(cwd, DEFAULT_LEGACY_MANIFEST);
  const defaultByMode = mode === "skeleton" ? canonical : legacy;
  const fallbacks = mode === "skeleton" ? [canonical, legacy] : [legacy, canonical];

  if (userSpecified) {
    const resolved = path.resolve(cwd, userSpecified);
    return {
      resolved,
      display: userSpecified,
    };
  }

  if (fs.existsSync(defaultByMode)) {
    return { resolved: defaultByMode, display: path.relative(cwd, defaultByMode) };
  }

  for (const candidate of fallbacks) {
    if (fs.existsSync(candidate)) {
      return { resolved: candidate, display: path.relative(cwd, candidate) };
    }
  }

  console.error("[capabilities] 找不到 manifest，请使用 --manifest 指定路径。");
  process.exit(1);
}

function runValidation(manifestPath, mode, passthroughArgs) {
  const scriptPath = path.resolve(process.cwd(), "scripts/capabilities/validate-capabilities.mjs");
  const args = [scriptPath, "--manifest", manifestPath, "--mode", mode, ...passthroughArgs];
  const result = spawnSync(process.execPath, args, { stdio: "inherit" });
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

async function invokeCapability(options) {
  const capabilityId = options.capabilityId?.trim();
  if (!capabilityId) {
    console.warn("[capabilities] 未提供 --cap，跳过 Gateway 调用。");
    return;
  }

  const payload = resolvePayload(options.payloadArg);
  const action = options.action?.trim() || "List";
  const requestBody = {
    capabilityId,
    action,
    payload,
  };
  const requestId = options.requestId?.trim() || crypto.randomUUID();
  const headers = {
    "Content-Type": "application/json",
    "X-Request-ID": requestId,
  };
  if (options.tenantUUID) {
    headers["X-Tenant-UUID"] = options.tenantUUID;
  }
  if (options.toolToken) {
    headers.Authorization = `Bearer ${options.toolToken}`;
  }

  const sanitizedHeaders = sanitizeHeaders(headers);
  const targetUrl = options.useMockModule ? "mock://cli" : buildUrl(options.gatewayBase, options.endpoint);
  console.log("[capabilities] capability invocation request →");
  console.log(
    JSON.stringify(
      {
        url: targetUrl,
        method: "POST",
        headers: sanitizedHeaders,
        body: requestBody,
      },
      null,
      2,
    ),
  );

  if (options.useMockModule) {
    const mockResponse = buildMockResponse(options.useMockModule, capabilityId, action, payload);
    console.log("[capabilities] capability invocation response ←");
    console.log(JSON.stringify(mockResponse, null, 2));
    return;
  }

  validateGatewayCredentials(options);

  const controller = new AbortController();
  const timeout = Number.isFinite(options.timeoutMs) ? Math.max(1000, Number(options.timeoutMs)) : DEFAULT_TIMEOUT_MS;
  const timer = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(targetUrl, {
      method: "POST",
      headers,
      body: JSON.stringify(requestBody),
      signal: controller.signal,
    });
    const rawText = await response.text();
    let parsedBody = rawText;
    try {
      parsedBody = rawText ? JSON.parse(rawText) : {};
    } catch {
      /* keep raw string */
    }
    const responseLog = {
      status: response.status,
      ok: response.ok,
      traceId: response.headers.get("x-trace-id") || (parsedBody?.traceId ?? null),
      headers: pickHeaders(response.headers, ["x-trace-id", "content-type"]),
      body: parsedBody,
    };
    console.log("[capabilities] capability invocation response ←");
    console.log(JSON.stringify(responseLog, null, 2));
    if (!response.ok) {
      process.exit(response.status || 1);
    }
  } catch (error) {
    console.error("[capabilities] 调用 Gateway 失败:", error?.message ?? error);
    process.exit(1);
  } finally {
    clearTimeout(timer);
  }
}

function resolvePayload(payloadArg) {
  if (!payloadArg) {
    return {};
  }
  const trimmed = payloadArg.trim();
  if (!trimmed || trimmed === "{}") {
    return {};
  }
  if (trimmed.startsWith("@")) {
    const filePath = trimmed.slice(1);
    const resolved = path.resolve(process.cwd(), filePath);
    if (!fs.existsSync(resolved)) {
      console.error(`[capabilities] payload 文件不存在: ${filePath}`);
      process.exit(1);
    }
    const raw = fs.readFileSync(resolved, "utf8");
    return JSON.parse(raw || "{}");
  }
  return JSON.parse(trimmed);
}

function buildUrl(base, endpoint) {
  if (!base && endpoint?.startsWith("http")) {
    return endpoint;
  }
  const normalizedBase = (base || "").replace(/\/+$/, "");
  const normalizedEndpoint = endpoint?.startsWith("/") ? endpoint : `/${endpoint ?? ""}`;
  return `${normalizedBase}${normalizedEndpoint}`.replace(/\/{2,}/g, "/").replace(":/", "://");
}

function validateGatewayCredentials(options) {
  const missing = [];
  if (!options.gatewayBase) {
    missing.push("PX_GATEWAY_BASE_URL");
  }
  if (!options.toolToken) {
    missing.push("PX_TOOL_TOKEN/PX_PLUGIN_TOOL_TOKEN");
  }
  if (!options.tenantUUID) {
    missing.push("PX_TENANT_UUID");
  }
  if (missing.length) {
    console.error(`[capabilities] 缺少 Gateway 凭证: ${missing.join(", ")}`);
    process.exit(1);
  }
}

function sanitizeHeaders(headers) {
  const clone = { ...headers };
  if (clone.Authorization) {
    clone.Authorization = maskToken(clone.Authorization);
  }
  return clone;
}

function maskToken(value) {
  const token = String(value);
  const prefix = token.startsWith("Bearer ") ? "Bearer " : "";
  const raw = prefix ? token.slice(7) : token;
  if (!raw) return token;
  if (raw.length <= 6) {
    return `${prefix}${"*".repeat(raw.length)}`;
  }
  return `${prefix}${raw.slice(0, 4)}***${raw.slice(-2)}`;
}

function pickHeaders(headers, keys) {
  const lower = keys.map((key) => key.toLowerCase());
  const entries = [];
  for (const [key, value] of headers.entries()) {
    if (lower.includes(key.toLowerCase())) {
      entries.push([key, value]);
    }
  }
  return Object.fromEntries(entries);
}

function buildMockResponse(module, capabilityId, action, payload) {
  return {
    status: "mock",
    traceId: `mock-cli-${module}-${Date.now()}`,
    data: {
      mock: true,
      module,
      capability: capabilityId,
      action,
      message: `Mock 模式生效 (--use-mock=${module})`,
      echoPayload: payload ?? {},
    },
  };
}
