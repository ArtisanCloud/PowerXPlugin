import crypto from "node:crypto";
import { promises as fs } from "node:fs";
import path from "node:path";

export interface PublishPipelineOptions {
  manifest: any;
  channel: "stable" | "beta";
  notes: string;
  changelogPath?: string;
}

export interface PublishPipelineResult {
  publishId: string;
  versionId: string;
  reviewQueueId: string;
  uploadUrl: string;
  channel: "stable" | "beta";
  submittedAt: string;
  notes?: string;
}

export async function executePublishPipeline(options: PublishPipelineOptions): Promise<PublishPipelineResult> {
  await attachChangelog(options);
  const publishId = crypto.randomUUID();
  const versionId = `${options.manifest.id}-${options.manifest.version}`;
  const reviewQueueId = crypto.randomUUID();
  const uploadUrl = buildSignedUrl(options.manifest.id, publishId);
  return {
    publishId,
    versionId,
    reviewQueueId,
    uploadUrl,
    channel: options.channel,
    submittedAt: new Date().toISOString(),
    notes: options.notes || undefined,
  };
}

async function attachChangelog(options: PublishPipelineOptions): Promise<void> {
  if (!options.changelogPath) {
    return;
  }
  const resolved = path.resolve(options.changelogPath);
  try {
    const changelog = await fs.readFile(resolved, "utf-8");
    options.notes = `${options.notes}\n\n${changelog}`.trim();
  } catch (error) {
    // 非阻断：若 changelog 不存在只打印提醒
    console.warn(`publish pipeline: unable to read changelog ${resolved}:`, error);
  }
}

function buildSignedUrl(pluginId: string, publishId: string): string {
  const base = "https://upload.marketplace.powerx.local";
  return `${base}/plugins/${encodeURIComponent(pluginId)}/uploads/${publishId}`;
}
