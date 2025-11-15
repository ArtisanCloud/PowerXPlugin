import { promises as fs } from "node:fs";
import path from "node:path";
import { assertCapabilitiesApproved } from "../lib/capabilities/state";
import { runPrechecks } from "../lib/publish/precheck";
import { executePublishPipeline, PublishPipelineOptions, PublishPipelineResult } from "../lib/publish/pipeline";
import { TelemetryEmitter } from "../lib/telemetry/emitter";

export interface PublishCommandOptions {
  manifestPath: string;
  channel: "stable" | "beta";
  notes?: string;
  receiptPath?: string;
  changelogPath?: string;
}

export async function runPublishCommand(options: PublishCommandOptions): Promise<PublishPipelineResult> {
  assertCapabilitiesApproved({ manifestPath: options.manifestPath });
  const manifest = await loadManifest(options.manifestPath);
  await runPrechecks({
    manifest,
    channel: options.channel,
    notes: options.notes ?? "",
    changelogPath: options.changelogPath,
  });

  const pipelineOptions: PublishPipelineOptions = {
    manifest,
    channel: options.channel,
    notes: options.notes ?? "",
    changelogPath: options.changelogPath,
  };
  const result = await executePublishPipeline(pipelineOptions);
  const receiptTarget = options.receiptPath ?? path.resolve(process.cwd(), "publish-receipt.json");
  await writeReceipt(receiptTarget, result);
  TelemetryEmitter.emitPublishEvent({
    type: "plugin.publish",
    publishId: result.publishId,
    pluginId: manifest.id,
    version: manifest.version,
    channel: options.channel,
    reviewQueueId: result.reviewQueueId,
  });
  return result;
}

async function loadManifest(manifestPath: string): Promise<any> {
  const resolved = path.resolve(manifestPath);
  const data = await fs.readFile(resolved, "utf-8");
  return JSON.parse(data);
}

async function writeReceipt(target: string, receipt: PublishPipelineResult): Promise<void> {
  const dir = path.dirname(target);
  await fs.mkdir(dir, { recursive: true });
  await fs.writeFile(target, JSON.stringify(receipt, null, 2), "utf-8");
}
