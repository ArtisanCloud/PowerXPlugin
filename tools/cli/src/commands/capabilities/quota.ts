import fs from "node:fs";
import path from "node:path";
import { TelemetryEmitter } from "../../lib/telemetry/emitter";

export interface CapabilitiesQuotaOptions {
  capabilityId: string;
  tenantId: string;
  baseUrl?: string;
  token?: string;
  qps?: number;
  burst?: number;
  limits?: number;
  dataScope?: string;
  rootDir?: string;
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
  if (!options?.capabilityId) {
    throw new Error("capabilityId is required");
  }
  if (!options?.tenantId) {
    throw new Error("tenantId is required");
  }

  const payload = {
    capabilityId: options.capabilityId,
    tenantId: options.tenantId,
    qps: options.qps ?? 10,
    burst: options.burst ?? 20,
    limits: options.limits ?? 1000,
    dataScope: options.dataScope ?? "default",
  };

  const baseUrl = resolveBaseUrl(options);
  const token = resolveToken(options);
  const url = `${baseUrl}/internal/plugins/capabilities/${encodeURIComponent(
    options.capabilityId,
  )}/tenants/${encodeURIComponent(options.tenantId)}/quota`;

  const response = (await postJSON(url, token, payload)) as QuotaResponse;
  writeSamples(
    options.capabilityId,
    options.tenantId,
    {
      url,
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: payload,
    },
    options.rootDir,
  );
  TelemetryEmitter.emitCapabilityEvent({
    type: "capability.cli.quota_total",
    capabilityId: options.capabilityId,
    tenantId: options.tenantId,
    qps: payload.qps,
    burst: payload.burst,
    limits: payload.limits,
    dataScope: payload.dataScope,
  });
  return response;
}
