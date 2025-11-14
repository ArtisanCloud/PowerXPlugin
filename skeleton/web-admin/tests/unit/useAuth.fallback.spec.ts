import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ref } from "vue";

const refreshMock = vi.fn();
const logoutMock = vi.fn();

vi.mock("~/composables/api/services/authService", () => ({
  useAuthService: () => ({
    refreshToken: refreshMock,
    logout: logoutMock,
  }),
}));

vi.mock("~/stores/user", () => ({
  useUserStore: () => ({ clearUserState: vi.fn() }),
}));

const loadAuth = async () => (await import("~/composables/useAuth")).useAuth();

const sampleTokens = () => ({
  access_token: "access-token",
  refresh_token: "refresh-token",
  token_type: "Bearer",
  expires_in: 3600,
  scope: "basic",
});

describe("useAuth", () => {
  let localStore: Record<string, string>;
  let cookieJar = "";

  beforeEach(() => {
    vi.resetModules();

    const stateMap = new Map<string, ReturnType<typeof ref>>();
    vi.stubGlobal("useState", (key: string, init?: () => any) => {
      if (!stateMap.has(key)) {
        stateMap.set(key, ref(init ? init() : null));
      }
      return stateMap.get(key)!;
    });
    vi.stubGlobal("readonly", (val: any) => val);
    vi.stubGlobal("onBeforeUnmount", vi.fn());

    vi.stubGlobal("navigateTo", vi.fn());
    vi.stubGlobal("sessionStorage", { clear: vi.fn() } as any);

    localStore = {};
    const localStorageMock = {
      getItem: vi.fn((key: string) => localStore[key] ?? null),
      setItem: vi.fn((key: string, val: string) => {
        localStore[key] = val;
      }),
      removeItem: vi.fn((key: string) => {
        delete localStore[key];
      }),
      clear: vi.fn(() => {
        localStore = {};
      }),
    };

    vi.stubGlobal("window", {
      localStorage: localStorageMock,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    } as any);
    vi.stubGlobal("localStorage", localStorageMock);

    const documentMock = {
      get cookie() {
        return cookieJar;
      },
      set cookie(val: string) {
        cookieJar = val;
      },
      get location() {
        return { hostname: "localhost" };
      },
    };
    vi.stubGlobal("document", documentMock);
    cookieJar = "";

    vi.stubGlobal("process", { client: true });

    refreshMock.mockReset();
    logoutMock.mockReset();
    refreshMock.mockResolvedValue({ success: false });
    logoutMock.mockResolvedValue({ success: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("falls back to cookie token when localStorage access_token 读取失败", async () => {
    const auth = await loadAuth();
    auth.setAuth(sampleTokens() as any);
    (window.localStorage.getItem as any).mockImplementation((key: string) => {
      if (key === "access_token") {
        throw new Error("blocked");
      }
      return localStore[key] ?? null;
    });
    document.cookie = "token=access-token";

    auth.initAuth();

    expect(auth.token.value).toBe("access-token");
    expect(auth.isAuthenticated.value).toBe(true);
  });

  it("强制清理过期 token 并要求重新登录", async () => {
    const auth = await loadAuth();
    auth.setAuth(sampleTokens() as any);
    auth.expiresAt.value = Date.now() - 10_000;

    const value = auth.getToken();

    expect(value).toBeNull();
    expect(auth.isAuthenticated.value).toBe(false);
    expect(auth.token.value).toBeNull();
  });

  it("ensureFreshToken 会在过期时调用 refreshToken 并落盘新 token", async () => {
    const auth = await loadAuth();
    auth.setAuth(sampleTokens() as any);
    auth.expiresAt.value = Date.now() - 5_000;
    refreshMock.mockResolvedValueOnce({
      success: true,
      data: {
        token_type: "Bearer",
        access_token: "next-token",
        refresh_token: "next-refresh",
        expires_in: 1200,
        scope: "basic",
      },
    });

    const token = await auth.ensureFreshToken();

    expect(refreshMock).toHaveBeenCalledWith({ refreshToken: "refresh-token" });
    expect(token).toBe("next-token");
    expect(window.localStorage.setItem).toHaveBeenCalledWith(
      "access_token",
      "next-token"
    );
  });

  it("refreshToken 失败时清理凭证，强制重新登录", async () => {
    const auth = await loadAuth();
    auth.setAuth(sampleTokens() as any);
    auth.expiresAt.value = Date.now() - 5_000;
    refreshMock.mockResolvedValueOnce({ success: false });

    const token = await auth.ensureFreshToken();

    expect(token).toBeNull();
    expect(auth.token.value).toBeNull();
    expect(auth.refreshToken.value).toBeNull();
  });

  it("logout 会调用后端并清空所有存储", async () => {
    const auth = await loadAuth();
    auth.setAuth(sampleTokens() as any);

    await auth.logout();

    expect(logoutMock).toHaveBeenCalledWith("refresh-token");
    expect(auth.token.value).toBeNull();
    expect(sessionStorage.clear).toHaveBeenCalled();
    expect(navigateTo).toHaveBeenCalledWith("/users/login");
  });
});
