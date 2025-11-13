import path from "node:path";
import { packageOfflineBuild } from "../lib/dist/offlinePackager";
import { assertCapabilitiesApproved } from "../lib/capabilities/state";

export interface DistCommandOptions {
  manifestPath: string;
  artefactPaths: string[];
  outputDir?: string;
  outputFileName?: string;
  marketplacePublicKeyPem: string;
  keyId: string;
}

export async function runDistCommand(options: DistCommandOptions) {
  if (!options.manifestPath) {
    throw new Error("manifestPath is required");
  }
  assertCapabilitiesApproved({ manifestPath: options.manifestPath });
  if (!options.marketplacePublicKeyPem) {
    throw new Error("marketplace public key is required");
  }
  const outputs = await packageOfflineBuild({
    manifestPath: path.resolve(options.manifestPath),
    artefactPaths: options.artefactPaths.map((p) => path.resolve(p)),
    outputDir: options.outputDir ?? path.resolve(process.cwd(), "dist"),
    outputFileName: options.outputFileName,
    marketplacePublicKeyPem: options.marketplacePublicKeyPem,
    keyId: options.keyId,
  });
  return outputs;
}
