import { lintCapabilities } from "../../lib/capabilities/manager";

export interface CapabilitiesLintCommandOptions {
  manifestPath?: string;
}

export async function runCapabilitiesLintCommand(
  options: CapabilitiesLintCommandOptions = {},
) {
  const result = lintCapabilities({
    manifestPath: options.manifestPath,
  });
  if (result.errors.length) {
    const message = [
      `capability lint 发现 ${result.errors.length} 个问题:`,
      ...result.errors.map((err) => ` - ${err}`),
    ].join("\n");
    throw new Error(message);
  }
  return result;
}
