import type { PowerXNavigationRequest } from './bridge';

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
export function createPowerXRouterNavigator(router: PowerXRouterAdapter, baseURL: string, origin: string) {
  return async (path: string, request: PowerXNavigationRequest): Promise<void> => {
    if (request.signal.aborted) throw new Error('PLUGIN_NAVIGATION_ABORTED');
    const internal = resolvePowerXRouterPath(path, baseURL, origin);
    const dispose = router.beforeResolve(to => { if (to.fullPath === internal && request.signal.aborted) return false; });
    try {
      const failure = await router.push(internal);
      if (request.signal.aborted) throw new Error('PLUGIN_NAVIGATION_ABORTED');
      if (failure) throw new Error('PLUGIN_NAVIGATION_FAILED');
    } finally { dispose(); }
  };
}
