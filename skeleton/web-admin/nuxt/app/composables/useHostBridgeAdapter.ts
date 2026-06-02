// app/composables/useHostBridgeAdapter.ts
import { initPowerXBridge } from "~/bridge/powerx-bridge-client";
import { applyPowerXHostAuthToken } from "@artisan-cloud/plugin-framework-client";
import { useI18n, useRuntimeConfig } from "#imports";
import { useTheme } from "~/composables/useTheme";
import { useAuth } from "~/composables/useAuth";
import { useHostCtxStore } from "~/stores/hostCtx";
import { useUserStore } from "~/stores/user";

type BridgeOptions = { pluginId?: string; instanceId?: string; debug?: boolean };

/** 将宿主广播适配到项目内现有的语言/主题切换实现 */
export function setupHostBridgeAdapter(opts: BridgeOptions = {}) {
  const { setLocale, locale } = useI18n();
  const { setTheme } = useTheme(); // ← 不再解构 currentTheme
  const runtimeConfig = useRuntimeConfig();
  const auth = useAuth();
  const hostCtxStore = useHostCtxStore();
  const userStore = useUserStore();

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
  const insidePowerX =
    runtimeConfig.public?.insidePowerX === true ||
    runtimeConfig.public?.insidePowerX === "true";

  if (shouldLog) {
    console.info("[Bridge][Plugin] debug mode enabled");
  }

  const applyAuthToken = async (payload: Record<string, any>) => {
    if (!insidePowerX) {
      if (shouldLog) {
        console.info("[Bridge][Plugin] ignore auth-token because insidePowerX=false");
      }
      return;
    }
    const result = await applyPowerXHostAuthToken({
      payload,
      pluginId: resolvedPluginId,
      storeHostCtx: (key, ctx) => hostCtxStore.setCtx(key, ctx),
      setAuth: (authPayload) => auth.setAuth(authPayload as any),
      fetchUserContext: (input) => userStore.fetchUserContext(input),
      validateIdentity: () => {
        if (!userStore.isRoot && !String(userStore.currentTenantUuid || "").trim()) {
          throw new Error("missing current tenant uuid");
        }
      },
    });
    if (shouldLog) {
      console.info("[Bridge][Plugin] applyAuthToken -> setAuth", {
        pluginId: resolvedPluginId,
        expiresIn: result.expiresIn,
        token: `${result.accessToken.slice(0, 4)}...${result.accessToken.slice(-4)}`,
      });
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
    onAuthToken: (p) => {
      applyAuthToken(p).catch((error) => {
        console.error("[Bridge][Plugin] applyAuthToken failed", error);
        userStore.clearUserState?.();
        auth.failClosed?.("PowerX 会话已失效，请回到宿主重新登录");
      });
    },
  });

  return { bridge };
}
