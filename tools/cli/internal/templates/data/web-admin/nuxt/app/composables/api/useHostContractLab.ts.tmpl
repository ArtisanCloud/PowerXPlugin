import { apiPost } from "./_client";

export type HostContractModule =
  | "cache" | "taskcenter" | "iam" | "knowledge" | "media" | "agent" | "ai"
  | "capability_registry" | "integration_gateway" | "skills" | "notifications" | "plugin_runtime";

export interface HostContractProbeResult {
  module: HostContractModule;
  operation: string;
  provider_mode: string;
  capability_id: string;
  trace_id?: string;
  reason_code?: string;
  result?: Record<string, unknown>;
  observed_at: string;
}

export interface HostContractProbeError { status?: number; reasonCode: string; traceID?: string }

export function hostContractBindingState(result: HostContractProbeResult): boolean | null {
  return typeof result.result?.adapter_available === "boolean" ? result.result.adapter_available : null;
}

export function useHostContractLabApi() {
  async function probe(module: HostContractModule) {
    return await apiPost<{ data: HostContractProbeResult }>("/admin/host-contract/probe", {
      module,
      operation: "status",
      input: {},
      confirm: false,
    });
  }
  function normalizeError(error: any): HostContractProbeError {
    const payload = error?.data || error?.response?._data || error?._data || {};
    return {
      status: error?.statusCode || error?.status || error?.response?.status,
      reasonCode: String(payload?.error?.reason_code || payload?.reason_code || "HOST_CONTRACT_UPSTREAM_DEPENDENCY"),
      traceID: String(payload?.error?.trace_id || payload?.data?.trace_id || "") || undefined,
    };
  }
  return { probe, normalizeError };
}
