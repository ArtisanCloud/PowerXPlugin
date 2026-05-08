export type FrameworkLogLevel = "debug" | "info" | "warn" | "error";

export interface FrameworkLoggerContext {
  component?: string;
  mode?: string;
  traceId?: string;
  requestId?: string;
  topic?: string;
  tenantUuid?: string;
  [key: string]: unknown;
}

export interface FrameworkLogger {
  debug: (message: string, context?: FrameworkLoggerContext) => void;
  info: (message: string, context?: FrameworkLoggerContext) => void;
  warn: (message: string, context?: FrameworkLoggerContext) => void;
  error: (message: string, context?: FrameworkLoggerContext) => void;
}

export interface FrameworkLoggerOptions {
  enabled?: boolean;
  minLevel?: FrameworkLogLevel;
}

const loggerOptions: FrameworkLoggerOptions = {};

function getRuntimePublicConfig(): Record<string, any> {
  try {
    const nuxtConfig = (globalThis as any)?.__NUXT__?.config?.public;
    return (nuxtConfig && typeof nuxtConfig === "object") ? nuxtConfig : {};
  } catch {
    return {};
  }
}

function toBool(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  const normalized = String(value ?? "").trim().toLowerCase();
  return normalized === "1" || normalized === "true" || normalized === "yes" || normalized === "on";
}

function rank(level: FrameworkLogLevel): number {
  switch (level) {
    case "debug":
      return 10;
    case "info":
      return 20;
    case "warn":
      return 30;
    case "error":
      return 40;
    default:
      return 20;
  }
}

function resolveMinLevel(): FrameworkLogLevel {
  if (loggerOptions.minLevel) return loggerOptions.minLevel;
  const pub = getRuntimePublicConfig();
  const raw = String(pub.frameworkLogLevel || "").trim().toLowerCase();
  if (raw === "debug" || raw === "info" || raw === "warn" || raw === "error") {
    return raw;
  }
  return "info";
}

function isLogEnabled(): boolean {
  if (typeof loggerOptions.enabled === "boolean") return loggerOptions.enabled;
  const pub = getRuntimePublicConfig();
  const raw = pub.frameworkLogEnabled;
  if (raw === undefined || raw === null || raw === "") return true;
  return toBool(raw);
}

function normalizeContext(context?: FrameworkLoggerContext): Record<string, unknown> {
  if (!context) return {};
  const output: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(context)) {
    if (value === undefined || value === null || value === "") continue;
    output[key] = value;
  }
  return output;
}

function emit(level: FrameworkLogLevel, message: string, context?: FrameworkLoggerContext) {
  if (typeof console === "undefined") return;
  if (!isLogEnabled()) return;
  if (rank(level) < rank(resolveMinLevel())) return;
  const payload = {
    at: new Date().toISOString(),
    level,
    system: "powerx",
    scope: "framework.frontend",
    message,
    ...normalizeContext(context),
  };
  const method = level === "debug" ? "debug" : level;
  const target = (console as any)[method] || console.log;
  target("[framework]", payload);
}

export function createFrameworkLogger(component = "framework-client"): FrameworkLogger {
  const withComponent = (context?: FrameworkLoggerContext) => ({
    component,
    ...context,
  });
  return {
    debug: (message, context) => emit("debug", message, withComponent(context)),
    info: (message, context) => emit("info", message, withComponent(context)),
    warn: (message, context) => emit("warn", message, withComponent(context)),
    error: (message, context) => emit("error", message, withComponent(context)),
  };
}

export function setFrameworkLoggerOptions(options?: FrameworkLoggerOptions) {
  loggerOptions.enabled = options?.enabled;
  loggerOptions.minLevel = options?.minLevel;
}
