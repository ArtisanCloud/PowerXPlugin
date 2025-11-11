import path from "node:path";
import { ScaffoldExecutor } from "../../executors/scaffold";

export interface PluginInitCommandOptions {
  pluginId: string;
  directory?: string;
  modulePath?: string;
  preset?: string;
  templateRoot?: string;
  force?: boolean;
  installDeps?: boolean;
  sbomPath?: string;
  publishManifestPath?: string;
  version?: string;
  goVersion?: string;
}

/**
 * runPluginInitCommand is the Node/TypeScript entry point for `px-plugin init`.
 * It wires CLI flags to the ScaffoldExecutor which knows how to render templates,
 * emit SBOM metadata and prepare publish manifests.
 */
export async function runPluginInitCommand(options: PluginInitCommandOptions) {
  if (!options?.pluginId) {
    throw new Error("pluginId is required");
  }

  const executor = new ScaffoldExecutor({
    pluginId: options.pluginId,
    targetDir: options.directory ? path.resolve(options.directory) : undefined,
    modulePath: options.modulePath,
    preset: options.preset,
    templateRoot: options.templateRoot,
    force: options.force,
    installDeps: options.installDeps,
    sbomPath: options.sbomPath,
    publishManifestPath: options.publishManifestPath,
    version: options.version,
    goVersion: options.goVersion,
  });

  return executor.run();
}
