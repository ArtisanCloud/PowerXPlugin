import path from "node:path";
import { DoctorExecutor, DoctorExecutorOptions } from "../../executors/doctor";

export interface PluginDoctorCommandOptions extends DoctorExecutorOptions {
  directory?: string;
}

export async function runPluginDoctorCommand(options: PluginDoctorCommandOptions = {}) {
  const executor = new DoctorExecutor({
    projectDir: options.directory ? path.resolve(options.directory) : options.projectDir,
    outputPath: options.outputPath,
    fix: options.fix,
    requiredFlags: options.requiredFlags,
  });
  return executor.run();
}
