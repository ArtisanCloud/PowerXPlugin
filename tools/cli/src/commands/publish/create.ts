import { promises as fs } from "node:fs";
import path from "node:path";

export interface PublishCreateOptions {
  baseUrl: string;
  manifestPath: string;
  channel: "stable" | "beta";
  notes?: string;
  rolloutStrategy?: "canary" | "direct";
  batches?: Array<{ percentage: number; wait?: string }>;
  windowStart?: string;
  windowEnd?: string;
  autoRollback?: boolean;
  dryRun?: boolean;
}

export interface PublishCreateResponse {
  planId: string;
  publishId: string;
  channel: string;
  status: string;
  window?: {
    start?: string;
    end?: string;
  };
}

async function loadManifest(filePath: string) {
  const resolved = path.resolve(filePath);
  const content = await fs.readFile(resolved, "utf-8");
  return JSON.parse(content);
}

export async function runPublishCreateCommand(options: PublishCreateOptions): Promise<PublishCreateResponse> {
  if (!options.baseUrl) {
    throw new Error("baseUrl is required");
  }
  if (!options.manifestPath) {
    throw new Error("manifestPath is required");
  }

  const manifest = await loadManifest(options.manifestPath);
  const payload = {
    manifest,
    channel: options.channel,
    notes: options.notes ?? "",
    rollout: {
      strategy: options.rolloutStrategy ?? "canary",
      batches: options.batches ?? [],
    },
    window: {
      start: options.windowStart,
      end: options.windowEnd,
    },
    autoRollback: options.autoRollback ?? true,
    dryRun: options.dryRun ?? false,
  };

  const response = await fetch(`${options.baseUrl}/internal/publish/create`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "px-plugin-cli/1.0 publish-create",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`publish create failed: ${response.status} ${response.statusText} ${text}`);
  }

  return (await response.json()) as PublishCreateResponse;
}
