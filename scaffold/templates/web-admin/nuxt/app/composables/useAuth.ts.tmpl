import { getCurrentScope, onScopeDispose, readonly } from "vue";
import { useRuntimeConfig } from "#imports";
import type { LoginResponse } from "~/composables/api/services/authService";
import { useAuthService } from "~/composables/api/services/authService";
import { persistTenantUUID } from "~/utils/tenant-context";

const STORAGE_KEYS = [
  "access_token",
  "__px_access_token",
  "refresh_token",
  "token_type",
  "expires_in",
  "expires_at",
  "scope",
];

const AUTH_ERROR_KEY = "powerx-auth-error";
const REFRESH_COOKIE_KEY = "px_refresh_token";
const EXPIRES_AT_COOKIE_KEY = "px_expires_at";
let RUNTIME_TOKEN_CACHE: string | null = null;
let RUNTIME_REFRESH_CACHE: string | null = null;

const getStableLocalStorage = (): Storage | null => {
  if (typeof window === "undefined") return null;
  const w = window as any;
  if (w.__PX_STABLE_LOCAL_STORAGE__) return w.__PX_STABLE_LOCAL_STORAGE__ as Storage;
  try {
    const ls = window.localStorage;
    if (ls) {
      w.__PX_STABLE_LOCAL_STORAGE__ = ls;
      return ls;
    }
  } catch {}
  return null;
};

type Nullable<T> = T | null;

const readCookie = (name: string) => {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${name}=([^;]+)`, "i")
  );
  return match ? decodeURIComponent(match[1]) : null;
};

const writeCookie = (name: string, value: string | null) => {
  if (typeof document === "undefined") return;
  if (!value) {
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/;`;
    return;
  }
  document.cookie = `${name}=${encodeURIComponent(
    value
  )}; path=/; SameSite=Lax`;
};

const decodeBase64Url = (input: string) => {
  if (!input) return "";
  let output = input.replace(/-/g, "+").replace(/_/g, "/");
  while (output.length % 4 !== 0) {
    output += "=";
  }
  if (typeof atob === "function") {
    return atob(output);
  }
  if (typeof globalThis !== "undefined" && (globalThis as any).Buffer) {
    return (globalThis as any).Buffer.from(output, "base64").toString("utf-8");
  }
  return "";
};

const decodeJwtPayload = (token?: string | null): Record<string, any> | null => {
  if (!token) return null;
  const parts = token.split(".");
  if (parts.length < 2) return null;
  try {
    return JSON.parse(decodeBase64Url(parts[1]));
  } catch {
    return null;
  }
};

const extractTenantUuidFromToken = (token?: string | null) => {
  if (!token) return null;
  const parts = token.split(".");
  if (parts.length < 2) return null;
  try {
    const payload = JSON.parse(decodeBase64Url(parts[1]));
    const candidate =
      payload?.tid ??
      payload?.tenant_uuid ??
      payload?.tenantUuid ??
      payload?.tenantID ??
      payload?.tenantId;
    if (typeof candidate === "string" && candidate.trim() !== "") {
      return candidate.trim();
    }
  } catch (err) {
    console.warn("[useAuth] failed to parse tenant uuid from token", err);
  }
  return null;
};

const storeTenantUuid = (uuid?: string | null) => {
  const normalized = String(uuid || "").trim();
  writeCookie("tenant_uuid", normalized || null);
  persistTenantUUID(normalized || null);
};

const storeTenantUuidFromAuth = (data: LoginResponse, token?: string | null) => {
  const uuid =
    String(data.current_tenant_uuid || "").trim() ||
    String(data.tenant_uuid || "").trim() ||
    extractTenantUuidFromToken(token);
  storeTenantUuid(uuid);
};

const safeLocalStorage = {
  getItem(key: string) {
    if (typeof window === "undefined") return null;
    try {
      const ls = getStableLocalStorage();
      return ls?.getItem(key) ?? null;
    } catch (err) {
      console.warn("[useAuth] localStorage.getItem failed", err);
      return null;
    }
  },
  setItem(key: string, value: string) {
    if (typeof window === "undefined") return;
    try {
      const ls = getStableLocalStorage();
      ls?.setItem(key, value);
    } catch (err) {
      console.warn("[useAuth] localStorage.setItem failed", err);
    }
  },
  removeItem(key: string) {
    if (typeof window === "undefined") return;
    try {
      const ls = getStableLocalStorage();
      ls?.removeItem(key);
    } catch (err) {
      console.warn("[useAuth] localStorage.removeItem failed", err);
    }
  },
};

