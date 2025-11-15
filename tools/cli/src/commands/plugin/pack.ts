import path from "node:path";
import { promises as fs } from "node:fs";
import { packageOfflineBuild } from "../../lib/dist/offlinePackager";

export interface PackCommandOptions {
  manifestPath: string;
  artefactPaths: string[];
  outputDir?: string;
  channel?: string;
  notes?: string;
  marketplacePublicKeyPem: string;
  keyId: string;
}

export interface PackCommandResult {
  pxpPath: string;
  integrityPath: string;
  reportPath: string;
  auditPath: string;
  releaseManifestPath: string;
}

export async function runPackCommand(options: PackCommandOptions): Promise<PackCommandResult> {
  if (!options.manifestPath) {
    throw new Error("manifestPath is required");
  }
  if (!options.marketplacePublicKeyPem) {
    throw new Error("marketplace public key is required");
  }
  const outputs = await packageOfflineBuild({
    manifestPath: path.resolve(options.manifestPath),
    artefactPaths: options.artefactPaths.map((p) => path.resolve(p)),
    outputDir: options.outputDir ?? path.resolve(process.cwd(), "dist"),
    outputFileName: undefined,
    marketplacePublicKeyPem: options.marketplacePublicKeyPem,
    keyId: options.keyId,
  });

  const releaseManifestPath = path.join(
    options.outputDir ?? path.resolve(process.cwd(), "dist"),
    "release.manifest.json"
  );
  const releaseManifest = {
    channel: options.channel ?? "stable",
    notes: options.notes ?? "",
    package: path.basename(outputs.pxpPath),
    integrityFile: path.basename(outputs.integrityPath),
    generatedAt: new Date().toISOString(),
  };
  await fs.writeFile(releaseManifestPath, JSON.stringify(releaseManifest, null, 2), "utf-8");

  return {
    ...outputs,
    releaseManifestPath,
  };
}
