import { apiGet } from "./_client";
import type { ApiResponse } from "./_base";

export interface ProviderModeDiagnostics {
  mode: string;
  provider: string;
  delegated_available: boolean;
  local_available: boolean;
  read_only?: boolean;
  source?: string;
}

export const defaultProviderMode = (): ProviderModeDiagnostics => ({
  mode: "local",
  provider: "local",
  delegated_available: false,
  local_available: false,
  read_only: false,
});

export function normalizeProviderMode(input?: Partial<ProviderModeDiagnostics> | null): ProviderModeDiagnostics {
  const mode = String(input?.mode || input?.provider || "local").trim() === "delegated" ? "delegated" : "local";
  return {
    mode,
    provider: String(input?.provider || mode),
    delegated_available: Boolean(input?.delegated_available),
    local_available: Boolean(input?.local_available),
    read_only: Boolean(input?.read_only || mode === "delegated"),
    source: input?.source,
  };
}

export function fetchProviderMode(path: string, init?: any) {
  return apiGet<ApiResponse<ProviderModeDiagnostics>>(path, undefined, init);
}
