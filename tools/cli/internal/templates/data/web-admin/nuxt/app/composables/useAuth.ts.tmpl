import type { LoginResponse } from "~/composables/api/services/authService";
import { useAuthService } from "~/composables/api/services/authService";

const STORAGE_KEYS = [
  "access_token",
  "refresh_token",
  "token_type",
  "expires_in",
  "expires_at",
  "scope",
];

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

const safeLocalStorage = {
  getItem(key: string) {
    if (typeof window === "undefined") return null;
    try {
      return window.localStorage?.getItem(key);
    } catch (err) {
      console.warn("[useAuth] localStorage.getItem failed", err);
      return null;
    }
  },
  setItem(key: string, value: string) {
    if (typeof window === "undefined") return;
    try {
      window.localStorage?.setItem(key, value);
    } catch (err) {
      console.warn("[useAuth] localStorage.setItem failed", err);
    }
  },
  removeItem(key: string) {
    if (typeof window === "undefined") return;
    try {
      window.localStorage?.removeItem(key);
    } catch (err) {
      console.warn("[useAuth] localStorage.removeItem failed", err);
    }
  },
};

export const useAuth = () => {
  const isAuthenticated = useState("auth.isAuthenticated", () => false);
  const user = useState("auth.user", () => null);
  const token = useState<Nullable<string>>("auth.token", () => null);
  const refreshToken = useState<Nullable<string>>("auth.refreshToken", () => null);
  const expiresAt = useState<Nullable<number>>("auth.expiresAt", () => null);

  const { refreshToken: refresh, logout: apiLogout } = useAuthService();

  const persist = (data: LoginResponse) => {
    const expires = Date.now() + data.expires_in * 1000;
    safeLocalStorage.setItem("access_token", data.access_token);
    safeLocalStorage.setItem("refresh_token", data.refresh_token);
    safeLocalStorage.setItem("token_type", data.token_type);
    safeLocalStorage.setItem("expires_in", data.expires_in.toString());
    safeLocalStorage.setItem("scope", data.scope);
    safeLocalStorage.setItem("expires_at", expires.toString());
    writeCookie("token", data.access_token);
    token.value = data.access_token;
    refreshToken.value = data.refresh_token;
    expiresAt.value = expires;
    isAuthenticated.value = true;
  };

  const setAuth = (payload: LoginResponse) => {
    if (process.client) {
      persist(payload);
    }
  };

  const clearAuth = () => {
    if (process.client) {
      STORAGE_KEYS.forEach((key) => safeLocalStorage.removeItem(key));
      writeCookie("token", null);
      sessionStorage?.clear();
      const legacyCookies = [
        "px_token",
        "auth_token",
        "auth-token",
        "i18n_redirected",
      ];
      legacyCookies.forEach((key) => writeCookie(key, null));
      Object.keys(localStorage ?? {}).forEach((key) => {
        if (key.includes("auth") || key.includes("token") || key.includes("px_")) {
          safeLocalStorage.removeItem(key);
        }
      });
    }
    token.value = null;
    refreshToken.value = null;
    expiresAt.value = null;
    isAuthenticated.value = false;
    user.value = null;
  };

  const isTokenExpired = () => {
    if (!process.client) return true;
    const stored = expiresAt.value ?? Number(safeLocalStorage.getItem("expires_at"));
    if (!stored || Number.isNaN(stored)) return true;
    return Date.now() > stored - 5_000;
  };

  const getStoredToken = () => {
    if (!process.client) return null;
    return safeLocalStorage.getItem("access_token") ?? readCookie("token");
  };

  const getToken = () => {
    if (isTokenExpired()) {
      clearAuth();
      return null;
    }
    if (!token.value) {
      token.value = getStoredToken();
    }
    return token.value;
  };

  const syncFromStorage = () => {
    if (!process.client) return;
    const storedToken = getStoredToken();
    const storedRefresh = safeLocalStorage.getItem("refresh_token");
    const storedExpires = safeLocalStorage.getItem("expires_at");
    if (storedToken && storedRefresh && storedExpires) {
      token.value = storedToken;
      refreshToken.value = storedRefresh;
      expiresAt.value = Number(storedExpires);
      isAuthenticated.value = !isTokenExpired();
    } else {
      clearAuth();
    }
  };

  const ensureFreshToken = async () => {
    if (!refreshToken.value) return token.value;
    if (!isTokenExpired()) return token.value;
    try {
      const resp = await refresh({ refreshToken: refreshToken.value });
      if (resp.success) {
        persist(resp.data);
        return resp.data.access_token;
      }
    } catch (error) {
      console.warn("[useAuth] refresh failed", error);
    }
    clearAuth();
    return null;
  };

  const initAuth = () => {
    if (!process.client) return;
    syncFromStorage();
    const handler = (event: StorageEvent) => {
      if (!event.key || !STORAGE_KEYS.includes(event.key)) return;
      syncFromStorage();
    };
    window.addEventListener("storage", handler);
    onBeforeUnmount(() => {
      window.removeEventListener("storage", handler);
    });
  };

  const logout = async () => {
    try {
      if (refreshToken.value) {
        await apiLogout(refreshToken.value);
      }
    } catch (error) {
      console.error("logout API failed", error);
    } finally {
      clearAuth();
      try {
        const { useUserStore } = await import("~/stores/user");
        const userStore = useUserStore();
        userStore?.clearUserState?.();
      } catch (err) {
        console.warn("[useAuth] user store not available", err);
      }
      await navigateTo("/users/login");
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
  };
};
