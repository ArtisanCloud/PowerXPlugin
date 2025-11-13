import { ensureCapabilityEntry } from "../../lib/capabilities/manager";
import { TelemetryEmitter } from "../../lib/telemetry/emitter";

export interface CapabilitiesInitCommandOptions {
  manifestPath?: string;
  capabilityId: string;
  version?: string;
  descriptorPath?: string;
  inputSchemaPath?: string;
  outputSchemaPath?: string;
  handlerPath?: string;
  restPath?: string;
  method?: string;
  description?: string;
}

export async function runCapabilitiesInitCommand(
  options: CapabilitiesInitCommandOptions,
) {
  if (!options?.capabilityId) {
    throw new Error("capabilityId is required");
  }
  const result = ensureCapabilityEntry({
    manifestPath: options.manifestPath,
    capabilityId: options.capabilityId,
    version: options.version,
    descriptorPath: options.descriptorPath,
    inputSchemaPath: options.inputSchemaPath,
    outputSchemaPath: options.outputSchemaPath,
    handlerPath: options.handlerPath,
    restPath: options.restPath,
    method: options.method,
    description: options.description,
  });
  TelemetryEmitter.emitCapabilityEvent({
    type: "capability.cli.init_total",
    capabilityId: options.capabilityId,
    manifestPath: result.manifestPath,
  });
  return result;
}
