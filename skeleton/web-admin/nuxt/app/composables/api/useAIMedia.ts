import { onScopeDispose, watch } from 'vue';
import { createAIMediaClient, createPluginSSEClient, isPowerXEmbedded, type MediaTaskWatchOptions } from '@artisan-cloud/plugin-framework-client';
import { useApiClient } from './_client';
import { resolveApiBase, getAuthToken, getTenantUuid } from './_base';
import { useAuth } from '~/composables/useAuth';
import { useUserStore } from '~/stores/user';

// Core API keys and STS credentials stay in the plugin backend. The browser
// authenticates only to the plugin, with the current session in headers.
export function useAIMediaApi() {
  const config = useRuntimeConfig(), api = useApiClient(), auth = useAuth(), user = useUserStore();
  const active = new Set<AbortController>();
  const close = () => { for (const controller of active) controller.abort(); active.clear(); };
  watch(() => [auth.token.value, user.context?.current_tenant_uuid, user.userUuid], close);
  onScopeDispose(close);
  const client = createAIMediaClient({
    request: (path, options) => api.client(path, options),
    stream: options => createPluginSSEClient({
      pluginId: String(config.public.powerxPluginId), apiBaseURL: resolveApiBase(),
      insidePowerX: isPowerXEmbedded(config.public.insidePowerX),
      token: getAuthToken() || undefined, tenantUuid: getTenantUuid() || undefined,
    }).stream(options),
  });
  return {
    ...client,
    close,
    async watchTask(taskUUID: string, options: MediaTaskWatchOptions) {
      const controller = new AbortController();
      const abort = () => controller.abort(options.signal.reason);
      if (options.signal.aborted) abort();
      options.signal.addEventListener('abort', abort, { once: true });
      active.add(controller);
      try { await client.watchTask(taskUUID, { ...options, signal: controller.signal,
        onTask: task => { if (!controller.signal.aborted) options.onTask(task); },
        onState: state => { if (!controller.signal.aborted) options.onState?.(state); },
      }); }
      finally { controller.abort(); active.delete(controller); options.signal.removeEventListener('abort', abort); }
    },
  };
}
