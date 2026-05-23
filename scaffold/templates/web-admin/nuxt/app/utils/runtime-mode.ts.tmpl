export type FrontendRuntimeMode = "host_delegated" | "local_proxy" | "standalone_local";

export interface FrontendRuntimeModeState {
  mode: FrontendRuntimeMode;
  insidePowerX: boolean;
  powerxProxy: "0" | "1";
  iamMode: "local" | "delegated";
  gatewayAuthScheme: "bearer" | "apikey" | "";
}

function toBool(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "1" || normalized === "true";
  }
  return false;
}

function normalizeIAMMode(value: unknown): "local" | "delegated" {
  const normalized = String(value || "").trim().toLowerCase();
  return normalized === "delegated" ? "delegated" : "local";
}

function normalizeGatewayAuthScheme(value: unknown): "bearer" | "apikey" | "" {
  const normalized = String(value || "").trim().toLowerCase();
  if (normalized === "bearer") return "bearer";
  if (normalized === "apikey" || normalized === "api_key" || normalized === "api-key") return "apikey";
  return "";
}

export function resolveFrontendRuntimeMode(): FrontendRuntimeModeState {
  const runtimeConfig = useRuntimeConfig();
  const pub = runtimeConfig.public || {};
  const insidePowerX = toBool(pub.insidePowerX);
  const iamMode = normalizeIAMMode(pub.iamMode);
  const powerxProxy: "0" | "1" = toBool(pub.powerxProxy) || insidePowerX ? "1" : "0";
  const gatewayAuthScheme = normalizeGatewayAuthScheme(pub.gatewayAuthScheme);

  let mode: FrontendRuntimeMode = "standalone_local";
  if (iamMode === "delegated") {
    mode = "host_delegated";
  } else if (powerxProxy === "1" && iamMode === "local") {
    mode = "local_proxy";
  }

  return {
    mode,
    insidePowerX,
    powerxProxy,
    iamMode,
    gatewayAuthScheme,
  };
}
