import path from "node:path";
import fs from "node:fs";
import YAML from "yaml";
import {
  CapabilityProvideEntry,
  loadManifest,
} from "../../lib/capabilities/manager";
import {
  appendCapabilityAuditLog,
  updateCapabilityStateEntry,
} from "../../lib/capabilities/state";
import { TelemetryEmitter } from "../../lib/telemetry/emitter";

export interface CapabilitiesSubmitOptions {
  manifestPath?: string;
  capabilityId?: string;
  baseUrl?: string;
  token?: string;
  rootDir?: string;
  exposureOnly?: boolean;
}

interface SubmitResponse {
  id: string;
  status: string;
  note?: string;
  submitted_at?: string;
}

function resolveBaseUrl(options: CapabilitiesSubmitOptions) {
  return (
    options.baseUrl ||
    process.env.PX_DEV_API_BASEURL ||
    "http://127.0.0.1:8077"
  ).replace(/\/$/, "");
}

function resolveToken(options: CapabilitiesSubmitOptions) {
  return options.token || process.env.PX_DEV_API_TOKEN || "";
}

function readDescriptor(manifestDir: string, entry: CapabilityProvideEntry) {
  if (!entry.descriptor) {
    return null;
  }
  const descriptorPath = path.resolve(manifestDir, entry.descriptor);
  if (!fs.existsSync(descriptorPath)) {
    return null;
  }
  const raw = fs.readFileSync(descriptorPath, "utf8");
  return YAML.parse(raw);
}

async function postJSON(url: string, token: string, body: Record<string, any>) {
  const resp = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`请求失败 ${resp.status}: ${text}`);
  }
  return resp.json();
}

export async function runCapabilitiesSubmitCommand(
  options: CapabilitiesSubmitOptions,
) {
  const manifestPath = path.resolve(options.manifestPath ?? "plugin.yaml");
  const manifestDir = path.dirname(manifestPath);
  const manifest = loadManifest(manifestPath);
  const provides = manifest?.capabilities?.provides ?? [];
  if (provides.length === 0) {
    throw new Error("manifest 中缺少 capabilities.provides，无法提交。");
  }

  const targetCapabilities = options.capabilityId
    ? provides.filter((entry) => entry.id === options.capabilityId)
    : provides;

  if (!targetCapabilities.length) {
    throw new Error(
      `在 manifest 中找不到 capability ${options.capabilityId ?? "(全部)"}`,
    );
  }

  const baseUrl = resolveBaseUrl(options);
  const token = resolveToken(options);
  const rootDir = options.rootDir ?? process.cwd();
  const results: SubmitResponse[] = [];

  for (const entry of targetCapabilities) {
    const descriptor = readDescriptor(manifestDir, entry);
    const payload = {
      manifestEntry: entry,
      descriptor,
    };
    let registryResp: SubmitResponse | null = null;
    if (!options.exposureOnly) {
      registryResp = (await postJSON(
        `${baseUrl}/internal/plugins/capabilities`,
        token,
        payload,
      )) as SubmitResponse;
      results.push(registryResp);
      updateCapabilityStateEntry(
        {
          id: entry.id,
          status:
            (registryResp.status as SubmitResponse["status"]) ?? "pending",
          lastSubmittedAt:
            registryResp.submitted_at ?? new Date().toISOString(),
          note: registryResp.note,
        },
        rootDir,
      );
      appendCapabilityAuditLog(entry.id, registryResp, rootDir);
    }

    const exposureResp = (await postJSON(
      `${baseUrl}/internal/plugins/capabilities/${encodeURIComponent(entry.id)}/exposure`,
      token,
      {
        manifestEntry: entry,
      },
    )) as SubmitResponse;
    results.push(exposureResp);
    appendCapabilityAuditLog(entry.id, exposureResp, rootDir);
    TelemetryEmitter.emitCapabilityEvent({
      type: "capability.cli.submit_total",
      capabilityId: entry.id,
      registryStatus: registryResp?.status ?? (options.exposureOnly ? "skipped" : "pending"),
      exposureStatus: exposureResp?.status ?? "pending",
      exposureOnly: options.exposureOnly ?? false,
    });
  }

  return {
    submitted: results,
  };
}
