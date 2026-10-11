import { useRuntimeConfig } from '#imports';
import { isPowerXEmbedded } from '@artisan-cloud/plugin-framework-client';
import { setupHostBridgeAdapter } from '~/composables/useHostBridgeAdapter';

export default defineNuxtPlugin(app => {
  const config = useRuntimeConfig();
  const auth = useAuth();
  auth.setProviderModeFlags(config.public?.providerMode === 'delegated');
  auth.initAuth();
  if (!isPowerXEmbedded(config.public?.insidePowerX)) {
    return;
  }
  try {
    const adapter = setupHostBridgeAdapter();
    app.vueApp.onUnmount(adapter.dispose);
    if (import.meta.hot) import.meta.hot.dispose(adapter.dispose);
  } catch {
    auth.failClosed(app.$i18n.t('hostSession.bridgeUnavailable'));
  }
});
