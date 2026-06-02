import type { PowerXAuthTokenPayload } from "./bridge";

export interface PowerXHostCtx {
  token?: string;
  refreshToken?: string;
  tokenType?: string;
  tenantUuid?: string;
  ctx?: string;
  ctxSig?: string;
  hostOrigin?: string;
  expiresAt?: number;
  expiresIn?: number;
  scope?: string;
}

export interface PowerXLoginPayload {
  access_token: string;
  refresh_token?: string;
  token_type: string;
  expires_in: number;
  scope?: string;
}

export interface ApplyPowerXHostAuthTokenOptions {
  payload: PowerXAuthTokenPayload | Record<string, any>;
  pluginId: string;
  pluginOrigin?: string;
  storeHostCtx: (key: string, ctx: PowerXHostCtx) => void;
  setAuth: (payload: PowerXLoginPayload) => void;
  fetchUserContext?: (input: {
    force: boolean;
    authToken: string;
    tenantUuid?: string;
  }) => Promise<void>;
  validateIdentity?: () => void;
}

export interface AppliedPowerXHostAuthToken {
  ctxKey: string;
  accessToken: string;
  tenantUuid?: string;
  expiresIn: number;
}

const text = (value: unknown) => String(value ?? "").trim();

const numeric = (value: unknown) => {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
};

export function normalizeBearerToken(token: string) {
  const clean = text(token);
  if (!clean) return "";
  return /^Bearer\s/i.test(clean) ? clean : `Bearer ${clean}`;
}

export function normalizePowerXAuthTokenPayload(payload: PowerXAuthTokenPayload | Record<string, any>) {
  const accessToken = text(payload?.accessToken || payload?.access_token);
  const refreshToken = text(payload?.refreshToken || payload?.refresh_token);
  const tokenType = text(payload?.tokenType || payload?.token_type || "Bearer");
  const scope = text(payload?.scope || "powerx");
  const pluginId = text(payload?.pluginId || payload?.plugin_id);
  const hostOrigin = text(payload?.hostOrigin || payload?.host_origin);
  const tenantUuid = text(payload?.tenantUuid || payload?.tenant_uuid);
  const ctx = text(payload?.ctx || payload?.x_powerx_ctx);
  const ctxSig = text(payload?.ctxSig || payload?.x_powerx_ctx_sig);
  const rawExpiresIn = numeric(payload?.expiresIn ?? payload?.expires_in);
  const rawExpiresAt = numeric(payload?.expiresAt ?? payload?.expires_at);
  let expiresIn = rawExpiresIn;
  if ((!expiresIn || expiresIn <= 0) && rawExpiresAt > 0) {
    expiresIn = Math.floor((rawExpiresAt - Date.now()) / 1000);
  }
  return {
    accessToken,
    refreshToken,
    tokenType,
    scope,
    pluginId,
    hostOrigin,
    tenantUuid,
    ctx,
    ctxSig,
    rawExpiresAt,
    expiresIn,
  };
}

export async function applyPowerXHostAuthToken(
  options: ApplyPowerXHostAuthTokenOptions,
): Promise<AppliedPowerXHostAuthToken> {
  const normalized = normalizePowerXAuthTokenPayload(options.payload);
  if (!normalized.accessToken) {
    throw new Error("PowerX host auth-token missing accessToken");
  }
  if (!normalized.expiresIn || normalized.expiresIn <= 0) {
    throw new Error("PowerX host auth-token missing valid expiresIn/expiresAt");
  }

  const pluginId = normalized.pluginId || options.pluginId;
  if (!pluginId) {
    throw new Error("PowerX host auth-token missing pluginId");
  }

  const pluginOrigin =
    options.pluginOrigin ||
    (typeof window === "undefined" ? "plugin" : window.location.origin);
  const ctxKey = `${pluginOrigin}::${pluginId}`;

  options.storeHostCtx(ctxKey, {
    token: normalized.accessToken,
    refreshToken: normalized.refreshToken || undefined,
    tokenType: normalized.tokenType,
    tenantUuid: normalized.tenantUuid || undefined,
    ctx: normalized.ctx || undefined,
    ctxSig: normalized.ctxSig || undefined,
    hostOrigin: normalized.hostOrigin || undefined,
    expiresAt: normalized.rawExpiresAt > 0 ? normalized.rawExpiresAt : undefined,
    expiresIn: normalized.expiresIn,
    scope: normalized.scope,
  });

  options.setAuth({
    access_token: normalized.accessToken,
    refresh_token: normalized.refreshToken || "",
    token_type: normalized.tokenType || "Bearer",
    expires_in: normalized.expiresIn,
    scope: normalized.scope || "powerx",
  });

  if (options.fetchUserContext) {
    await options.fetchUserContext({
      force: true,
      authToken: normalized.accessToken,
      ...(normalized.tenantUuid ? { tenantUuid: normalized.tenantUuid } : {}),
    });
  }
  options.validateIdentity?.();

  return {
    ctxKey,
    accessToken: normalized.accessToken,
    tenantUuid: normalized.tenantUuid || undefined,
    expiresIn: normalized.expiresIn,
  };
}
