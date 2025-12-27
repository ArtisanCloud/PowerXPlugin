import fs from "node:fs";
import path from "node:path";
import YAML from "yaml";
import { TelemetryEmitter } from "../../lib/telemetry/emitter";

export interface CapabilitiesQuotaOptions {
  capabilityId?: string;
  tenantId: string;
  baseUrl?: string;
  token?: string;
  qps?: number;
  burst?: number;
  limits?: number;
  dataScope?: string;
  rootDir?: string;
  manifestPath?: string;
}

interface QuotaResponse {
  id: string;
  tenantId: string;
  capabilityId: string;
  qps: number;
  burst: number;
  limits: number;
  dataScope?: string;
}

function sanitize(name: string) {
  return name.replace(/[^A-Za-z0-9._-]/g, "-");
}

function resolveBaseUrl(options: CapabilitiesQuotaOptions) {
  return (
    options.baseUrl ||
    process.env.PX_DEV_API_BASEURL ||
    "http://127.0.0.1:8077"
  ).replace(/\/$/, "");
}

function resolveToken(options: CapabilitiesQuotaOptions) {
  return options.token || process.env.PX_DEV_API_TOKEN || "";
}

interface ManifestContext {
  manifestPath: string;
  rootDir: string;
  capabilityIds: string[];
}

function loadManifestContext(manifestPath: string): ManifestContext {
  const resolved = path.resolve(process.cwd(), manifestPath);
  if (!fs.existsSync(resolved)) {
    throw new Error(`manifest not found: ${manifestPath}`);
  }
  const raw = fs.readFileSync(resolved, "utf8");
  const parsed = YAML.parse(raw) ?? {};
  const provides: any[] = Array.isArray(parsed?.capabilities?.provides)
    ? parsed.capabilities.provides
    : [];
  const capabilityIds = provides
    .map((entry) => {
      if (typeof entry === "string") {
        return entry.trim();
      }
      if (entry && typeof entry === "object") {
        return String(entry.id ?? "").trim();
      }
      return "";
    })
    .filter((value) => value.length > 0);
  return {
    manifestPath: resolved,
    rootDir: path.dirname(resolved),
    capabilityIds,
  };
}

function resolveCapabilityId(
  provided: string | undefined,
  manifestCtx?: ManifestContext,
) {
  if (provided?.trim()) {
    const trimmed = provided.trim();
    if (
      manifestCtx &&
      manifestCtx.capabilityIds.length > 0 &&
      !manifestCtx.capabilityIds.includes(trimmed)
    ) {
      throw new Error(
        `capabilityId ${trimmed} 不存在 ${manifestCtx.manifestPath}，请确认 manifest.capabilities.provides`,
      );
    }
    return trimmed;
  }
  if (!manifestCtx) {
    throw new Error("capabilityId is required");
  }
  if (manifestCtx.capabilityIds.length === 0) {
    throw new Error(
      `manifest ${manifestCtx.manifestPath} 缺少 capabilities.provides，请使用 --capability-id 显式指定`,
    );
  }
  if (manifestCtx.capabilityIds.length > 1) {
    const preview = manifestCtx.capabilityIds.slice(0, 5).join(", ");
    throw new Error(
      `manifest ${manifestCtx.manifestPath} 含多个 capability，请使用 --capability-id 选择其一（可选: ${preview}${manifestCtx.capabilityIds.length > 5 ? " ..." : ""})`,
    );
  }
  return manifestCtx.capabilityIds[0];
}

function resolveRootDir(
  options: CapabilitiesQuotaOptions,
  manifestCtx?: ManifestContext,
) {
  if (options.rootDir) {
    return path.resolve(process.cwd(), options.rootDir);
  }
  return manifestCtx?.rootDir ?? process.cwd();
}

async function postJSON(
  url: string,
  token: string,
  payload: Record<string, any>,
) {
  const resp = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(payload),
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`请求失败 ${resp.status}: ${text}`);
  }
  return resp.json();
}

function writeSamples(
  capabilityId: string,
  tenantId: string,
  request: Record<string, any>,
  rootDir = process.cwd(),
) {
  const baseDir = path.resolve(
    rootDir,
    "dist",
    "capabilities",
    sanitize(capabilityId),
    "samples",
  );
  fs.mkdirSync(baseDir, { recursive: true });

  const postman = {
    info: {
      name: `${capabilityId} quota for ${tenantId}`,
      schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
    },
    item: [
      {
        name: `quota ${tenantId}`,
        request: {
          method: "POST",
          header: [
            { key: "Content-Type", value: "application/json" },
            ...(request.headers?.Authorization
              ? [{ key: "Authorization", value: request.headers.Authorization }]
              : []),
          ],
          body: {
            mode: "raw",
            raw: JSON.stringify(request.body, null, 2),
          },
          url: request.url,
        },
      },
    ],
  };

  fs.writeFileSync(
    path.join(
      baseDir,
      `tenant-${sanitize(tenantId)}-quota.postman.json`,
    ),
    JSON.stringify(postman, null, 2),
    "utf8",
  );

  const httpExample = [
    `POST ${request.url}`,
    "Content-Type: application/json",
    request.headers?.Authorization
      ? `Authorization: ${request.headers.Authorization}`
      : "",
    "",
    JSON.stringify(request.body, null, 2),
  ]
    .filter(Boolean)
    .join("\n");

  fs.writeFileSync(
    path.join(baseDir, `tenant-${sanitize(tenantId)}-quota.http`),
    httpExample,
    "utf8",
  );
}

export async function runCapabilitiesQuotaCommand(
  options: CapabilitiesQuotaOptions,
) {
  if (!options?.tenantId) {
    throw new Error("tenantId is required");
  }

  const manifestCtx = options.manifestPath
    ? loadManifestContext(options.manifestPath)
    : undefined;
  const capabilityId = resolveCapabilityId(options.capabilityId, manifestCtx);
  const rootDir = resolveRootDir(options, manifestCtx);

  const payload = {
    capabilityId,
    tenantId: options.tenantId,
    qps: options.qps ?? 10,
    burst: options.burst ?? 20,
    limits: options.limits ?? 1000,
    dataScope: options.dataScope ?? "default",
  };

  const baseUrl = resolveBaseUrl(options);
  const token = resolveToken(options);
  const url = `${baseUrl}/internal/plugins/capabilities/${encodeURIComponent(
    capabilityId,
  )}/tenants/${encodeURIComponent(options.tenantId)}/quota`;

  const response = (await postJSON(url, token, payload)) as QuotaResponse;
  writeSamples(
    capabilityId,
    options.tenantId,
    {
      url,
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: payload,
    },
    rootDir,
  );
  TelemetryEmitter.emitCapabilityEvent({
    type: "capability.cli.quota_total",
    capabilityId,
    tenantId: options.tenantId,
    qps: payload.qps,
    burst: payload.burst,
    limits: payload.limits,
    dataScope: payload.dataScope,
  });
  return response;
}
