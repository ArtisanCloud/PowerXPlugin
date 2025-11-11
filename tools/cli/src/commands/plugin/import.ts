import path from "node:path";
import { promises as fs } from "node:fs";
import { ImportGuardExecutor, ImportGuardOptions } from "../../executors/import_guard";

export interface PluginImportCommandOptions extends ImportGuardOptions {
  directory?: string;
}

export interface OfflineMarketplaceImportOptions {
  baseUrl: string;
  packagePath: string;
  integrityPath?: string;
  signaturePath?: string;
  whitelist?: string[];
  notes?: string;
}

export async function runPluginImportCommand(options: PluginImportCommandOptions) {
  const executor = new ImportGuardExecutor({
    sourcePath: options.sourcePath,
    type: options.type,
    provider: options.provider,
    license: options.license,
    projectDir: options.directory ? path.resolve(options.directory) : options.projectDir,
    policyPath: options.policyPath,
    outputPath: options.outputPath,
  });
  return executor.run();
}

export async function runOfflineMarketplaceImport(options: OfflineMarketplaceImportOptions): Promise<any> {
  if (!options.baseUrl) {
    throw new Error("baseUrl is required");
  }
  const pkg = await fs.readFile(path.resolve(options.packagePath));
  const integrity = options.integrityPath
    ? await fs.readFile(path.resolve(options.integrityPath), "utf-8")
    : undefined;
  const signature = options.signaturePath
    ? await fs.readFile(path.resolve(options.signaturePath), "utf-8")
    : undefined;

  const payload = {
    packageName: path.basename(options.packagePath),
    packageContent: pkg.toString("base64"),
    integrity,
    signature,
    whitelist: options.whitelist ?? [],
    notes: options.notes ?? "",
  };

  const resp = await fetch(`${options.baseUrl}/internal/marketplace/offline/import`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "px-plugin-cli/1.0 offline-import",
    },
    body: JSON.stringify(payload),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`offline import failed: ${resp.status} ${resp.statusText} ${text}`);
  }

  return resp.json();
}
