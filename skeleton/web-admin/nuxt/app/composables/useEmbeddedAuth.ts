import { readonly, ref } from 'vue';
import { useNuxtApp, useRuntimeConfig } from '#imports';
import { getPowerXHostSession, PowerXHostSessionError } from '@artisan-cloud/plugin-framework-client';

/** Embedded authentication has no persistence or refresh credential. */
export function useEmbeddedAuth() {
  const config = useRuntimeConfig();
  const app = useNuxtApp();
  const pluginId = String(config.public?.powerxPluginId || 'com.powerx.plugins.base');
  const session = typeof window === 'undefined' ? null : getPowerXHostSession(pluginId);
  const token = useState<string | null>('auth.token', () => null);
  const refreshToken = useState<null>('auth.refreshToken', () => null);
  const expiresAt = useState<number | null>('auth.expiresAt', () => null);
  const isAuthenticated = useState('auth.isAuthenticated', () => false);
  const user = useState('auth.user', () => null);
  const error = useState('auth.delegatedError', () => '');
  const initialized = useState('auth.hostInitialized', () => false);
  const delegatedIAM = useState('auth.delegatedIAM', () => config.public?.providerMode === 'delegated');
  const localIAMEnabled = useState('auth.localIAMEnabled', () => !delegatedIAM.value);
  const message = () => app.$i18n.t('hostSession.required');
  const rememberAuthError = (text?: string) => { error.value = text || message(); };
  const initAuth = () => {
    if (!session || initialized.value) return;
    initialized.value = true;
    const unsubscribe = session.subscribe(state => {
      const snapshot = session.getReadySnapshot();
      token.value = snapshot?.accessToken || null;
      expiresAt.value = snapshot?.expiresAt || null;
      refreshToken.value = null;
      isAuthenticated.value = Boolean(snapshot);
      if (snapshot) error.value = '';
      else if (state.error) rememberAuthError();
    });
    app.vueApp.onUnmount(() => { unsubscribe(); initialized.value = false; });
  };
  const ensureFreshToken = async () => {
    if (!session) return null;
    try { return (await session.ensureSnapshot()).accessToken; }
    catch (cause) { if (!session.getReadySnapshot()) rememberAuthError(); throw cause; }
  };
  const managed = () => { throw new PowerXHostSessionError('AUTH_SESSION_HOST_MANAGED'); };
  const clearAuth = () => { session?.invalidate('PLUGIN_AUTH_REQUIRED'); user.value = null; };
  return {
    isAuthenticated: readonly(isAuthenticated), user: readonly(user), token, refreshToken, expiresAt,
    getToken: () => session?.getReadySnapshot()?.accessToken || null,
    isTokenExpired: () => !session?.getReadySnapshot(), ensureFreshToken, initAuth,
    setAuth: managed, logout: managed, clearAuth,
    failClosed: (text?: string) => { session?.invalidate('PLUGIN_AUTH_CONTEXT_FAILED'); rememberAuthError(text); },
    rememberAuthError, consumeAuthError: () => error.value, delegatedError: readonly(error),
    clearDelegatedError: () => { error.value = ''; },
    restoreFromStorage: async () => {
      try { return (await session!.ensureSnapshot({ recovery: true })).accessToken; } catch (cause) { if (!session?.getReadySnapshot()) rememberAuthError(); throw cause; }
    },
    hostManagedAuth: readonly(ref(true)),
    localIAMEnabled: readonly(localIAMEnabled), delegatedIAM: readonly(delegatedIAM),
    setProviderModeFlags: (delegated: boolean) => { delegatedIAM.value = delegated; localIAMEnabled.value = !delegated; },
  };
}
