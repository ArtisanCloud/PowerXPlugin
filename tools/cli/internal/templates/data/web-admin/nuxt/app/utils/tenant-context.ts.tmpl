import { useCookie } from "#app";

export const TENANT_UUID_STORAGE_KEY = "px_current_tenant_uuid";
const LEGACY_TENANT_UUID_STORAGE_KEY = "tenant_uuid";
const PLACEHOLDER_TENANT_UUIDS = new Set([
  "00000000-0000-0000-0000-000000000000",
]);

const normalize = (value?: string | null) => {
  const normalized = value ? value.trim().toLowerCase() : "";
  if (!normalized || PLACEHOLDER_TENANT_UUIDS.has(normalized)) return "";
  return normalized;
};

const readClientCookie = (name: string) => {
  if (!process.client) return "";
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${name}=([^;]+)`, "i")
  );
  return match ? decodeURIComponent(match[1]) : "";
};

const decodeJwtPayload = (token?: string | null): Record<string, any> | null => {
  if (!token) return null;
  const parts = token.split(".");
  if (parts.length < 2) return null;
  try {
    const padded =
      parts[1].padEnd(parts[1].length + ((4 - (parts[1].length % 4)) % 4), "=");
    const json = atob(padded.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(json);
  } catch {
    return null;
  }
};

export const getStoredTenantUUID = (): string | null => {
  if (process.client) {
    const candidates = [
      localStorage.getItem(TENANT_UUID_STORAGE_KEY),
      localStorage.getItem(LEGACY_TENANT_UUID_STORAGE_KEY),
      readClientCookie(TENANT_UUID_STORAGE_KEY),
      readClientCookie(LEGACY_TENANT_UUID_STORAGE_KEY),
    ];
    for (const candidate of candidates) {
      const normalized = normalize(candidate);
      if (normalized) return normalized;
    }
    return null;
  }

  const legacyCookie = useCookie<string | null>(LEGACY_TENANT_UUID_STORAGE_KEY);
  const cookie = useCookie<string | null>(TENANT_UUID_STORAGE_KEY);
  return normalize(legacyCookie.value) || normalize(cookie.value) || null;
};

export const persistTenantUUID = (tenantUUID: string | null) => {
  const normalized = normalize(tenantUUID);

  if (process.client) {
    if (normalized) {
      localStorage.setItem(TENANT_UUID_STORAGE_KEY, normalized);
    } else {
      localStorage.removeItem(TENANT_UUID_STORAGE_KEY);
      localStorage.removeItem(LEGACY_TENANT_UUID_STORAGE_KEY);
    }

    document.cookie = normalized
      ? `${TENANT_UUID_STORAGE_KEY}=${normalized}; path=/; SameSite=Lax`
      : `${TENANT_UUID_STORAGE_KEY}=; path=/; expires=${new Date(
          0
        ).toUTCString()}; SameSite=Lax`;
    if (!normalized) {
      document.cookie = `${LEGACY_TENANT_UUID_STORAGE_KEY}=; path=/; expires=${new Date(
        0
      ).toUTCString()}; SameSite=Lax`;
    }
  }

  const cookie = useCookie<string | null>(TENANT_UUID_STORAGE_KEY);
  cookie.value = normalized || null;
};

export const resolveTenantUUIDForRequest = () =>
  (() => {
    if (!process.client) return undefined;
    const tokenKeys = ["access_token", "__px_access_token", "auth_token", "token"];
    for (const key of tokenKeys) {
      const claims = decodeJwtPayload(localStorage.getItem(key));
      const tenantUUID =
        (typeof claims?.tid === "string" && claims.tid) ||
        (typeof claims?.tenant_uuid === "string" && claims.tenant_uuid) ||
        (typeof claims?.tenantUuid === "string" && claims.tenantUuid) ||
        "";
      const normalized = normalize(tenantUUID);
      if (normalized) return normalized;
    }
    return undefined;
  })() ||
  getStoredTenantUUID() ||
  undefined;