const resolveInsidePowerX = (value: unknown) => {
  if (value === true) return true;
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "1" || normalized === "yes";
  }
  return false;
};

const resolveDelegatedMode = (value: unknown, fallback: boolean) => {
  if (value === true) return true;
  if (value === false) return false;
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    if (normalized === "true" || normalized === "1" || normalized === "yes") {
      return true;
    }
    if (normalized === "false" || normalized === "0" || normalized === "no") {
      return false;
    }
  }
  return fallback;
};

export const useAuth = () => {
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX = resolveInsidePowerX(runtimeConfig.public?.insidePowerX);
  const delegatedMode = resolveDelegatedMode(
    runtimeConfig.public?.delegatedMode,
    insidePowerX
  );
  // Standalone 模式下宿主/脚手架可能只广播 access token（无 refresh token），允许继续维持会话。
  const allowRefreshlessSession = !delegatedMode;

  const isAuthenticated = useState("auth.isAuthenticated", () => false);
  const user = useState("auth.user", () => null);
  const token = useState<Nullable<string>>("auth.token", () => null);
  const refreshToken = useState<Nullable<string>>("auth.refreshToken", () => null);
  const expiresAt = useState<Nullable<number>>("auth.expiresAt", () => null);
  const lastError = useState<string>("auth.lastError", () => "");
  const hasAuthenticated = useState("auth.hasAuthenticated", () => false);
  const delegatedAuthError = useState<string>("auth.delegatedError", () => "");
  const localIAMEnabled = useState("auth.localIAMEnabled", () => !delegatedMode);
  const delegatedIAM = useState("auth.delegatedIAM", () => delegatedMode);

  const { refreshToken: refresh, logout: apiLogout } = useAuthService();
  const setTokenState = (
    nextToken: Nullable<string>,
    nextRefresh: Nullable<string>,
    nextExpiresAt: Nullable<number>,
    _from: string
  ) => {
    token.value = nextToken;
    refreshToken.value = nextRefresh;
    expiresAt.value = nextExpiresAt;
    if (nextToken) RUNTIME_TOKEN_CACHE = nextToken;
    if (nextRefresh) RUNTIME_REFRESH_CACHE = nextRefresh;
  };
  const ensureStorageConsistency = (_from: string) => {
    if (!process.client) return;
    const access = String(token.value || RUNTIME_TOKEN_CACHE || "").trim();
    if (!access) return;
    try {
      const hasAccess = Boolean(safeLocalStorage.getItem("access_token"));
      const hasBackup = Boolean(safeLocalStorage.getItem("__px_access_token"));
      if (!hasAccess || !hasBackup) {
        safeLocalStorage.setItem("access_token", access);
        safeLocalStorage.setItem("__px_access_token", access);
        if (refreshToken.value || RUNTIME_REFRESH_CACHE) {
          safeLocalStorage.setItem("refresh_token", String(refreshToken.value || RUNTIME_REFRESH_CACHE));
        }
        if (expiresAt.value) {
          safeLocalStorage.setItem("expires_at", String(expiresAt.value));
        }
      }
    } catch {}
  };

  const persist = (data: LoginResponse) => {
    const access = String(data.access_token || "").trim();
    if (!access) {
      throw new Error("setAuth 失败：access_token 为空");
    }
    const nextRefresh = String(data.refresh_token || "").trim();
    const prevRefresh =
      String(refreshToken.value || safeLocalStorage.getItem("refresh_token") || "").trim();
    const finalRefresh = nextRefresh || prevRefresh;
    const expiresIn = Number(data.expires_in || 0);
    const expires = Date.now() + (expiresIn > 0 ? expiresIn : 3600) * 1000;
    safeLocalStorage.setItem("access_token", access);
    safeLocalStorage.setItem("__px_access_token", access);
    if (finalRefresh) {
      safeLocalStorage.setItem("refresh_token", finalRefresh);
    } else {
      safeLocalStorage.removeItem("refresh_token");
    }
    safeLocalStorage.setItem("token_type", data.token_type);
    safeLocalStorage.setItem("expires_in", String(expiresIn > 0 ? expiresIn : 3600));
    safeLocalStorage.setItem("scope", data.scope);
    safeLocalStorage.setItem("expires_at", expires.toString());
    writeCookie("token", access);
    writeCookie(REFRESH_COOKIE_KEY, finalRefresh || null);
    writeCookie(EXPIRES_AT_COOKIE_KEY, String(expires));
    storeTenantUuidFromAuth(data, access);
    setTokenState(access, finalRefresh || null, expires, "persist");
    isAuthenticated.value = true;
    hasAuthenticated.value = true;
  };

  const setAuth = (payload: LoginResponse) => {
    if (process.client) {
      persist(payload);
      delegatedAuthError.value = "";
      lastError.value = "";
      try {
        sessionStorage?.removeItem(AUTH_ERROR_KEY);
      } catch (err) {
        console.warn("[useAuth] failed to clear auth error", err);
      }
    }
  };

  type ClearAuthReason =
    | "logout"
    | "token_invalid"
    | "missing_auth"
    | "expired"
    | "system";

  const clearAuth = (
    reason: ClearAuthReason = "system",
    purgeStorage = reason === "logout" || reason === "token_invalid" || reason === "expired"
  ) => {
    // system/fail-closed 只重置运行态；只有显式退出或明确 token 无效才清持久化。
    if (process.client && purgeStorage) {
      if (purgeStorage) {
        STORAGE_KEYS.forEach((key) => safeLocalStorage.removeItem(key));
      }
      writeCookie("token", null);
      writeCookie(REFRESH_COOKIE_KEY, null);
      writeCookie(EXPIRES_AT_COOKIE_KEY, null);
      writeCookie("tenant_uuid", null);
      try {
        const preserved = sessionStorage?.getItem(AUTH_ERROR_KEY);
        sessionStorage?.clear();
        if (preserved) {
          sessionStorage?.setItem(AUTH_ERROR_KEY, preserved);
        }
      } catch (err) {
        console.warn("[useAuth] sessionStorage.clear failed", err);
      }
      const legacyCookies = [
        "i18n_redirected",
      ];
      legacyCookies.forEach((key) => writeCookie(key, null));
    }
    setTokenState(null, null, null, `clearAuth:${reason}`);
    if (purgeStorage) {
      RUNTIME_TOKEN_CACHE = null;
      RUNTIME_REFRESH_CACHE = null;
    }
    isAuthenticated.value = false;
    user.value = null;
  };

  const isTokenExpired = () => {
    if (!process.client) return true;
    let stored =
      expiresAt.value ??
      Number(safeLocalStorage.getItem("expires_at")) ??
      Number(readCookie(EXPIRES_AT_COOKIE_KEY));
    if (!stored || Number.isNaN(stored)) {
      const tokenCandidate = token.value || getStoredToken();
      const payload = decodeJwtPayload(tokenCandidate);
      const exp = Number(payload?.exp || 0);
      if (exp > 0) {
        stored = exp * 1000;
      }
    }
    if (!stored || Number.isNaN(stored)) return false;
    return Date.now() > stored - 5_000;
  };

  const readAuthCookieToken = () => {
    const cookieCandidates = insidePowerX ? ["px_ctx_jwt", "token"] : ["token"];
    for (const name of cookieCandidates) {
      const value = readCookie(name);
      if (value) {
        return value;
      }
    }
    return null;
  };

  const getStoredToken = () => {
    if (!process.client) return null;
    const tryLocalStorageGet = (key: string): string | null | undefined => {
      if (typeof window === "undefined") return null;
      try {
        return safeLocalStorage.getItem(key);
      } catch {
        return undefined;
      }
    };

    const stored = tryLocalStorageGet("access_token");
    // If localStorage is blocked (throws), fall back to cookie token.
    if (stored === undefined) return readAuthCookieToken();
    if (stored) return stored;
    if (RUNTIME_TOKEN_CACHE) return RUNTIME_TOKEN_CACHE;
    const backup = tryLocalStorageGet("__px_access_token");
    if (backup) {
      safeLocalStorage.setItem("access_token", backup);
      return backup;
    }

    // If localStorage still contains auth footprint but access_token is gone,
    // treat it as a logout/invalid session and do NOT fall back to cookies.
    // This makes cross-tab logout (storage event) consistent.
    const hasFootprint = Boolean(
      safeLocalStorage.getItem("expires_at") ||
        safeLocalStorage.getItem("refresh_token") ||
        safeLocalStorage.getItem("token_type") ||
        safeLocalStorage.getItem("scope")
    );
    if (hasFootprint) return null;

    return readAuthCookieToken();
  };

  const getStoredRefreshToken = () => {
    const fromLs = safeLocalStorage.getItem("refresh_token");
    if (fromLs) return fromLs;
    const fromCookie = readCookie(REFRESH_COOKIE_KEY);
    if (fromCookie) {
      safeLocalStorage.setItem("refresh_token", fromCookie);
      return fromCookie;
    }
    return RUNTIME_REFRESH_CACHE;
  };

  const getToken = () => {
    ensureStorageConsistency("getToken");
    if (isTokenExpired()) {
      return null;
    }
    if (!token.value) {
      setTokenState(getStoredToken(), refreshToken.value, expiresAt.value, "getToken.restore");
    }
    return token.value;
  };

  const syncFromStorage = () => {
    if (!process.client) return;
    // 在 standalone 下，内存态 token 代表当前会话真值，不允许被一次存储空窗覆盖。
    if (token.value) {
      const storedRefresh = safeLocalStorage.getItem("refresh_token");
      const storedExpires = safeLocalStorage.getItem("expires_at");
      if ((!refreshToken.value && storedRefresh) || (!expiresAt.value && storedExpires)) {
        setTokenState(
          token.value,
          refreshToken.value || storedRefresh || null,
          expiresAt.value || Number(storedExpires) || null,
          "syncFromStorage.backfillFromStorage"
        );
      } else {
      }
      return;
    }

    const storedToken = getStoredToken();
    const storedRefresh = getStoredRefreshToken();
    const storedExpires = safeLocalStorage.getItem("expires_at") || readCookie(EXPIRES_AT_COOKIE_KEY);
    const derivedExpFromToken = (() => {
      if (storedExpires) return Number(storedExpires);
      const payload = decodeJwtPayload(storedToken);
      const exp = Number(payload?.exp || 0);
      return exp > 0 ? exp * 1000 : null;
    })();
    const hasTokenAndExpiry = Boolean(storedToken && derivedExpFromToken);
    const hasRefresh = Boolean(storedRefresh);
    const canRestoreSession =
      hasTokenAndExpiry && (hasRefresh || allowRefreshlessSession);

    if (canRestoreSession) {
      setTokenState(
        storedToken!,
        hasRefresh ? storedRefresh : null,
        Number(derivedExpFromToken),
        "syncFromStorage.restoreSession"
      );
      isAuthenticated.value = !isTokenExpired();
      return;
    }

    // 没有任何会话数据（首次访问/手动清除），无需提示“会话失效”。
    const hasAnySessionData = Boolean(storedToken || storedRefresh || storedExpires);
    if (!hasAnySessionData) {
      if (RUNTIME_TOKEN_CACHE) {
        const payload = decodeJwtPayload(RUNTIME_TOKEN_CACHE);
        const exp = Number(payload?.exp || 0);
        const runtimeExpires = exp > 0 ? exp * 1000 : null;
        setTokenState(
          RUNTIME_TOKEN_CACHE,
          RUNTIME_REFRESH_CACHE,
          runtimeExpires,
          "syncFromStorage.restoreFromRuntimeCache"
        );
        ensureStorageConsistency("syncFromStorage.restoreFromRuntimeCache");
        isAuthenticated.value = !isTokenExpired();
        return;
      }
      setTokenState(null, null, null, "syncFromStorage.noSessionData");
      isAuthenticated.value = false;
      user.value = null;
      hasAuthenticated.value = false;
      return;
    }

    if (delegatedMode && !hasAuthenticated.value) {
      setTokenState(null, null, null, "syncFromStorage.delegatedNoAuthHistory");
      isAuthenticated.value = false;
      user.value = null;
      return;
    }
    failClosed();
  };

  const ensureFreshToken = async () => {
    if (!process.client) return token.value;
    ensureStorageConsistency("ensureFreshToken.begin");
    // 仅在缺少 access token 时才从存储恢复，避免被 refresh token 抖动误伤。
    if (!token.value) {
      syncFromStorage();
    }
    if (!token.value) {
      setTokenState(getStoredToken(), refreshToken.value, expiresAt.value, "ensureFreshToken.restore");
    }
    if (!refreshToken.value) {
      const restoredRefresh = getStoredRefreshToken();
      if (restoredRefresh) {
        setTokenState(token.value, restoredRefresh, expiresAt.value, "ensureFreshToken.restoreRefresh");
      }
      return token.value;
    }
    if (!isTokenExpired()) {
      if (!token.value) {
        setTokenState(getStoredToken(), refreshToken.value, expiresAt.value, "ensureFreshToken.restoreWhenValid");
      }
      return token.value;
    }
    if (token.value && !refreshToken.value) {
      return null;
    }
    try {
      const resp = await refresh({ refreshToken: refreshToken.value });
      if (resp.success) {
        persist(resp.data);
        return resp.data.access_token;
      }
    } catch (error: any) {
      const status = error?.response?.status;
      if (status === 503) {
        failClosed(error?.response?._data?.message || "宿主认证不可用，请稍后重试");
      } else {
        console.warn("[useAuth] refresh failed", error);
      }
    }
    failClosed();
    return null;
  };

  const initAuth = () => {
    if (!process.client) return;
    syncFromStorage();
    ensureStorageConsistency("initAuth.afterSync");
    const handler = (event: StorageEvent) => {
      // `storage` event may fire with `key === null` (e.g. clear()) or when
      // simulated in tests; in that case we still want to reconcile auth state.
      if (event.key && !STORAGE_KEYS.includes(event.key)) return;
      syncFromStorage();
    };
    window.addEventListener("storage", handler);
    if (getCurrentScope()) {
      onScopeDispose(() => {
        window.removeEventListener("storage", handler);
      });
    }
  };

  const setIAMModeFlags = (isDelegated: boolean) => {
    delegatedIAM.value = isDelegated;
    localIAMEnabled.value = !isDelegated;
  };

  const logout = async () => {
    try {
      if (refreshToken.value) {
        await apiLogout(refreshToken.value);
      }
    } catch (error) {
      console.error("logout API failed", error);
    } finally {
      clearAuth("logout", true);
      delegatedAuthError.value = "";
      lastError.value = "";
      hasAuthenticated.value = false;
      try {
        sessionStorage?.removeItem(AUTH_ERROR_KEY);
      } catch (err) {
        console.warn("[useAuth] failed to clear auth error state", err);
      }
      try {
        const { useUserStore } = await import("~/stores/user");
        const userStore = useUserStore();
        userStore?.clearUserState?.();
      } catch (err) {
        console.warn("[useAuth] user store not available", err);
      }
      await navigateTo("/");
    }
  };

  const rememberAuthError = (message?: string) => {
    if (!process.client || !message) return;
    try {
      sessionStorage?.setItem(AUTH_ERROR_KEY, message);
      lastError.value = message;
    } catch (err) {
      console.warn("[useAuth] failed to persist auth error", err);
    }
    if (delegatedMode) {
      delegatedAuthError.value = message;
    }
  };

  const consumeAuthError = () => {
    if (!process.client) return "";
    try {
      const msg = sessionStorage?.getItem(AUTH_ERROR_KEY) || lastError.value;
      if (msg) {
        sessionStorage?.removeItem(AUTH_ERROR_KEY);
        lastError.value = "";
      }
      return msg || "";
    } catch (err) {
      console.warn("[useAuth] failed to read auth error", err);
      return "";
    }
  };

  const failClosed = (message?: string) => {
    clearAuth("system", false);
    const fallbackMessage =
      message ||
      (delegatedMode
        ? "PowerX 会话已失效，请回到宿主重新登录"
        : "会话已失效，请重新登录");
    if (fallbackMessage) {
      rememberAuthError(fallbackMessage);
    }
    if (delegatedMode) {
      return;
    }
    if (
      process.client &&
      typeof window !== "undefined" &&
      !window.location.pathname.startsWith("/users")
    ) {
      const redirect = window.location.pathname + window.location.search;
      navigateTo({ path: "/users/login", query: { redirect } });
    }
  };

  const clearDelegatedError = () => {
    delegatedAuthError.value = "";
    lastError.value = "";
    if (!process.client) return;
    try {
      sessionStorage?.removeItem(AUTH_ERROR_KEY);
    } catch (err) {
      console.warn("[useAuth] failed to clear delegated error", err);
    }
  };

  return {
    isAuthenticated: readonly(isAuthenticated),
    user: readonly(user),
    token,
    refreshToken,
    expiresAt,
    setAuth,
    clearAuth,
    getToken,
    isTokenExpired,
    ensureFreshToken,
    initAuth,
    logout,
    consumeAuthError,
    failClosed,
    rememberAuthError,
    delegatedError: readonly(delegatedAuthError),
    clearDelegatedError,
    restoreFromStorage: syncFromStorage,
    localIAMEnabled: readonly(localIAMEnabled),
    delegatedIAM: readonly(delegatedIAM),
    setIAMModeFlags,
  };
};
