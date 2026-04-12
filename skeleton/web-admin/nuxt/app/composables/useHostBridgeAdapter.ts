// app/composables/useHostBridgeAdapter.ts
import { initPowerXBridge } from "~/bridge/powerx-bridge-client";
import { useI18n, useRuntimeConfig } from "#imports";
import { useTheme } from "~/composables/useTheme";
import { useAuth } from "~/composables/useAuth";
import type { LoginResponse } from "~/composables/api/services/authService";
import { useHostCtxStore } from "~/stores/hostCtx";

type BridgeOptions = { pluginId?: string; instanceId?: string; debug?: boolean };

/** 将宿主广播适配到项目内现有的语言/主题切换实现 */
export function setupHostBridgeAdapter(opts: BridgeOptions = {}) {
  const { setLocale, locale } = useI18n();
  const { setTheme } = useTheme(); // ← 不再解构 currentTheme
  const runtimeConfig = useRuntimeConfig();
  const auth = useAuth();
  const hostCtxStore = useHostCtxStore();

  // 宿主 'system' ↔ 本地 'auto'
  const fromHostTheme = (t: string) => (t === "system" ? "auto" : t);

  const applyLocale = async (code: string) => {
    if (!code || code === String(locale.value)) return;
    await setLocale(code);
  };

  const applyTheme = (t: string) => {
    setTheme(fromHostTheme(t) as any);
  };

  const defaultDebug =
    typeof runtimeConfig.public?.bridgeDebug === "boolean"
      ? runtimeConfig.public.bridgeDebug
      : import.meta.dev;
  const resolvedPluginId =
    opts.pluginId ||
    String(runtimeConfig.public?.powerxPluginId || "").trim() ||
    "com.powerx.plugins.base";
  const shouldLog =
    typeof opts.debug === "boolean" ? opts.debug : defaultDebug;

  if (shouldLog) {
    console.info("[Bridge][Plugin] debug mode enabled");
  }

  const applyAuthToken = (payload: {
    accessToken?: string;
    access_token?: string;
    refreshToken?: string;
    refresh_token?: string;
    tokenType?: string;
    token_type?: string;
    expiresIn?: number;
    expires_in?: number;
    expiresAt?: number;
    expires_at?: number;
    scope?: string;
    pluginId?: string;
    plugin_id?: string;
    ctx?: string;
    ctx_sig?: string;
    ctxSig?: string;
    ctxJwt?: string;
    ctx_jwt?: string;
    hostOrigin?: string;
    host_origin?: string;
  }) => {
    const accessToken =
      payload?.accessToken ||
      payload?.access_token ||
      payload?.ctxJwt ||
      payload?.ctx_jwt;
    const refreshToken = payload?.refreshToken || payload?.refresh_token;
    const tokenType = payload?.tokenType || payload?.token_type || "Bearer";
    const payloadPluginID = payload?.pluginId || payload?.plugin_id;
    const payloadCtxSig = payload?.ctxSig || payload?.ctx_sig;
    const payloadCtxJwt = payload?.ctxJwt || payload?.ctx_jwt;
    const payloadHostOrigin = payload?.hostOrigin || payload?.host_origin;
    const payloadExpiresAt = payload?.expiresAt || payload?.expires_at;
    let payloadExpiresIn = payload?.expiresIn || payload?.expires_in;
    if (!accessToken) {
      console.warn(
        "[Bridge][Plugin] 收到 auth-token 但 accessToken 为空，已忽略",
      );
      return;
    }

    // 优先使用 expiresIn；若缺失则尝试用 expiresAt 推导，保证有最小有效期
    let expiresIn = payloadExpiresIn;
    if ((!expiresIn || expiresIn <= 0) && payloadExpiresAt) {
      expiresIn = Math.max(
        1,
        Math.floor((payloadExpiresAt - Date.now()) / 1000),
      );
    }
    if (!expiresIn || expiresIn <= 0) {
      expiresIn = 300; // 缺省 5 分钟，避免立刻过期
    }

    const pluginOrigin =
      typeof window !== "undefined" ? window.location.origin : "plugin";
    const storePluginId =
      payloadPluginID || resolvedPluginId;
    const ctxKey = `${pluginOrigin}::${storePluginId}`;
    if (shouldLog) {
      console.info("[Bridge][Plugin] applyAuthToken storing ctx", {
        key: ctxKey,
        hasCtx: Boolean(payload.ctx),
        hasCtxSig: Boolean(payloadCtxSig),
        hasCtxJwt: Boolean(payloadCtxJwt),
      });
    }
    hostCtxStore.setCtx(ctxKey, {
      token: accessToken,
      refreshToken,
      tokenType,
      tenantUuid: undefined,
      ctx: payload.ctx,
      ctxSig: payloadCtxSig,
      ctxJwt: payloadCtxJwt,
      hostOrigin: payloadHostOrigin,
      expiresAt: payloadExpiresAt,
      expiresIn: payloadExpiresIn,
      scope: payload.scope,
    });

    const authPayload: LoginResponse = {
      access_token: accessToken,
      refresh_token: refreshToken || "",
      token_type: tokenType,
      expires_in: expiresIn,
      scope: payload.scope || "powerx",
    };
    if (shouldLog) {
      console.info("[Bridge][Plugin] applyAuthToken -> setAuth", {
        pluginId: payloadPluginID,
        expiresIn,
        token: `${accessToken.slice(0, 4)}...${accessToken.slice(-4)}`,
      });
    }
    auth.setAuth(authPayload);
    if (shouldLog) {
      try {
        const stored = localStorage.getItem("access_token");
        console.info(
          "[Bridge][Plugin] after setAuth localStorage.access_token",
          stored ? `${stored.slice(0, 4)}...${stored.slice(-4)}` : "<none>",
        );
        console.info(
          "[Bridge][Plugin] hostCtx snapshot",
          hostCtxStore.registry[ctxKey],
        );
      } catch {}
    }
  };

  const bridge = initPowerXBridge({
    debug: shouldLog,
    pluginId: resolvedPluginId,
    instanceId: opts.instanceId ?? "dev-bridge",
    allowedOrigins: ["*"],
    // allowedOrigins: import.meta.env.DEV ? ['*'] : ['https://admin.powerx.cloud'],
    onLocale: (code) => applyLocale(code),
    onTheme: (t) => applyTheme(t),
    onSync: ({ locale, theme }) => {
      applyLocale(locale);
      applyTheme(theme);
    },
    onAuthToken: (p) => applyAuthToken(p),
  });

  return { bridge };
}
