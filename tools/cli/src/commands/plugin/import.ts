import path from "node:path";
import { ImportGuardExecutor, ImportGuardOptions } from "../../executors/import_guard";

export interface PluginImportCommandOptions extends ImportGuardOptions {
  directory?: string;
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
