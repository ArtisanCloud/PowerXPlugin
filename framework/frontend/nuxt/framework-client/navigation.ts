import type { PowerXNavigationRequest } from './bridge';
import type { PowerXHostSession } from './host-session';

export interface PowerXRouterAdapter {
  push(path: string): Promise<unknown>;
  beforeResolve(guard: (to: { fullPath: string }) => boolean | void): () => void;
}
export function resolvePowerXRouterPath(path: string, baseURL: string, origin: string): string {
  if (!baseURL.startsWith('/') || !path.startsWith('/') || path.startsWith('//')) throw new Error('PLUGIN_NAVIGATION_INVALID');
  const target = new URL(path, origin);
  const mount = new URL(baseURL, origin).pathname.replace(/\/$/, '');
  if (target.origin !== origin || !mount || !(target.pathname === mount || target.pathname.startsWith(mount + '/'))) throw new Error('PLUGIN_NAVIGATION_INVALID');
  return (target.pathname.slice(mount.length) || '/') + target.search + target.hash;
}
export function createPowerXRouterNavigator(router: PowerXRouterAdapter, baseURL: string, origin: string, session?: PowerXHostSession) {
  return async (path: string, request: PowerXNavigationRequest): Promise<void> => {
    if (request.signal.aborted) throw new Error('PLUGIN_NAVIGATION_ABORTED');
    const snapshot = session?.getReadySnapshot();
    if (session && !snapshot) throw new Error('PLUGIN_AUTH_REQUIRED');
    const controller = new AbortController();
    const abort = () => controller.abort();
    if (request.signal.aborted) abort();
    const internal = resolvePowerXRouterPath(path, baseURL, origin);
    request.signal.addEventListener('abort', abort, { once: true });
    const unsubscribe = session?.subscribe(() => { if (!session.isCurrent(snapshot!)) abort(); });
    const dispose = router.beforeResolve(to => { if (to.fullPath === internal && controller.signal.aborted) return false; });
    try {
      if (controller.signal.aborted) throw new Error('PLUGIN_NAVIGATION_ABORTED');
      const failure = await router.push(internal);
      if (controller.signal.aborted) throw new Error('PLUGIN_NAVIGATION_ABORTED');
      if (failure) throw new Error('PLUGIN_NAVIGATION_FAILED');
    } finally { dispose(); unsubscribe?.(); request.signal.removeEventListener('abort', abort); }
  };
}
